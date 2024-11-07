package kafkadocker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
	"github.com/avast/retry-go"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/sync/errgroup"
)

// Cluster represents a Kafka cluster.
type Cluster struct {
	Brokers     int      // For specifying the number of brokers to start.
	Topics      []string // For specifying the topics to create.
	HealthProbe bool     // For specifying whether to health-probe the brokers after creation.
	Kraft       bool     // For specifying whether to use the Kafka Raft protocol.

	zookeeperContainer testcontainers.Container
	brokerContainers   []brokerContainer
	started            atomic.Bool
	network            *testcontainers.DockerNetwork
}

// BrokerAddresses returns the addresses of the brokers in the cluster.
func (c *Cluster) BrokerAddresses() []string {
	addrs := make([]string, len(c.brokerContainers))

	for i, b := range c.brokerContainers {
		addrs[i] = b.hostAddress
	}

	return addrs
}

type brokerContainer struct {
	testcontainers.Container
	hostAddress string
}

// Start creates the containers and the network for the cluster.
// nolint: gocognit, gocyclo
func (c *Cluster) Start(ctx context.Context) error {
	if c.started.Swap(true) {
		return ErrBrokerAlreadyStarted
	}

	kafkaNetwork, err := network.New(ctx)
	if err != nil {
		return fmt.Errorf("create network: %w", err)
	}

	c.network = kafkaNetwork

	if !c.Kraft {
		if err := c.startZookeeperCluster(ctx, kafkaNetwork.Name); err != nil {
			return fmt.Errorf("start zookeeper cluster: %w", err)
		}

		return nil
	}

	if err := c.startKraftCluster(ctx, kafkaNetwork.Name); err != nil {
		return fmt.Errorf("start kraft cluster: %w", err)
	}

	return nil
}

// Stop removes all the containers and the network concerning the cluster.
// nolint: gocognit, gocyclo
func (c *Cluster) Stop(ctx context.Context) error {
	if !c.started.Load() {
		return ErrBrokerWasNotStarted
	}

	eg, egCtx := errgroup.WithContext(ctx)

	if c.zookeeperContainer != nil {
		eg.Go(func() error {
			if err := c.zookeeperContainer.Terminate(egCtx); err != nil {
				return fmt.Errorf("terminate zookeeper container: %w", err)
			}

			return nil
		})
	}

	for i := range c.brokerContainers {
		eg.Go(func() error {
			rc, err := c.brokerContainers[i].Logs(ctx)
			if err != nil {
				return fmt.Errorf("get broker container %d logs: %w", i, err)
			}

			logs, err := io.ReadAll(rc)
			if err != nil {
				return fmt.Errorf("read broker container %d logs: %w", i, err)
			}

			log.Println(string(logs))

			if err := c.brokerContainers[i].Terminate(egCtx); err != nil {
				return fmt.Errorf("terminate broker container %d: %w", i, err)
			}

			return nil
		})
	}

	var errs error

	if err := eg.Wait(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("terminate containers: %w", err))
	}

	if c.network != nil {
		if err := c.network.Remove(ctx); err != nil {
			errs = errors.Join(errs, fmt.Errorf("remove network: %w", err))
		}
	}

	c.started.Store(false)

	return errs
}

func (c *Cluster) startKraftCluster(ctx context.Context, networkName string) error {
	const starterScriptName = "/testcontainers_start.sh"

	brokerControllerContainerReq := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:    "confluentinc/cp-kafka",
			Name:     "kafkadocker",
			Hostname: "kafkadocker",
			ExposedPorts: []string{
				"9092/tcp",
			},
			// nolint: revive // line too long
			Env: map[string]string{
				"KAFKA_PROCESS_ROLES":                        "controller,broker",
				"KAFKA_NODE_ID":                              "1",
				"CLUSTER_ID":                                 "h6EwvA-jRU6omVykKrSg1w",
				"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP":       "CONTROLLER:PLAINTEXT,BROKER:SASL_PLAINTEXT",
				"KAFKA_LISTENERS":                            "BROKER://kafkadocker:9092,CONTROLLER://kafkadocker:9093",
				"KAFKA_CONTROLLER_LISTENER_NAMES":            "CONTROLLER",
				"KAFKA_CONTROLLER_QUORUM_VOTERS":             "1@kafkadocker:9093",
				"KAFKA_INTER_BROKER_LISTENER_NAME":           "BROKER",
				"KAFKA_SASL_ENABLED_MECHANISMS":              "PLAIN",
				"KAFKA_SASL_MECHANISM_INTER_BROKER_PROTOCOL": "PLAIN",
				"KAFKA_OPTS":                                 "-Djava.security.auth.login.config=/etc/kafka/kafka_server_jaas.conf",
			},
			Networks: []string{networkName},
			NetworkAliases: map[string][]string{
				networkName: {"kafkadocker"},
			},
			WaitingFor: wait.ForLog("Kafka Server started"),
			HostConfigModifier: func(config *container.HostConfig) {
				config.RestartPolicy = container.RestartPolicy{Name: "unless-stopped"}
			},
			Cmd: []string{
				"sh",
				"-c",
				fmt.Sprintf(
					`while [ ! -f %[1]s ]; do sleep 0.1; done; %[1]s`,
					starterScriptName,
				),
			},
			LifecycleHooks: []testcontainers.ContainerLifecycleHooks{
				{
					PostStarts: []testcontainers.ContainerHook{
						func(ctx context.Context, container testcontainers.Container) error {
							var (
								advertisedListenerAddress = "BROKER://localhost:"
								startScript               string
							)

							const (
								retryAttempts = 5
								retryDelay    = time.Second / 10
							)

							if err := retry.Do(func() error {
								p, err := container.MappedPort(ctx, "9092/tcp")
								if err != nil {
									return fmt.Errorf("get hook mapped port: %w", err)
								}

								advertisedListenerAddress += p.Port()

								return nil
							}, retry.Attempts(retryAttempts), retry.Delay(retryDelay)); err != nil {
								return fmt.Errorf("get advertised listener address: %w", err)
							}

							startScriptReader, err := container.CopyFileFromContainer(
								ctx,
								"/etc/confluent/docker/run",
							)
							if err != nil {
								// nolint: revive // line too long
								return fmt.Errorf(
									"copy start script from container: %w",
									err,
								)
							}

							ss, err := io.ReadAll(startScriptReader)
							if err != nil {
								return fmt.Errorf("read start script: %w", err)
							}

							if err := startScriptReader.Close(); err != nil {
								return fmt.Errorf("close start script reader: %w", err)
							}

							startScript = string(ss)

							lastFiIdx := strings.LastIndex(startScript, "fi\n")

							startScript = startScript[:lastFiIdx+3] + fmt.Sprintf(
								"\nexport KAFKA_ADVERTISED_LISTENERS=%s;env\n",
								advertisedListenerAddress,
							) + startScript[lastFiIdx+3:]

							const fileMode = 0o755

							if err := container.CopyToContainer(
								ctx,
								[]byte(startScript),
								starterScriptName,
								fileMode,
							); err != nil {
								return fmt.Errorf("copy start script to container: %w", err)
							}

							return nil
						},
					},
				},
			},
		},
		Reuse:   true,
		Started: false,
	}

	brokContContainer, err := testcontainers.GenericContainer(ctx, brokerControllerContainerReq)
	if err != nil {
		return fmt.Errorf("create broker controller container: %w", err)
	}

	const fileMode755 = 0o755

	if err := brokContContainer.CopyToContainer(
		ctx,
		[]byte(`KafkaServer {
  org.apache.kafka.common.security.plain.PlainLoginModule required
  username="admin"
  password="admin-secret"
  user_admin="admin-secret"
  user_user1="user1-secret"
  user_user2="user2-secret";
};`),
		"/etc/kafka/kafka_server_jaas.conf",
		fileMode755,
	); err != nil {
		return fmt.Errorf("copy jaas config to container: %w", err)
	}

	if err := brokContContainer.Start(ctx); err != nil {
		return fmt.Errorf("start broker controller container: %w", err)
	}

	c.brokerContainers = []brokerContainer{{Container: brokContContainer}}

	for i, bc := range c.brokerContainers {
		containerIP, err := bc.Host(ctx)
		if err != nil {
			return fmt.Errorf("get broker container %d ip: %w", i, err)
		}

		port, err := bc.MappedPort(ctx, "9092/tcp")
		if err != nil {
			return fmt.Errorf("get broker container %d mapped port: %w", i, err)
		}

		c.brokerContainers[i].hostAddress = fmt.Sprintf("%s:%s", containerIP, port.Port())
	}

	return nil
}

func (c *Cluster) startZookeeperCluster(ctx context.Context, networkName string) error {
	zookeeperReq := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "confluentinc/cp-zookeeper",
			Name:  "kafkadocker-zookeeper",
			Env: map[string]string{
				"ZOOKEEPER_SERVER_ID":   "1",
				"ZOOKEEPER_CLIENT_PORT": "2181",
				"ZOOKEEPER_TICK_TIME":   "2000",
			},
			Networks: []string{networkName},
			NetworkAliases: map[string][]string{
				networkName: {"kafkadocker-zookeeper"},
			},
			WaitingFor: wait.ForLog("binding to port"),
			HostConfigModifier: func(config *container.HostConfig) {
				config.RestartPolicy = container.RestartPolicy{Name: "unless-stopped"}
			},
		},
		Reuse:   true,
		Started: true,
	}

	zookeeperContainer, err := testcontainers.GenericContainer(ctx, zookeeperReq)
	if err != nil {
		return fmt.Errorf("create zookeeper container: %w", err)
	}

	c.zookeeperContainer = zookeeperContainer

	brokers := 1

	if c.Brokers > 0 {
		brokers = c.Brokers
	}

	const starterScriptName = "/testcontainers_start.sh"

	brokerRequests := make(testcontainers.ParallelContainerRequest, brokers)

	for brokerID := 1; brokerID <= brokers; brokerID++ {
		containerName := fmt.Sprintf("kafkadocker-broker-%d", brokerID)
		port := fmt.Sprintf("909%d", brokerID)
		portTCP := port + "/tcp"

		brokerRequests[brokerID-1] = testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image: "confluentinc/cp-kafka",
				ExposedPorts: []string{
					portTCP,
				},
				// nolint: revive // line too long
				Env: map[string]string{
					"KAFKA_BROKER_ID":                        strconv.Itoa(brokerID),
					"KAFKA_ZOOKEEPER_CONNECT":                "kafkadocker-zookeeper:2181",
					"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP":   "INTERNAL:PLAINTEXT,EXTERNAL:PLAINTEXT",
					"KAFKA_INTER_BROKER_LISTENER_NAME":       "INTERNAL",
					"KAFKA_DELETE_TOPIC_ENABLE":              "true",
					"KAFKA_AUTO_CREATE_TOPICS_ENABLE":        "true",
					"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR": "1",
					"KAFKA_LISTENERS": fmt.Sprintf(
						"EXTERNAL://0.0.0.0:%[1]s,INTERNAL://0.0.0.0:2%[1]s",
						port,
					),
				},
				Networks: []string{networkName},
				NetworkAliases: map[string][]string{
					networkName: {containerName},
				},
				// WaitingFor: wait.ForLog("started (kafka.server.KafkaServer)"),
				Name: containerName,
				HostConfigModifier: func(config *container.HostConfig) {
					config.RestartPolicy = container.RestartPolicy{Name: "unless-stopped"}
				},
				Cmd: []string{
					"sh",
					"-c",
					fmt.Sprintf(
						`while [ ! -f %[1]s ]; do sleep 0.1; done; %[1]s`,
						starterScriptName,
					),
				},
				LifecycleHooks: []testcontainers.ContainerLifecycleHooks{
					{
						PostStarts: []testcontainers.ContainerHook{
							func(ctx context.Context, container testcontainers.Container) error {
								eg, egCtx := errgroup.WithContext(ctx)

								var (
									advertisedListenerAddress string
									startScript               string
								)

								eg.Go(func() error {
									const (
										retryAttempts = 5
										retryDelay    = time.Second / 10
									)

									return retry.Do(func() error {
										p, err := container.MappedPort(egCtx, nat.Port(portTCP))
										if err != nil {
											return fmt.Errorf("get hook mapped port: %w", err)
										}

										h, err := container.Host(egCtx)
										if err != nil {
											return fmt.Errorf("get hook host: %w", err)
										}

										// nolint: revive // line too long
										advertisedListenerAddress = fmt.Sprintf(
											"%s:%s",
											h,
											p.Port(),
										)

										return nil
									}, retry.Attempts(retryAttempts), retry.Delay(retryDelay))
								})

								eg.Go(func() error {
									startScriptReader, err := container.CopyFileFromContainer(
										egCtx,
										"/etc/confluent/docker/run",
									)
									if err != nil {
										// nolint: revive // line too long
										return fmt.Errorf(
											"copy start script from container: %w",
											err,
										)
									}

									ss, err := io.ReadAll(startScriptReader)
									if err != nil {
										return fmt.Errorf("read start script: %w", err)
									}

									if err := startScriptReader.Close(); err != nil {
										return fmt.Errorf("close start script reader: %w", err)
									}

									startScript = string(ss)

									return nil
								})

								if err := eg.Wait(); err != nil {
									return err
								}

								lastFiIdx := strings.LastIndex(startScript, "fi\n")

								advListeners := fmt.Sprintf(
									"EXTERNAL://%s,INTERNAL://%s:2%s",
									advertisedListenerAddress,
									containerName,
									port,
								)

								startScript = startScript[:lastFiIdx+3] + fmt.Sprintf(
									"\nexport KAFKA_ADVERTISED_LISTENERS=%s;env\n",
									advListeners,
								) + startScript[lastFiIdx+3:]

								const fileMode = 0o755

								if err := container.CopyToContainer(
									ctx,
									[]byte(startScript),
									starterScriptName,
									fileMode,
								); err != nil {
									return fmt.Errorf("copy start script to container: %w", err)
								}

								return nil
							},
						},
					},
				},
			},
			Reuse:   true,
			Started: true,
		}
	}

	bcs, err := testcontainers.ParallelContainers(
		ctx,
		brokerRequests,
		testcontainers.ParallelContainersOptions{},
	)

	c.brokerContainers = make([]brokerContainer, len(bcs))
	for i, bc := range bcs {
		c.brokerContainers[i] = brokerContainer{Container: bc}
	}

	if err != nil {
		return fmt.Errorf("create broker containers: %w", err)
	}

	for i, bc := range c.brokerContainers {
		containerIP, err := bc.Host(ctx)
		if err != nil {
			return fmt.Errorf("get broker container %d ip: %w", i, err)
		}

		name, err := bc.Name(ctx)
		if err != nil {
			return fmt.Errorf("get broker container %d name: %w", i, err)
		}

		idx := strings.Split(name, "-")[2]

		port, err := bc.MappedPort(ctx, nat.Port(fmt.Sprintf("909%s/tcp", idx)))
		if err != nil {
			return fmt.Errorf("get broker container %d mapped port: %w", i, err)
		}

		c.brokerContainers[i].hostAddress = fmt.Sprintf("%s:%s", containerIP, port.Port())
	}

	if c.HealthProbe {
		eg, egCtx := errgroup.WithContext(ctx)

		for i := range c.brokerContainers {
			eg.Go(func() error {
				return probeBroker(egCtx, c.brokerContainers[i])
			})
		}

		if err := eg.Wait(); err != nil {
			return fmt.Errorf("probe brokers: %w", err)
		}
	}

	return nil
}

func probeBroker(ctx context.Context, c brokerContainer) error {
	return retry.Do(func() error {
		brk := sarama.NewBroker(c.hostAddress)

		if err := brk.Open(nil); err != nil {
			return fmt.Errorf("open: %w", err)
		}

		// nolint: errcheck
		defer brk.Close()

		conn, err := brk.Connected()
		if err != nil {
			return fmt.Errorf("connected: %w", err)
		}

		if !conn {
			return ErrBrokerNotConnected
		}

		if _, err = brk.Heartbeat(&sarama.HeartbeatRequest{}); err != nil {
			return fmt.Errorf("heartbeat: %w", err)
		}

		if err := brk.Close(); err != nil {
			return fmt.Errorf("close: %w", err)
		}

		return nil
	}, retry.Context(ctx))
}

package kafkadocker

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
	"github.com/avast/retry-go"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/sync/errgroup"
)

// Cluster represents a Kafka cluster.
type Cluster struct {
	zookeeperContainer testcontainers.Container
	brokerContainers   []brokerContainer
	started            atomic.Bool
	network            testcontainers.Network

	Brokers     int      // For specifying the number of brokers to start.
	Topics      []string // For specifying the topics to create.
	HealthProbe bool
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

	const networkName = "kafkadocker"

	network, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: testcontainers.NetworkRequest{
			Name:           networkName,
			CheckDuplicate: true,
		},
	})
	if err != nil {
		return fmt.Errorf("create network: %w", err)
	}

	c.network = network

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
										advertisedListenerAddress = fmt.Sprintf("%s:%s", h, p.Port())

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
										return fmt.Errorf("copy start script from container: %w", err)
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
									"\necho wtf;export KAFKA_ADVERTISED_LISTENERS=%s;env\n",
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
			i := i

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

// Stop removes all the containers and the network concerning the cluster.
// nolint: gocognit, gocyclo
func (c *Cluster) Stop(ctx context.Context) error {
	if !c.started.Load() {
		return ErrBrokerWasNotStarted
	}

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		if c.zookeeperContainer != nil {
			if err := c.zookeeperContainer.Terminate(egCtx); err != nil {
				return fmt.Errorf("terminate zookeeper container: %w", err)
			}
		}

		return nil
	})

	for i := range c.brokerContainers {
		i := i

		eg.Go(func() error {
			if err := c.brokerContainers[i].Terminate(egCtx); err != nil {
				return fmt.Errorf("terminate broker container %d: %w", i, err)
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	if c.network != nil {
		if err := c.network.Remove(ctx); err != nil {
			return fmt.Errorf("remove network: %w", err)
		}
	}

	c.started.Store(false)

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

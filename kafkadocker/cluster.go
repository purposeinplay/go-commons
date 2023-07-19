package kafkadocker

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/sync/errgroup"
	"strconv"
	"strings"
	"sync/atomic"
)

type Cluster struct {
	zookeeperContainer testcontainers.Container
	brokerContainers   []brokerContainer
	started            atomic.Bool
	network            testcontainers.Network

	Brokers int // For specifying the number of brokers to start.
}

func (c *Cluster) BrokerAddresses() []string {
	addrs := make([]string, len(c.brokerContainers))

	for i, b := range c.brokerContainers {
		addrs[i] = b.HostAddress()
	}

	return addrs
}

type brokerContainer struct {
	testcontainers.Container
	hostAddress string
}

func (b brokerContainer) HostAddress() string {
	return b.hostAddress
}

func (c *Cluster) Start(ctx context.Context) error {
	if c.started.Swap(true) {
		return ErrBrokerAlreadyStarted
	}

	const networkName = "kafkadocker"

	network, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: testcontainers.NetworkRequest{
			Name: networkName,
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

	brokerRequests := make(testcontainers.ParallelContainerRequest, brokers)

	for brokerID := 1; brokerID <= brokers; brokerID++ {
		containerName := fmt.Sprintf("kafkadocker-broker-%d", brokerID)
		port := fmt.Sprintf("909%d", brokerID)

		brokerRequests[brokerID-1] = testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image: "confluentinc/cp-kafka",
				ExposedPorts: []string{
					port + "/tcp",
				},
				Env: map[string]string{
					"KAFKA_BROKER_ID": strconv.Itoa(brokerID),
					// "KAFKA_ZOOKEEPER_CONNECT": fmt.Sprintf("%s:2181", zookeeperContainerIP),
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
					"KAFKA_ADVERTISED_LISTENERS": fmt.Sprintf(
						"EXTERNAL://localhost:%[1]s,INTERNAL://%s:2%[1]s",
						port,
						containerName,
					),
				},
				Networks: []string{networkName},
				NetworkAliases: map[string][]string{
					networkName: {containerName},
				},
				WaitingFor: wait.ForLog("started (kafka.server.KafkaServer)"),
				Name:       containerName,
				HostConfigModifier: func(config *container.HostConfig) {
					config.RestartPolicy = container.RestartPolicy{Name: "unless-stopped"}
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
		ip, err := bc.Host(ctx)
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

		c.brokerContainers[i].hostAddress = fmt.Sprintf("%s:%s", ip, port.Port())
	}

	return nil
}

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

			fmt.Println("closed broker container", i)

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

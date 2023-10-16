package kafkasarama

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/xdg-go/scram"
)

// NewSASLSubscriberConfig creates a new kafka subscriber config with SASL authentication.
func NewSASLSubscriberConfig(username, password string) *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Consumer.Return.Errors = true

	return saslConfig(cfg, username, password)
}

// NewSASLPublisherConfig creates a new kafka publisher config with SASL authentication.
func NewSASLPublisherConfig(username, password string) *sarama.Config {
	cfg := sarama.NewConfig()

	cfg.Producer.Retry.Max = 10
	cfg.Producer.Return.Successes = true
	cfg.Metadata.Retry.Backoff = time.Second * 2

	return saslConfig(cfg, username, password)
}

// saslConfig configures the sarama config for SASL authentication.
func saslConfig(cfg *sarama.Config, username, password string) *sarama.Config {
	cfg.Net.TLS.Enable = true
	cfg.Net.TLS.Config = &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS13,
	}

	cfg.Version = sarama.V3_3_1_0

	cfg.Net.SASL.Enable = true
	cfg.Net.SASL.Handshake = true
	cfg.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
	cfg.Net.SASL.User = username
	cfg.Net.SASL.Version = sarama.SASLHandshakeV1
	cfg.Net.SASL.Password = password
	cfg.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
		return &xdgScramClient{
			HashGeneratorFcn: scram.SHA512,
		}
	}

	return cfg
}

var _ sarama.SCRAMClient = (*xdgScramClient)(nil)

type xdgScramClient struct {
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

func (c *xdgScramClient) Begin(username, password, authzID string) error {
	client, err := c.HashGeneratorFcn.NewClient(username, password, authzID)
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}

	c.ClientConversation = client.NewConversation()

	return nil
}

func (c *xdgScramClient) Step(challenge string) (string, error) {
	return c.ClientConversation.Step(challenge)
}

func (c *xdgScramClient) Done() bool {
	return c.ClientConversation.Done()
}

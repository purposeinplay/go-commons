package kafkashopifysarama

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/Shopify/sarama"
)

// NewTLSSubscriberConfig creates a new kafka subscriber config with TLS
// authentication.
func NewTLSSubscriberConfig(
	cerfFilePath, keyFilePath, caFilePath string,
) (*sarama.Config, error) {
	cfg := sarama.NewConfig()
	cfg.Consumer.Return.Errors = true

	return loadTLSConfig(cfg, cerfFilePath, keyFilePath, caFilePath)
}

// NewTLSPublisherConfig creates a new kafka publisher config with TLS
// authentication.
func NewTLSPublisherConfig(
	cerfFilePath, keyFilePath, caFilePath string,
) (*sarama.Config, error) {
	cfg := sarama.NewConfig()

	cfg.Producer.Retry.Max = 10
	cfg.Producer.Return.Successes = true
	cfg.Metadata.Retry.Backoff = time.Second * 2

	return loadTLSConfig(cfg, cerfFilePath, keyFilePath, caFilePath)
}

func loadTLSConfig(
	cfg *sarama.Config,
	cerfFilePath, keyFilePath, caFilePath string,
) (*sarama.Config, error) {
	// Load client cert
	cert, err := tls.LoadX509KeyPair(
		cerfFilePath,
		keyFilePath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load client cert: %w", err)
	}

	// Load CA cert
	//nolint: gosec // The file is set by the server, should be safe.
	caCert, err := os.ReadFile(caFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA cert: %w", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: false,
	}

	cfg.Net.TLS.Enable = true
	cfg.Net.TLS.Config = tlsCfg

	return cfg, nil
}

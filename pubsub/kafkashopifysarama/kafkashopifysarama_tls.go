package kafkashopifysarama

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/Shopify/sarama"
)

// NewTLSSubscriberConfig creates a new kafka subscriber config with TLS
// authentication.
func NewTLSSubscriberConfig(tlsCfg *tls.Config) *sarama.Config {
	cfg := sarama.NewConfig()

	cfg.Net.TLS.Enable = true
	cfg.Net.TLS.Config = tlsCfg

	return cfg
}

// LoadTLSConfig loads the TLS config from the given folder.
func LoadTLSConfig(cerfFilePath, keyFilePath, caFilePath string) (*tls.Config, error) {
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

	return tlsCfg, nil
}

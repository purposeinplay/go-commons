package psqlutil

import (
	"fmt"
)

// ConnectionConfig is a PostgreSQL connection configuration.
type ConnectionConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DSN returns a PostgreSQL Data Source Name.
func (c ConnectionConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s "+
			"user=%s "+
			"password=%s "+
			"dbname=%s "+
			"port=%s "+
			"sslmode=%s",
		c.Host,
		c.User,
		c.Password,
		c.DBName,
		c.Port,
		c.SSLMode,
	)
}

// ComposeDSN returns a PostgreSQL Data Source Name.
//
// Deprecated: Use github.com/purposeinplay/go-commons/psqlutil.ConnectionConfig instead.
func ComposeDSN(
	host,
	port,
	user,
	password,
	dbName,
	sslMode string,
) string {
	return fmt.Sprintf(
		"host=%s "+
			"user=%s "+
			"password=%s "+
			"dbname=%s "+
			"port=%s "+
			"sslmode=%s",
		host,
		user,
		password,
		dbName,
		port,
		sslMode,
	)
}

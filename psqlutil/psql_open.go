package psqlutil

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/avast/retry-go"
	// import for init function.
	_ "github.com/jackc/pgx/v4/stdlib"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
)

// GormOpen opens a new db connection
// and returns a *gorm.DB.
func GormOpen(
	ctx context.Context,
	zapLogger *zap.Logger,
	postgresDSN string,
) (*gorm.DB, error) {
	const (
		connectionAttempts = 5
		delaySeconds       = 2
	)

	var db *gorm.DB

	err := retry.Do(func() error {
		var err error

		db, err = gorm.Open(
			postgres.New(postgres.Config{
				DriverName: "postgres",
				DSN:        postgresDSN,
			}),
			&gorm.Config{
				SkipDefaultTransaction: true,
				Logger:                 zapgorm2.New(zapLogger),
			})

		return err
	},
		retry.Attempts(connectionAttempts),
		retry.Delay(delaySeconds*time.Second),
		retry.Context(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	return db, nil
}

// SQLOpenWithGormDriver opens a new db connection and returns a
// *sql.DB with a gorm driver.
func SQLOpenWithGormDriver(
	ctx context.Context,
	postgresDSN string,
) (*sql.DB, error) {
	return sqlOpen(ctx, "postgres", postgresDSN)
}

// SQLOpenWithPgxDriver opens a new db connection with a pgx driver.
func SQLOpenWithPgxDriver(
	ctx context.Context,
	postgresDSN string,
) (*sql.DB, error) {
	return sqlOpen(ctx, "pgx", postgresDSN)
}

// MustNewSQL panics if err is not nil, otherwise it returns db.
func MustNewSQL(db *sql.DB, err error) *sql.DB {
	if err != nil {
		panic(err)
	}

	return db
}

func sqlOpen(
	ctx context.Context,
	driver,
	postgresDSN string,
) (*sql.DB, error) {
	const (
		connectionAttempts = 5
		delaySeconds       = 2
	)

	var db *sql.DB

	err := retry.Do(func() error {
		var err error

		db, err = sql.Open(driver, postgresDSN)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}

		err = db.Ping()
		if err != nil {
			return fmt.Errorf("ping: %w", err)
		}

		return err
	},
		retry.Attempts(connectionAttempts),
		retry.Delay(delaySeconds*time.Second),
		retry.Context(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	return db, nil
}

package psqlutil

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/avast/retry-go"
	// import for init function.
	_ "github.com/jackc/pgx/v5/stdlib"
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
	ignoreRecordNotFoundErr bool,
	errorPlugin GORMErrorsPlugin,
) (*gorm.DB, error) {
	const (
		connectionAttempts = 5
		delaySeconds       = 2
	)

	var db *gorm.DB

	logger := zapgorm2.New(zapLogger)
	logger.IgnoreRecordNotFoundError = ignoreRecordNotFoundErr

	err := retry.Do(func() error {
		var err error

		db, err = gorm.Open(
			postgres.Open(postgresDSN),
			&gorm.Config{
				SkipDefaultTransaction: true,
				Logger:                 logger,
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

	if err := db.Use(errorPlugin); err != nil {
		return nil, fmt.Errorf("use gorm errors plugin: %w", err)
	}

	return db, nil
}

// MustNewSQL panics if err is not nil, otherwise it returns db.
func MustNewSQL(db *sql.DB, err error) *sql.DB {
	if err != nil {
		panic(err)
	}

	return db
}

// SQLOpen opens a new db connection and returns a
// *sql.DB with a gorm driver.
func SQLOpen(
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

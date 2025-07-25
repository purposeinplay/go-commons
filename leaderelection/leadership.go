package leaderelection

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// LeaderElection is a struct that manages leader election in a distributed
// system.
type LeaderElection struct {
	logger              *slog.Logger
	lockID              int64
	leaderCheckInterval time.Duration
	dsn                 string
	lockConn            *gorm.DB
	isLeader            bool
	leadershipLost      chan struct{}
	contextCancelled    chan struct{}
}

// New creates a new LeaderElection instance with the provided parameters.
func New(
	logger *slog.Logger,
	lockID int64,
	leaderCheckInterval time.Duration,
	dsn string,
) (*LeaderElection, error) {
	if dsn == "" {
		return nil, ErrEmptyDSN
	}

	return &LeaderElection{
		logger:              logger,
		lockID:              lockID,
		leaderCheckInterval: leaderCheckInterval,
		dsn:                 dsn,
		lockConn:            nil,
		isLeader:            false,
	}, nil
}

// IsLeader returns true if the current instance is the leader.
func (le *LeaderElection) IsLeader(ctx context.Context) (bool, error) {
	// Reuse existing connection if available and usable, otherwise create a new one
	if !le.isConnectionUsable() {
		if err := le.createLockConnection(); err != nil {
			return false, fmt.Errorf("create lock connection: %w", err)
		}
	}

	// Try to acquire leadership using PostgreSQL advisory lock
	isLeader, err := le.tryAcquireLeadership(ctx)
	if err != nil {
		return false, fmt.Errorf("leader election failed: %w", err)
	}

	if !isLeader {
		return false, nil
	}

	le.isLeader = true
	le.logger.InfoContext(ctx, "elected as leader")

	// Create a channel to signal leadership loss
	le.leadershipLost = make(chan struct{})
	// Create a channel to signal context cancellation
	le.contextCancelled = make(chan struct{})

	// Start health check to detect leadership loss
	go le.leaderHealthCheck(ctx)

	return true, nil
}

// LeadershipLost returns a channel that is closed when leadership is lost.
func (le *LeaderElection) LeadershipLost() <-chan struct{} {
	return le.leadershipLost
}

// ContextCancelled returns a channel that is closed when the context is
// cancelled.
func (le *LeaderElection) ContextCancelled() <-chan struct{} {
	return le.contextCancelled
}

// Close releases the advisory lock and closes the lock connection.
func (le *LeaderElection) Close() error {
	if le.isConnectionUsable() {
		// Close the dedicated lock connection
		sqlDB, err := le.lockConn.DB()
		if err != nil {
			return fmt.Errorf("get lock sql db: %w", err)
		}

		if err := sqlDB.Close(); err != nil {
			return fmt.Errorf("close lock connection: %w", err)
		}
	}

	return nil
}

// isConnectionUsable checks if the database connection exists and is usable.
func (le *LeaderElection) isConnectionUsable() bool {
	if le.lockConn == nil {
		return false
	}

	var result int

	err := le.lockConn.Raw("SELECT 1").Scan(&result).Error

	return err == nil
}

// createLockConnection creates a dedicated connection for advisory lock
// operations.
func (le *LeaderElection) createLockConnection() error {
	// Create a dedicated connection for advisory lock operations. This ensures
	// the same connection is used for all lock operations.
	//
	// "Every connection in a pool is treated by Postgres as a separate
	// connection and will have a different backend PID. You must ensure that
	// the same connection is used for checking the leader status every time."
	// https://ramitmittal.com/blog/general/leader-election-advisory-locks
	lockConn, err := gorm.Open(
		postgres.Open(le.dsn),
		&gorm.Config{},
	)
	if err != nil {
		return fmt.Errorf("create lock connection: %w", err)
	}

	// Configure the lock connection to use only 1 connection to ensure connection reuse
	sqlDB, err := lockConn.DB()
	if err != nil {
		return fmt.Errorf("get lock sql db: %w", err)
	}

	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0) // Never expire connections

	le.lockConn = lockConn

	return nil
}

// tryAcquireLeadership attempts to acquire the PostgreSQL advisory lock for
// leadership.
func (le *LeaderElection) tryAcquireLeadership(
	ctx context.Context,
) (bool, error) {
	var acquired bool

	err := le.lockConn.WithContext(ctx).
		Raw("SELECT pg_try_advisory_lock(?)", le.lockID).
		Scan(&acquired).
		Error
	if err != nil {
		return false, fmt.Errorf("try advisory lock: %w", err)
	}

	return acquired, nil
}

// leaderHealthCheck periodically verifies that we still hold the leadership
// lock. If leadership is lost, it signals via the leadershipLost channel. This
// could happen if the database goes down for example, and this avoid holding
// the lock indefinitely.
func (le *LeaderElection) leaderHealthCheck(
	ctx context.Context,
) {
	ticker := time.NewTicker(le.leaderCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context cancelled - signal shutdown
			close(le.contextCancelled)
			return
		case <-ticker.C:
			if !le.isLeader {
				return // No longer the leader
			}

			// Check if we still hold the lock
			stillLeader := le.checkLeadershipStatus(ctx)
			if !stillLeader {
				le.logger.WarnContext(ctx, "lost leadership")

				le.isLeader = false

				// Signal leadership loss
				close(le.leadershipLost)

				return
			}
		}
	}
}

// checkLeadershipStatus checks if we still hold the advisory lock.
func (le *LeaderElection) checkLeadershipStatus(ctx context.Context) bool {
	// Use a simple query to check if the connection is still alive
	// If the connection is lost, we'll get an error
	var result int

	err := le.lockConn.WithContext(ctx).
		Raw("SELECT 1").
		Scan(&result).
		Error
	if err != nil {
		le.logger.WarnContext(
			ctx,
			"checkLeadershipStatus connection lost",
			"error",
			err,
		)

		return false // Connection lost
	}

	// Since we're using the same connection that acquired the lock,
	// and PostgreSQL advisory locks are tied to the connection,
	// if the connection is alive, we still hold the lock.
	// The lock will be automatically released when the connection is closed.
	//
	// This approach is safer than trying to re-acquire the lock because:
	// 1. It avoids the risk of increasing the lock count
	// 2. It prevents potential deadlocks from failed release operations
	// 3. It's more efficient (simple SELECT 1 vs advisory lock operation)
	return true
}

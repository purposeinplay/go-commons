package leaderelection

import (
	"log/slog"
	"testing"
	"time"

	"github.com/matryer/is"
)

const leaderCheckInterval = 10 * time.Millisecond

func createLeaderElection(i *is.I, lockID int64) *LeaderElection {
	i.Helper()

	logger := slog.Default()

	leaderElection, err := New(
		logger,
		lockID,
		leaderCheckInterval,
		TestDSN,
	)
	if err != nil {
		i.Fail() // failed to create leader election
	}

	if err := leaderElection.createLockConnection(); err != nil {
		i.Fail() // failed to create connection
	}

	return leaderElection
}

// TestLockConnBackendPID_Stability asserts that the backend PID never changes
// for a Subscriber during its lifetime.
func TestLockConnBackendPID_Stability(t *testing.T) { // nolint: wsl
	t.Parallel()

	i := is.New(t)
	ctx := t.Context()

	le := createLeaderElection(i, 300)

	// Create the connection by calling tryAcquireLeadership first
	_, err := le.tryAcquireLeadership(ctx)
	i.NoErr(err)

	// Get the initial backend PID
	var initialPID int
	err = le.lockConn.WithContext(ctx).
		Raw("SELECT pg_backend_pid()").
		Scan(&initialPID).
		Error
	i.NoErr(err)
	i.True(initialPID > 0)

	// Perform several advisory lock operations and check the PID each time
	for range 5 {
		_, err := le.tryAcquireLeadership(ctx)
		i.NoErr(err)

		var pid int
		err = le.lockConn.WithContext(ctx).
			Raw("SELECT pg_backend_pid()").
			Scan(&pid).
			Error
		i.NoErr(err)
		i.Equal(pid, initialPID)

		err = le.lockConn.WithContext(ctx).
			Raw("SELECT pg_backend_pid()").
			Scan(&pid).
			Error
		i.NoErr(err)
		i.Equal(pid, initialPID)
	}

	if err := le.Close(); err != nil {
		i.Fail() // failed to close subscriber
	}
}

// TestRun_DatabaseRecovery ensures a new leader is elected after DB goes down and comes back up.
func TestRun_DatabaseRecovery(t *testing.T) { // nolint: wsl
	t.Parallel()

	i := is.New(t)
	ctx := t.Context()

	le1 := createLeaderElection(i, 400)
	le2 := createLeaderElection(i, 400)

	isLeader1, err := le1.IsLeader(ctx)
	i.NoErr(err)
	isLeader2, err := le2.IsLeader(ctx)
	i.NoErr(err)

	// Verify only one leader exists
	i.True(isLeader1)
	i.True(!isLeader2)

	// Simulate database crash by forcefully closing the underlying connection
	// This simulates what happens when the database goes down unexpectedly
	if le1.lockConn != nil {
		sqlDB, err := le1.lockConn.DB()
		i.NoErr(err)

		sqlDB.Close()
	}

	// Wait for health check to detect the connection loss
	time.Sleep(2 * leaderCheckInterval)

	// Verify the other subscriber becomes leader after detecting the crash
	isLeader2, err = le2.IsLeader(ctx)
	i.NoErr(err)
	i.True(isLeader2)
}

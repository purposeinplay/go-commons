package leaderelection_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/leaderelection"
)

const leaderCheckInterval = 10 * time.Millisecond

func createLeaderElection(
	i *is.I,
	lockID int64,
) *leaderelection.LeaderElection {
	i.Helper()

	logger := slog.Default()

	leaderElection, err := leaderelection.New(
		logger,
		lockID,
		leaderCheckInterval,
		leaderelection.TestDSN,
	)
	if err != nil {
		i.Fail() // failed to create leader election
	}

	return leaderElection
}

// TestRun_ElectsOneLeader ensures only one leader is elected.
func TestRun_ElectsOneLeader(t *testing.T) { // nolint: wsl
	t.Parallel()

	i := is.New(t)
	ctx := t.Context()

	le1 := createLeaderElection(i, 1000)
	le2 := createLeaderElection(i, 1000)

	isLeader1, err := le1.IsLeader(ctx)
	i.NoErr(err)
	isLeader2, err := le2.IsLeader(ctx)
	i.NoErr(err)

	i.True(isLeader1)
	i.True(!isLeader2)
}

// TestRun_LeaderFailover ensures a new leader is elected if the current leader
// goes down.
func TestRun_LeaderFailover(t *testing.T) { // nolint: wsl
	t.Parallel()

	i := is.New(t)
	ctx := t.Context()

	le1 := createLeaderElection(i, 2000)
	le2 := createLeaderElection(i, 2000)

	isLeader1, err := le1.IsLeader(ctx)
	i.NoErr(err)
	isLeader2, err := le2.IsLeader(ctx)
	i.NoErr(err)

	// Verify only one leader exists
	i.True(isLeader1)
	i.True(!isLeader2)

	// Start monitoring for leadership loss signal
	leadershipLost := make(chan struct{})
	monitorLeadership := func() {
		select {
		case <-le1.LeadershipLost():
			close(leadershipLost)
		case <-time.After(5 * time.Second):
			t.Error("timeout waiting for leadership loss signal")
		}
	}

	go monitorLeadership()

	// Simulate leader going down by closing it
	le1.Close()

	// Wait for leadership loss to be detected
	select {
	case <-leadershipLost:
		// Leadership loss was properly detected
	case <-time.After(3 * time.Second):
		i.Fail() // timeout waiting for leadership loss
	}

	// Wait for new leader to be elected
	time.Sleep(2 * leaderCheckInterval)

	// Verify the other subscriber became leader
	isLeader2, err = le2.IsLeader(ctx)
	i.NoErr(err)
	i.True(isLeader2)
}

// TestRun_ContextCancellationDuringLeadership ensures graceful shutdown when
// context is cancelled while a subscriber is the leader.
func TestRun_ContextCancellationDuringLeadership(t *testing.T) { // nolint: wsl
	t.Parallel()

	i := is.New(t)
	ctxToCancel, cancel := context.WithCancel(t.Context())

	le1 := createLeaderElection(i, 3000)
	le2 := createLeaderElection(i, 3000)

	isLeader1, err := le1.IsLeader(ctxToCancel)
	i.NoErr(err)
	isLeader2, err := le2.IsLeader(t.Context())
	i.NoErr(err)

	i.True(isLeader1)
	i.True(!isLeader2)

	// Start monitoring for context cancellation signal
	contextCancelled := make(chan struct{})
	monitorContext := func() {
		select {
		case <-le1.ContextCancelled():
			close(contextCancelled)
		case <-time.After(5 * time.Second):
			t.Error("timeout waiting for context cancellation signal")
		}
	}

	go monitorContext()

	// Cancel context while leader is active
	cancel()

	// Wait for context cancellation to be detected
	select {
	case <-contextCancelled:
		// Context cancellation was properly detected
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for context cancellation")
	}

	// Now verify that le2 can become leader after le1's context is cancelled
	// Since both le1 and le2 share the same connection pool, we need to ensure
	// that le1's connection is properly closed and the lock is released.

	// Close le1 to release the lock
	le1.Close()

	// Wait for the connection to be properly closed
	time.Sleep(2 * leaderCheckInterval)

	// Now le2 should be able to become leader
	isLeader2, err = le2.IsLeader(t.Context())
	i.NoErr(err)
	i.True(isLeader2)
}

package grpcutils

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// ErrServiceUnavailable is returned whenever a service fails the health check.
var ErrServiceUnavailable = errors.New("service unavailable")

// CheckHealth verifies if a service implementing the Check RPC
// is available.
func CheckHealth(ctx context.Context, cc grpc.ClientConnInterface) error {
	resp, err := grpc_health_v1.NewHealthClient(cc).Check(
		ctx,
		&grpc_health_v1.HealthCheckRequest{},
	)
	if err != nil {
		return fmt.Errorf("check wallee service: %w", err)
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf(
			"received unexpected status %q: %w",
			resp.Status,
			ErrServiceUnavailable,
		)
	}

	return nil
}

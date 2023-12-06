package errorsgrpc

import (
	"context"

	"github.com/purposeinplay/go-commons/errors"
	commonserr "github.com/purposeinplay/go-commons/errors/proto/commons/error/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnmarshalErrorUnaryClientInterceptor returns a UnaryClientInterceptor that
// translates grpc error responses to errors.Error.
func UnmarshalErrorUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err == nil {
			return nil
		}

		sts, ok := status.FromError(err)
		if !ok {
			return err
		}

		if len(sts.Details()) != 1 {
			return err
		}

		errResp, ok := sts.Details()[0].(*commonserr.ErrorResponse)
		if !ok {
			return err
		}

		return &errors.Error{
			Type:    errors.ErrorType(sts.Message()),
			Code:    errors.ErrorCode(errResp.ErrorCode),
			Details: errResp.Message,
		}
	}
}

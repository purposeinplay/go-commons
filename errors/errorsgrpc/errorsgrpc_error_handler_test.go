package errorsgrpc_test

import (
	"strconv"
	"testing"

	// nolint: staticcheck
	"github.com/golang/protobuf/proto"
	"github.com/purposeinplay/go-commons/errors"
	"github.com/purposeinplay/go-commons/errors/errorsgrpc"
	commonserr "github.com/purposeinplay/go-commons/errors/proto/commons/error/v1"
	"github.com/stretchr/testify/require"
)

func TestRetrieveDetails(t *testing.T) {
	req := require.New(t)

	appErr := &errors.Error{
		Type:    errors.ErrorTypeInvalid,
		Code:    errors.ErrorCode(strconv.Itoa(int(rune(1)))),
		Message: "message",
	}

	sts, err := (&errorsgrpc.PanicErrorHandler{}).ErrorToGRPCStatus(appErr)
	req.NoError(err)

	req.Len(sts.Details(), 1)

	details, ok := sts.Details()[0].(*commonserr.ErrorResponse)
	req.True(ok)

	req.True(
		proto.Equal(
			&commonserr.ErrorResponse{
				ErrorCode: "1",
				Message:   "message",
			},
			details,
		),
	)
}

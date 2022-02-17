package grpc_test

import (
	"github.com/purposeinplay/go-commons/grpc"
	"testing"
)

func TestServer(t *testing.T) {
	s := grpc.NewServer()

	s.Stop()
}

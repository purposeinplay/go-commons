package grpc

import (
	"reflect"

	"google.golang.org/grpc"
	"slices"
)

func prependServerOptions(
	newInterceptor grpc.UnaryServerInterceptor,
	interceptors []grpc.UnaryServerInterceptor,
) []grpc.UnaryServerInterceptor {
	return slices.Insert(interceptors, 0, newInterceptor)
}

func isErrorHandlerNil(errorHandler ErrorHandler) bool {
	c := errorHandler

	return c == nil ||
		(reflect.ValueOf(c).Kind() == reflect.Ptr &&
			reflect.ValueOf(c).IsNil())
}

func isPanicHandlerNil(panicHandler PanicHandler) bool {
	c := panicHandler

	return c == nil ||
		(reflect.ValueOf(c).Kind() == reflect.Ptr &&
			reflect.ValueOf(c).IsNil())
}

func isMonitorOperationerNil(monitorOperationer MonitorOperationer) bool {
	c := monitorOperationer

	return c == nil ||
		(reflect.ValueOf(c).Kind() == reflect.Ptr &&
			reflect.ValueOf(c).IsNil())
}

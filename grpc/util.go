package grpc

import (
	"reflect"

	"google.golang.org/grpc"
)

func prependServerOption(
	newInterceptor grpc.UnaryServerInterceptor,
	interceptors []grpc.UnaryServerInterceptor,
) []grpc.UnaryServerInterceptor {
	newInterceptors := make(
		[]grpc.UnaryServerInterceptor,
		len(interceptors)+1,
	)

	copy(newInterceptors[1:], interceptors)

	newInterceptors[0] = newInterceptor

	return newInterceptors
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

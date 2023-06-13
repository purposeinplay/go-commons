package mock

import (
	"context"
	"sync"

	commonsgrpc "github.com/purposeinplay/go-commons/grpc"
	"google.golang.org/grpc/status"
)

// Ensure, that PanicHandlerMock does implement grpc.PanicHandler.
// If this is not the case, regenerate this file with moq.
var _ commonsgrpc.PanicHandler = &PanicHandlerMock{}

// PanicHandlerMock is a mock implementation of grpc.PanicHandler.
//
//	func TestSomethingThatUsesPanicHandler(t *testing.T) {
//
//		// make and configure a mocked grpc.PanicHandler
//		mockedPanicHandler := &PanicHandlerMock{
//			LogErrorFunc: func(err error)  {
//				panic("mock out the LogError method")
//			},
//			LogPanicFunc: func(ifaceVal any)  {
//				panic("mock out the LogPanic method")
//			},
//			ReportPanicFunc: func(contextMoqParam context.Context, ifaceVal any) error {
//				panic("mock out the ReportPanic method")
//			},
//		}
//
//		// use mockedPanicHandler in code that requires grpc.PanicHandler
//		// and then make assertions.
//
//	}
type PanicHandlerMock struct {
	// LogErrorFunc mocks the LogError method.
	LogErrorFunc func(err error)

	// LogPanicFunc mocks the LogPanic method.
	LogPanicFunc func(ifaceVal any)

	// ReportPanicFunc mocks the ReportPanic method.
	ReportPanicFunc func(contextMoqParam context.Context, ifaceVal any) error

	// calls tracks calls to the methods.
	calls struct {
		// LogError holds details about calls to the LogError method.
		LogError []struct {
			// Err is the err argument value.
			Err error
		}
		// LogPanic holds details about calls to the LogPanic method.
		LogPanic []struct {
			// IfaceVal is the ifaceVal argument value.
			IfaceVal any
		}
		// ReportPanic holds details about calls to the ReportPanic method.
		ReportPanic []struct {
			// ContextMoqParam is the contextMoqParam argument value.
			ContextMoqParam context.Context
			// IfaceVal is the ifaceVal argument value.
			IfaceVal any
		}
	}
	lockLogError    sync.RWMutex
	lockLogPanic    sync.RWMutex
	lockReportPanic sync.RWMutex
}

// LogError calls LogErrorFunc.
func (mock *PanicHandlerMock) LogError(err error) {
	if mock.LogErrorFunc == nil {
		panic("PanicHandlerMock.LogErrorFunc: method is nil but PanicHandler.LogError was just called")
	}
	callInfo := struct {
		Err error
	}{
		Err: err,
	}
	mock.lockLogError.Lock()
	mock.calls.LogError = append(mock.calls.LogError, callInfo)
	mock.lockLogError.Unlock()
	mock.LogErrorFunc(err)
}

// LogErrorCalls gets all the calls that were made to LogError.
// Check the length with:
//
//	len(mockedPanicHandler.LogErrorCalls())
func (mock *PanicHandlerMock) LogErrorCalls() []struct {
	Err error
} {
	var calls []struct {
		Err error
	}
	mock.lockLogError.RLock()
	calls = mock.calls.LogError
	mock.lockLogError.RUnlock()
	return calls
}

// LogPanic calls LogPanicFunc.
func (mock *PanicHandlerMock) LogPanic(ifaceVal any) {
	if mock.LogPanicFunc == nil {
		panic("PanicHandlerMock.LogPanicFunc: method is nil but PanicHandler.LogPanic was just called")
	}
	callInfo := struct {
		IfaceVal any
	}{
		IfaceVal: ifaceVal,
	}
	mock.lockLogPanic.Lock()
	mock.calls.LogPanic = append(mock.calls.LogPanic, callInfo)
	mock.lockLogPanic.Unlock()
	mock.LogPanicFunc(ifaceVal)
}

// LogPanicCalls gets all the calls that were made to LogPanic.
// Check the length with:
//
//	len(mockedPanicHandler.LogPanicCalls())
func (mock *PanicHandlerMock) LogPanicCalls() []struct {
	IfaceVal any
} {
	var calls []struct {
		IfaceVal any
	}
	mock.lockLogPanic.RLock()
	calls = mock.calls.LogPanic
	mock.lockLogPanic.RUnlock()
	return calls
}

// ReportPanic calls ReportPanicFunc.
func (mock *PanicHandlerMock) ReportPanic(contextMoqParam context.Context, ifaceVal any) error {
	if mock.ReportPanicFunc == nil {
		panic("PanicHandlerMock.ReportPanicFunc: method is nil but PanicHandler.ReportPanic was just called")
	}
	callInfo := struct {
		ContextMoqParam context.Context
		IfaceVal        any
	}{
		ContextMoqParam: contextMoqParam,
		IfaceVal:        ifaceVal,
	}
	mock.lockReportPanic.Lock()
	mock.calls.ReportPanic = append(mock.calls.ReportPanic, callInfo)
	mock.lockReportPanic.Unlock()
	return mock.ReportPanicFunc(contextMoqParam, ifaceVal)
}

// ReportPanicCalls gets all the calls that were made to ReportPanic.
// Check the length with:
//
//	len(mockedPanicHandler.ReportPanicCalls())
func (mock *PanicHandlerMock) ReportPanicCalls() []struct {
	ContextMoqParam context.Context
	IfaceVal        any
} {
	var calls []struct {
		ContextMoqParam context.Context
		IfaceVal        any
	}
	mock.lockReportPanic.RLock()
	calls = mock.calls.ReportPanic
	mock.lockReportPanic.RUnlock()
	return calls
}

// Ensure, that ErrorHandlerMock does implement grpc.ErrorHandler.
// If this is not the case, regenerate this file with moq.
var _ commonsgrpc.ErrorHandler = &ErrorHandlerMock{}

// ErrorHandlerMock is a mock implementation of grpc.ErrorHandler.
//
//	func TestSomethingThatUsesErrorHandler(t *testing.T) {
//
//		// make and configure a mocked grpc.ErrorHandler
//		mockedErrorHandler := &ErrorHandlerMock{
//			ErrorToGRPCStatusFunc: func(err error) (*status.Status, error) {
//				panic("mock out the ErrorToGRPCStatus method")
//			},
//			IsApplicationErrorFunc: func(err error) bool {
//				panic("mock out the IsApplicationError method")
//			},
//			LogErrorFunc: func(err error)  {
//				panic("mock out the LogError method")
//			},
//			ReportErrorFunc: func(contextMoqParam context.Context, err error) error {
//				panic("mock out the ReportError method")
//			},
//		}
//
//		// use mockedErrorHandler in code that requires grpc.ErrorHandler
//		// and then make assertions.
//
//	}
type ErrorHandlerMock struct {
	// ErrorToGRPCStatusFunc mocks the ErrorToGRPCStatus method.
	ErrorToGRPCStatusFunc func(err error) (*status.Status, error)

	// IsApplicationErrorFunc mocks the IsApplicationError method.
	IsApplicationErrorFunc func(err error) bool

	// LogErrorFunc mocks the LogError method.
	LogErrorFunc func(err error)

	// ReportErrorFunc mocks the ReportError method.
	ReportErrorFunc func(contextMoqParam context.Context, err error) error

	// calls tracks calls to the methods.
	calls struct {
		// ErrorToGRPCStatus holds details about calls to the ErrorToGRPCStatus method.
		ErrorToGRPCStatus []struct {
			// Err is the err argument value.
			Err error
		}
		// IsApplicationError holds details about calls to the IsApplicationError method.
		IsApplicationError []struct {
			// Err is the err argument value.
			Err error
		}
		// LogError holds details about calls to the LogError method.
		LogError []struct {
			// Err is the err argument value.
			Err error
		}
		// ReportError holds details about calls to the ReportError method.
		ReportError []struct {
			// ContextMoqParam is the contextMoqParam argument value.
			ContextMoqParam context.Context
			// Err is the err argument value.
			Err error
		}
	}
	lockErrorToGRPCStatus  sync.RWMutex
	lockIsApplicationError sync.RWMutex
	lockLogError           sync.RWMutex
	lockReportError        sync.RWMutex
}

// ErrorToGRPCStatus calls ErrorToGRPCStatusFunc.
func (mock *ErrorHandlerMock) ErrorToGRPCStatus(err error) (*status.Status, error) {
	if mock.ErrorToGRPCStatusFunc == nil {
		panic("ErrorHandlerMock.ErrorToGRPCStatusFunc: method is nil but ErrorHandler.ErrorToGRPCStatus was just called")
	}
	callInfo := struct {
		Err error
	}{
		Err: err,
	}
	mock.lockErrorToGRPCStatus.Lock()
	mock.calls.ErrorToGRPCStatus = append(mock.calls.ErrorToGRPCStatus, callInfo)
	mock.lockErrorToGRPCStatus.Unlock()
	return mock.ErrorToGRPCStatusFunc(err)
}

// ErrorToGRPCStatusCalls gets all the calls that were made to ErrorToGRPCStatus.
// Check the length with:
//
//	len(mockedErrorHandler.ErrorToGRPCStatusCalls())
func (mock *ErrorHandlerMock) ErrorToGRPCStatusCalls() []struct {
	Err error
} {
	var calls []struct {
		Err error
	}
	mock.lockErrorToGRPCStatus.RLock()
	calls = mock.calls.ErrorToGRPCStatus
	mock.lockErrorToGRPCStatus.RUnlock()
	return calls
}

// IsApplicationError calls IsApplicationErrorFunc.
func (mock *ErrorHandlerMock) IsApplicationError(err error) bool {
	if mock.IsApplicationErrorFunc == nil {
		panic("ErrorHandlerMock.IsApplicationErrorFunc: method is nil but ErrorHandler.IsApplicationError was just called")
	}
	callInfo := struct {
		Err error
	}{
		Err: err,
	}
	mock.lockIsApplicationError.Lock()
	mock.calls.IsApplicationError = append(mock.calls.IsApplicationError, callInfo)
	mock.lockIsApplicationError.Unlock()
	return mock.IsApplicationErrorFunc(err)
}

// IsApplicationErrorCalls gets all the calls that were made to IsApplicationError.
// Check the length with:
//
//	len(mockedErrorHandler.IsApplicationErrorCalls())
func (mock *ErrorHandlerMock) IsApplicationErrorCalls() []struct {
	Err error
} {
	var calls []struct {
		Err error
	}
	mock.lockIsApplicationError.RLock()
	calls = mock.calls.IsApplicationError
	mock.lockIsApplicationError.RUnlock()
	return calls
}

// LogError calls LogErrorFunc.
func (mock *ErrorHandlerMock) LogError(err error) {
	if mock.LogErrorFunc == nil {
		panic("ErrorHandlerMock.LogErrorFunc: method is nil but ErrorHandler.LogError was just called")
	}
	callInfo := struct {
		Err error
	}{
		Err: err,
	}
	mock.lockLogError.Lock()
	mock.calls.LogError = append(mock.calls.LogError, callInfo)
	mock.lockLogError.Unlock()
	mock.LogErrorFunc(err)
}

// LogErrorCalls gets all the calls that were made to LogError.
// Check the length with:
//
//	len(mockedErrorHandler.LogErrorCalls())
func (mock *ErrorHandlerMock) LogErrorCalls() []struct {
	Err error
} {
	var calls []struct {
		Err error
	}
	mock.lockLogError.RLock()
	calls = mock.calls.LogError
	mock.lockLogError.RUnlock()
	return calls
}

// ReportError calls ReportErrorFunc.
func (mock *ErrorHandlerMock) ReportError(contextMoqParam context.Context, err error) error {
	if mock.ReportErrorFunc == nil {
		panic("ErrorHandlerMock.ReportErrorFunc: method is nil but ErrorHandler.ReportError was just called")
	}
	callInfo := struct {
		ContextMoqParam context.Context
		Err             error
	}{
		ContextMoqParam: contextMoqParam,
		Err:             err,
	}
	mock.lockReportError.Lock()
	mock.calls.ReportError = append(mock.calls.ReportError, callInfo)
	mock.lockReportError.Unlock()
	return mock.ReportErrorFunc(contextMoqParam, err)
}

// ReportErrorCalls gets all the calls that were made to ReportError.
// Check the length with:
//
//	len(mockedErrorHandler.ReportErrorCalls())
func (mock *ErrorHandlerMock) ReportErrorCalls() []struct {
	ContextMoqParam context.Context
	Err             error
} {
	var calls []struct {
		ContextMoqParam context.Context
		Err             error
	}
	mock.lockReportError.RLock()
	calls = mock.calls.ReportError
	mock.lockReportError.RUnlock()
	return calls
}

package sentry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
)

// Client is a Sentry API Client.
type Client struct{}

// Errors returned by the Client's methods.
var (
	ErrNoClientOrHubAvailable = errors.New("no client or hub available")
	ErrDidNotFullyFlush       = errors.New("not fully flushed")
)

// NewClient returns a new instance of Sentry ReporterService.
func NewClient(
	dsn, environment, release string,
	traceSampleRate float64,
) (*Client, error) {
	err := sentry.Init(sentry.ClientOptions{
		// Either set your DSN here or set the SENTRY_DSN environment variable.
		Dsn: dsn,

		// Either set environment and release here or set the SENTRY_ENVIRONMENT
		// and SENTRY_RELEASE environment variables.
		Environment: environment,

		Release: release,

		TracesSampleRate: traceSampleRate,
	})
	if err != nil {
		return nil, fmt.Errorf("sentry init: %w", err)
	}

	return &Client{}, nil
}

// ReportError reports an error to Sentry.
func (*Client) ReportError(err error) error {
	eventID := sentry.CaptureException(err)
	if eventID == nil {
		return ErrNoClientOrHubAvailable
	}

	return nil
}

// ReportEvent reports an event to Sentry.
func (*Client) ReportEvent(event string) error {
	eventID := sentry.CaptureMessage(event)
	if eventID == nil {
		return ErrNoClientOrHubAvailable
	}

	return nil
}

// MonitorOperation returns a new context to be used with the operation
// and a done function to signal that the operation ended.
func (*Client) MonitorOperation(
	ctx context.Context,
	operation, itemName, traceID string,
	doFunc func(context.Context),
) {
	span := sentry.StartSpan(
		ctx,
		operation,
		sentry.TransactionName(fmt.Sprintf(
			"%s: %s",
			operation,
			itemName,
		)),
	)

	// using "00000000-0000-0000-0000-000000000000" if cannot parse
	// trace id.
	uuidTraceID, _ := uuid.Parse(traceID)

	span.TraceID = sentry.TraceID(uuidTraceID)

	// nolint: contextcheck // allow this context
	doFunc(span.Context())

	span.Finish()
}

// Close flushes the buffered events..
func (*Client) Close() error {
	flushed := sentry.Flush(time.Second)
	if !flushed {
		return ErrDidNotFullyFlush
	}

	return nil
}

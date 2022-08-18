package sentry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
)

// Client is a Sentry API Client.
type Client struct{}

// Errors returned by the Client's methods.
var (
	ErrNoClientOrScopeAvailable = errors.New("no client or hub available")
	ErrDidNotFullyFlush         = errors.New("not fully flushed")
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
func (*Client) ReportError(ctx context.Context, err error) error {
	eventID := hubFromContext(ctx).CaptureException(err)
	if eventID == nil {
		return ErrNoClientOrScopeAvailable
	}

	return nil
}

// ReportEvent reports an event to Sentry.
func (*Client) ReportEvent(ctx context.Context, event string) error {
	eventID := hubFromContext(ctx).CaptureMessage(event)
	if eventID == nil {
		return ErrNoClientOrScopeAvailable
	}

	return nil
}

// MonitorOperation returns a new context to be used with the operation
// and a done function to signal that the operation ended.
func (*Client) MonitorOperation(
	ctx context.Context,
	operation string,
	traceID [16]byte,
	doFunc func(context.Context),
) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
		// nolint: revive // complains about modifying an input param.
		ctx = sentry.SetHubOnContext(ctx, hub)
	}

	span := sentry.StartSpan(
		ctx,
		operation,
	)

	span.TraceID = traceID

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

// hubFromContext returns either a hub stored in the context or the current hub.
// The return value is guaranteed to be non-nil, unlike GetHubFromContext.
func hubFromContext(ctx context.Context) *sentry.Hub {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		return hub
	}

	return sentry.CurrentHub()
}

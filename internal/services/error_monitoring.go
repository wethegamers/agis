package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/getsentry/sentry-go"
)

// ErrorMonitor provides centralized error tracking and monitoring
type ErrorMonitor struct {
	enabled     bool
	environment string
}

// NewErrorMonitor creates a new error monitoring service
func NewErrorMonitor(sentryDSN, environment, release string) (*ErrorMonitor, error) {
	if sentryDSN == "" {
		log.Println("⚠️ Sentry DSN not configured - error monitoring disabled")
		return &ErrorMonitor{enabled: false}, nil
	}

	// Initialize Sentry
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              sentryDSN,
		Environment:      environment,
		Release:          release,
		TracesSampleRate: 0.1, // Sample 10% of transactions for performance monitoring
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Filter out sensitive data
			if event.Request != nil {
				// Scrub authorization headers
				if event.Request.Headers != nil {
					delete(event.Request.Headers, "Authorization")
					delete(event.Request.Headers, "X-WTG-Secret")
					delete(event.Request.Headers, "Cookie")
				}
				// Scrub query params that might contain tokens
				if event.Request.QueryString != "" {
					// Keep query string but mark as potentially sensitive
					event.Request.QueryString = "[REDACTED]"
				}
			}
			return event
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize Sentry: %w", err)
	}

	log.Printf("✅ Sentry error monitoring initialized (env: %s, release: %s)", environment, release)
	return &ErrorMonitor{
		enabled:     true,
		environment: environment,
	}, nil
}

// CaptureError captures and reports an error to Sentry
func (e *ErrorMonitor) CaptureError(err error, tags map[string]string, extra map[string]interface{}) {
	if !e.enabled || err == nil {
		return
	}

	hub := sentry.CurrentHub()
	hub.WithScope(func(scope *sentry.Scope) {
		// Add tags for filtering
		for key, value := range tags {
			scope.SetTag(key, value)
		}

		// Add extra context
		for key, value := range extra {
			scope.SetExtra(key, value)
		}

		hub.CaptureException(err)
	})
}

// CapturePanic captures and reports a panic to Sentry
func (e *ErrorMonitor) CapturePanic(recovered interface{}, tags map[string]string) {
	if !e.enabled || recovered == nil {
		return
	}

	hub := sentry.CurrentHub()
	hub.WithScope(func(scope *sentry.Scope) {
		// Add tags
		for key, value := range tags {
			scope.SetTag(key, value)
		}

		scope.SetLevel(sentry.LevelFatal)

		// Convert panic to error
		var err error
		switch v := recovered.(type) {
		case error:
			err = v
		case string:
			err = fmt.Errorf("panic: %s", v)
		default:
			err = fmt.Errorf("panic: %v", v)
		}

		hub.CaptureException(err)
	})

	// Flush to ensure panic is sent before process exits
	sentry.Flush(2 * time.Second)
}

// CaptureMessage captures a message (warning/info) to Sentry
func (e *ErrorMonitor) CaptureMessage(message string, level sentry.Level, tags map[string]string) {
	if !e.enabled {
		return
	}

	hub := sentry.CurrentHub()
	hub.WithScope(func(scope *sentry.Scope) {
		for key, value := range tags {
			scope.SetTag(key, value)
		}
		scope.SetLevel(level)
		hub.CaptureMessage(message)
	})
}

// RecoverAndCapture wraps a function to capture panics
func (e *ErrorMonitor) RecoverAndCapture(tags map[string]string) {
	if r := recover(); r != nil {
		e.CapturePanic(r, tags)
		// Re-panic to maintain original behavior
		panic(r)
	}
}

// WrapHandler wraps an error-returning function with error capture
func (e *ErrorMonitor) WrapHandler(fn func() error, context string, tags map[string]string) error {
	err := fn()
	if err != nil {
		if tags == nil {
			tags = make(map[string]string)
		}
		tags["context"] = context
		e.CaptureError(err, tags, nil)
	}
	return err
}

// TrackPaymentError captures payment-related errors with high priority
func (e *ErrorMonitor) TrackPaymentError(err error, userID, sessionID string, amount int64) {
	e.CaptureError(err, map[string]string{
		"error_type": "payment",
		"user_id":    userID,
		"session_id": sessionID,
	}, map[string]interface{}{
		"amount_cents": amount,
	})

	// Also send alert-level message for critical payment failures
	e.CaptureMessage(
		fmt.Sprintf("Payment failure for user %s: %v", userID, err),
		sentry.LevelError,
		map[string]string{"critical": "payment"},
	)
}

// TrackAdCallbackError captures ad callback errors
func (e *ErrorMonitor) TrackAdCallbackError(err error, provider, conversionID, userID string) {
	e.CaptureError(err, map[string]string{
		"error_type":    "ad_callback",
		"provider":      provider,
		"conversion_id": conversionID,
		"user_id":       userID,
	}, nil)
}

// TrackDatabaseError captures database-related errors
func (e *ErrorMonitor) TrackDatabaseError(err error, operation, table string) {
	e.CaptureError(err, map[string]string{
		"error_type": "database",
		"operation":  operation,
		"table":      table,
	}, nil)
}

// SetUser sets the current user context for subsequent errors
func (e *ErrorMonitor) SetUser(userID, username, ipAddress string) {
	if !e.enabled {
		return
	}

	hub := sentry.CurrentHub()
	hub.Scope().SetUser(sentry.User{
		ID:        userID,
		Username:  username,
		IPAddress: ipAddress,
	})
}

// SetContext adds custom context to the current scope
func (e *ErrorMonitor) SetContext(key string, value map[string]interface{}) {
	if !e.enabled {
		return
	}

	hub := sentry.CurrentHub()
	hub.Scope().SetContext(key, value)
}

// StartTransaction starts a performance transaction
func (e *ErrorMonitor) StartTransaction(ctx context.Context, name string, op string) *sentry.Span {
	if !e.enabled {
		return nil
	}

	return sentry.StartSpan(ctx, op, sentry.WithTransactionName(name))
}

// Flush ensures all pending events are sent (call on shutdown)
func (e *ErrorMonitor) Flush(timeout time.Duration) {
	if !e.enabled {
		return
	}

	sentry.Flush(timeout)
	log.Println("✅ Sentry events flushed")
}

// Close cleans up the error monitor
func (e *ErrorMonitor) Close() {
	if e.enabled {
		sentry.Flush(2 * time.Second)
	}
}

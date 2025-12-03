package tracing

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Enabled {
		t.Error("Expected Enabled to be false by default")
	}
	if cfg.ServiceName != "agis-bot" {
		t.Errorf("Expected ServiceName to be agis-bot, got %s", cfg.ServiceName)
	}
	if cfg.Environment != "development" {
		t.Errorf("Expected Environment to be development, got %s", cfg.Environment)
	}
	if cfg.SampleRate != 1.0 {
		t.Errorf("Expected SampleRate to be 1.0, got %f", cfg.SampleRate)
	}
}

func TestInit_Disabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	provider, err := Init(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if provider.Tracer() == nil {
		t.Error("Expected tracer to be non-nil even when disabled")
	}

	// Shutdown should be safe
	if err := provider.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestStartSpan(t *testing.T) {
	ctx, span := StartSpan(context.Background(), "test-span",
		attribute.String("test.key", "test-value"),
	)

	if span == nil {
		t.Fatal("Expected span to be non-nil")
	}

	if ctx == nil {
		t.Fatal("Expected context to be non-nil")
	}

	span.End()
}

func TestSpanFromContext(t *testing.T) {
	ctx, span := StartSpan(context.Background(), "parent-span")
	defer span.End()

	retrieved := SpanFromContext(ctx)
	if retrieved == nil {
		t.Error("Expected to retrieve span from context")
	}
}

func TestRecordError(t *testing.T) {
	ctx, span := StartSpan(context.Background(), "error-span")
	defer span.End()

	testErr := context.DeadlineExceeded
	RecordError(ctx, testErr)
	// No panic means success - error is recorded on span
}

func TestSetStatus(t *testing.T) {
	ctx, span := StartSpan(context.Background(), "status-span")
	defer span.End()

	SetStatus(ctx, codes.Ok, "all good")
	// No panic means success
}

func TestAddEvent(t *testing.T) {
	ctx, span := StartSpan(context.Background(), "event-span")
	defer span.End()

	AddEvent(ctx, "test-event", attribute.String("key", "value"))
	// No panic means success
}

func TestDiscordAttributes(t *testing.T) {
	tests := []struct {
		name      string
		guildID   string
		channelID string
		userID    string
		wantLen   int
	}{
		{"all set", "123", "456", "789", 3},
		{"only guild", "123", "", "", 1},
		{"guild and channel", "123", "456", "", 2},
		{"none set", "", "", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := DiscordAttributes(tt.guildID, tt.channelID, tt.userID)
			if len(attrs) != tt.wantLen {
				t.Errorf("Expected %d attributes, got %d", tt.wantLen, len(attrs))
			}
		})
	}
}

func TestHTTPMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	wrapped := HTTPMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestHTTPMiddleware_Error(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	wrapped := HTTPMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/error", http.NoBody)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rec.Code)
	}
}

func TestTraceID_NoSpan(t *testing.T) {
	id := TraceID(context.Background())
	if id != "" {
		t.Errorf("Expected empty trace ID for context without span, got %s", id)
	}
}

func TestSpanID_NoSpan(t *testing.T) {
	id := SpanID(context.Background())
	if id != "" {
		t.Errorf("Expected empty span ID for context without span, got %s", id)
	}
}

func TestTimed_Success(t *testing.T) {
	executed := false

	err := Timed(context.Background(), "timed-op", func(ctx context.Context) error {
		executed = true
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !executed {
		t.Error("Expected function to be executed")
	}
}

func TestTimed_Error(t *testing.T) {
	expectedErr := context.Canceled

	err := Timed(context.Background(), "timed-op-error", func(ctx context.Context) error {
		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestStatusWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rec, status: http.StatusOK}

	sw.WriteHeader(http.StatusNotFound)

	if sw.status != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", sw.status)
	}
}

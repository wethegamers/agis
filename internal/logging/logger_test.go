package logging

import (
	"bytes"
	"context"
	"testing"
)

func TestNewLogger(t *testing.T) {
	logger := New(Config{Level: "info"})
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestLoggerWithComponent(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Level: "info", Format: "json", Output: &buf})
	componentLogger := logger.WithComponent("test-component")
	componentLogger.Info("test message")
	if buf.Len() == 0 {
		t.Error("expected log output")
	}
}

func TestContextLogger(t *testing.T) {
	logger := New(Config{Level: "info"})
	ctx := WithLogger(context.Background(), logger)
	retrieved := FromContext(ctx)
	if retrieved == nil {
		t.Fatal("expected logger from context")
	}
}

func TestFromContextDefault(t *testing.T) {
	logger := FromContext(context.Background())
	if logger == nil {
		t.Error("expected default logger")
	}
}

package app

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockService struct {
	name    string
	stopped bool
}

func (m *mockService) Name() string                    { return m.name }
func (m *mockService) Start(ctx context.Context) error { return nil }
func (m *mockService) Stop(ctx context.Context) error  { m.stopped = true; return nil }

func TestNew(t *testing.T) {
	app := New("test-app", "1.0.0")
	if app.Name() != "test-app" {
		t.Errorf("expected name test-app, got %s", app.Name())
	}
	if app.Version() != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", app.Version())
	}
}

func TestWithShutdownTimeout(t *testing.T) {
	app := New("test", "1.0", WithShutdownTimeout(5*time.Second))
	if app.shutdownTimeout != 5*time.Second {
		t.Errorf("expected 5s timeout, got %v", app.shutdownTimeout)
	}
}

func TestRegister(t *testing.T) {
	app := New("test", "1.0")
	svc := &mockService{name: "test-service"}
	app.Register(svc)
	if len(app.services) != 1 {
		t.Errorf("expected 1 service, got %d", len(app.services))
	}
}

func TestShutdown(t *testing.T) {
	app := New("test", "1.0", WithShutdownTimeout(1*time.Second))
	svc := &mockService{name: "service1"}
	app.Register(svc)
	err := app.Shutdown()
	if err != nil {
		t.Errorf("unexpected shutdown error: %v", err)
	}
	if !svc.stopped {
		t.Error("service not stopped")
	}
}

func TestOnStartHookError(t *testing.T) {
	app := New("test", "1.0", WithOnStart(func(ctx context.Context) error {
		return errors.New("startup failed")
	}))
	err := app.Run(context.Background())
	if err == nil {
		t.Error("expected error from onStart hook")
	}
}

func TestOnStopHook(t *testing.T) {
	var hookCalled bool
	app := New("test", "1.0", WithOnStop(func(ctx context.Context) error {
		hookCalled = true
		return nil
	}))
	_ = app.Shutdown()
	if !hookCalled {
		t.Error("onStop hook was not called")
	}
}

func TestDoubleStart(t *testing.T) {
	app := New("test", "1.0")
	app.started = true
	err := app.Run(context.Background())
	if err == nil {
		t.Error("expected error for double start")
	}
}

func TestServiceFunc(t *testing.T) {
	var started, stopped bool
	svc := NewServiceFunc("test-func",
		func(ctx context.Context) error { started = true; return nil },
		func(ctx context.Context) error { stopped = true; return nil },
	)
	if svc.Name() != "test-func" {
		t.Errorf("expected name test-func, got %s", svc.Name())
	}
	_ = svc.Start(context.Background())
	if !started {
		t.Error("start function not called")
	}
	_ = svc.Stop(context.Background())
	if !stopped {
		t.Error("stop function not called")
	}
}

package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewChecker(t *testing.T) {
	c := NewChecker()
	if c == nil {
		t.Fatal("expected non-nil checker")
	}
	if c.IsReady() {
		t.Error("expected checker to start not ready")
	}
}

func TestChecker_SetReady(t *testing.T) {
	c := NewChecker()
	c.SetReady(true)
	if !c.IsReady() {
		t.Error("expected ready after SetReady(true)")
	}
	c.SetReady(false)
	if c.IsReady() {
		t.Error("expected not ready after SetReady(false)")
	}
}

func TestChecker_Register(t *testing.T) {
	c := NewChecker()
	called := false
	c.Register("test", func(ctx context.Context) error {
		called = true
		return nil
	})
	c.SetReady(true)
	status, results := c.RunAll(context.Background())
	if !called {
		t.Error("expected check to be called")
	}
	if status != StatusHealthy {
		t.Errorf("expected healthy status, got %s", status)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "test" {
		t.Errorf("expected name 'test', got %s", results[0].Name)
	}
}

func TestChecker_RunAll_Unhealthy(t *testing.T) {
	c := NewChecker()
	errTest := errors.New("test error")
	c.Register("failing", func(ctx context.Context) error {
		return errTest
	})
	status, results := c.RunAll(context.Background())
	// When 100% of checks fail, system is unhealthy (not degraded)
	if status != StatusUnhealthy {
		t.Errorf("expected unhealthy status, got %s", status)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusUnhealthy {
		t.Errorf("expected unhealthy check, got %s", results[0].Status)
	}
}

func TestChecker_RunAll_MajorityUnhealthy(t *testing.T) {
	c := NewChecker()
	c.Register("ok1", func(ctx context.Context) error { return nil })
	c.Register("fail1", func(ctx context.Context) error { return errors.New("fail") })
	c.Register("fail2", func(ctx context.Context) error { return errors.New("fail") })
	c.Register("fail3", func(ctx context.Context) error { return errors.New("fail") })
	status, _ := c.RunAll(context.Background())
	if status != StatusUnhealthy {
		t.Errorf("expected unhealthy when majority fail, got %s", status)
	}
}

func TestNewHandler(t *testing.T) {
	c := NewChecker()
	h := NewHandler(c, "1.0.0")
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestHandler_Liveness(t *testing.T) {
	c := NewChecker()
	h := NewHandler(c, "1.0.0")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Status != StatusHealthy {
		t.Errorf("expected healthy, got %s", resp.Status)
	}
}

func TestHandler_Readiness_NotReady(t *testing.T) {
	c := NewChecker()
	h := NewHandler(c, "1.0.0")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when not ready, got %d", rec.Code)
	}
}

func TestHandler_Readiness_Ready(t *testing.T) {
	c := NewChecker()
	c.SetReady(true)
	h := NewHandler(c, "1.0.0")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 when ready, got %d", rec.Code)
	}
}

func TestHandler_Detailed(t *testing.T) {
	c := NewChecker()
	c.SetReady(true)
	c.Register("test1", func(ctx context.Context) error { return nil })
	h := NewHandler(c, "1.0.0")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/health/detailed", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Checks) != 1 {
		t.Errorf("expected 1 check in response, got %d", len(resp.Checks))
	}
}

func TestDatabaseCheck(t *testing.T) {
	called := false
	check := DatabaseCheck(func(ctx context.Context) error {
		called = true
		return nil
	})
	err := check(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !called {
		t.Error("expected ping function to be called")
	}
}

func TestDiscordCheck_Connected(t *testing.T) {
	check := DiscordCheck(func() bool { return true })
	err := check(context.Background())
	if err != nil {
		t.Errorf("expected no error when connected, got %v", err)
	}
}

func TestDiscordCheck_Disconnected(t *testing.T) {
	check := DiscordCheck(func() bool { return false })
	err := check(context.Background())
	if err == nil {
		t.Error("expected error when disconnected")
	}
}

func TestKubernetesCheck(t *testing.T) {
	check := KubernetesCheck(func() (string, error) {
		return "v1.28.0", nil
	})
	err := check(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

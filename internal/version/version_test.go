package version

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGet(t *testing.T) {
	info := Get()
	if info.Version == "" {
		t.Error("expected non-empty version")
	}
	if info.GoVersion == "" {
		t.Error("expected non-empty Go version")
	}
	if info.Platform == "" {
		t.Error("expected non-empty platform")
	}
	if info.Uptime == "" {
		t.Error("expected non-empty uptime")
	}
}

func TestInfoString(t *testing.T) {
	info := Get()
	s := info.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
}

func TestInfoShortString(t *testing.T) {
	info := Get()
	s := info.ShortString()
	if s == "" {
		t.Error("expected non-empty short string")
	}
}

func TestHandler(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Error("expected Content-Type application/json")
	}

	var info Info
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if info.Version == "" {
		t.Error("expected version in response")
	}
}

func TestShortHandler(t *testing.T) {
	handler := ShortHandler()
	req := httptest.NewRequest(http.MethodGet, "/version/short", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/plain" {
		t.Error("expected Content-Type text/plain")
	}
	if rec.Body.String() == "" {
		t.Error("expected non-empty response body")
	}
}

func TestGetRuntime(t *testing.T) {
	stats := GetRuntime()
	if stats.NumCPU <= 0 {
		t.Error("expected positive NumCPU")
	}
	if stats.NumGoroutine <= 0 {
		t.Error("expected positive NumGoroutine")
	}
}

func TestRuntimeHandler(t *testing.T) {
	handler := RuntimeHandler()
	req := httptest.NewRequest(http.MethodGet, "/runtime", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var stats Runtime
	if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
}

func TestFullInfoHandler(t *testing.T) {
	handler := FullInfoHandler()
	req := httptest.NewRequest(http.MethodGet, "/info", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response struct {
		Version Info    `json:"version"`
		Runtime Runtime `json:"runtime"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response.Version.Version == "" {
		t.Error("expected version in response")
	}
}

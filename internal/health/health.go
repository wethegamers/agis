// Package health provides Kubernetes-compatible health and readiness probes.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status represents the health status of a component.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// Check is a function that performs a health check.
type Check func(ctx context.Context) error

// Checker manages health checks for the application.
type Checker struct {
	checks map[string]Check
	mu     sync.RWMutex
	ready  bool
}

// NewChecker creates a new health checker.
func NewChecker() *Checker {
	return &Checker{
		checks: make(map[string]Check),
		ready:  false,
	}
}

// Register adds a named health check.
func (c *Checker) Register(name string, check Check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks[name] = check
}

// SetReady sets the readiness state.
func (c *Checker) SetReady(ready bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ready = ready
}

// IsReady returns the readiness state.
func (c *Checker) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}

// CheckResult represents the result of a health check.
type CheckResult struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// HealthResponse is the response for health endpoints.
type HealthResponse struct {
	Status    Status        `json:"status"`
	Timestamp string        `json:"timestamp"`
	Version   string        `json:"version,omitempty"`
	Checks    []CheckResult `json:"checks,omitempty"`
}

// RunAll executes all registered health checks.
func (c *Checker) RunAll(ctx context.Context) (Status, []CheckResult) {
	c.mu.RLock()
	checks := make(map[string]Check, len(c.checks))
	for k, v := range c.checks {
		checks[k] = v
	}
	c.mu.RUnlock()

	results := make([]CheckResult, 0, len(checks))
	overallStatus := StatusHealthy

	for name, check := range checks {
		start := time.Now()
		err := check(ctx)
		latency := time.Since(start)

		result := CheckResult{
			Name:    name,
			Status:  StatusHealthy,
			Latency: latency.String(),
		}

		if err != nil {
			result.Status = StatusUnhealthy
			result.Message = err.Error()
			if overallStatus == StatusHealthy {
				overallStatus = StatusDegraded
			}
		}
		results = append(results, result)
	}

	// If more than half the checks failed, mark as unhealthy
	unhealthyCount := 0
	for _, r := range results {
		if r.Status == StatusUnhealthy {
			unhealthyCount++
		}
	}
	if unhealthyCount > len(results)/2 {
		overallStatus = StatusUnhealthy
	}

	return overallStatus, results
}

// Handler provides HTTP handlers for health endpoints.
type Handler struct {
	checker *Checker
	version string
}

// NewHandler creates a new health handler.
func NewHandler(checker *Checker, version string) *Handler {
	return &Handler{
		checker: checker,
		version: version,
	}
}

// RegisterRoutes registers health endpoints on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Kubernetes liveness probe
	mux.HandleFunc("GET /healthz", h.handleLiveness)
	mux.HandleFunc("GET /health", h.handleLiveness)

	// Kubernetes readiness probe
	mux.HandleFunc("GET /readyz", h.handleReadiness)
	mux.HandleFunc("GET /ready", h.handleReadiness)

	// Detailed health status (for debugging)
	mux.HandleFunc("GET /health/detailed", h.handleDetailed)
}

// handleLiveness handles the liveness probe.
// Returns 200 if the application is running, regardless of dependency status.
func (h *Handler) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthResponse{
		Status:    StatusHealthy,
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   h.version,
	})
}

// handleReadiness handles the readiness probe.
// Returns 200 only if the application is ready to serve traffic.
func (h *Handler) handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !h.checker.IsReady() {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(HealthResponse{
			Status:    StatusUnhealthy,
			Timestamp: time.Now().Format(time.RFC3339),
			Version:   h.version,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status, _ := h.checker.RunAll(ctx)

	httpStatus := http.StatusOK
	if status != StatusHealthy {
		httpStatus = http.StatusServiceUnavailable
	}

	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(HealthResponse{
		Status:    status,
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   h.version,
	})
}

// handleDetailed returns detailed health information.
func (h *Handler) handleDetailed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	status, checks := h.checker.RunAll(ctx)

	httpStatus := http.StatusOK
	if status != StatusHealthy {
		httpStatus = http.StatusServiceUnavailable
	}

	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(HealthResponse{
		Status:    status,
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   h.version,
		Checks:    checks,
	})
}

// Common health check implementations

// DatabaseCheck creates a health check for a database connection.
func DatabaseCheck(pingFn func(context.Context) error) Check {
	return func(ctx context.Context) error {
		return pingFn(ctx)
	}
}

// DiscordCheck creates a health check for the Discord connection.
func DiscordCheck(isConnectedFn func() bool) Check {
	return func(ctx context.Context) error {
		if !isConnectedFn() {
			return &healthError{message: "discord not connected"}
		}
		return nil
	}
}

// KubernetesCheck creates a health check for Kubernetes connectivity.
func KubernetesCheck(versionFn func() (string, error)) Check {
	return func(ctx context.Context) error {
		_, err := versionFn()
		return err
	}
}

type healthError struct {
	message string
}

func (e *healthError) Error() string {
	return e.message
}

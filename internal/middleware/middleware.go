// Package middleware provides HTTP middleware for AGIS.
package middleware

import (
"context"
"net/http"
"strconv"
"sync"
"time"

"github.com/google/uuid"
"github.com/prometheus/client_golang/prometheus"
"github.com/prometheus/client_golang/prometheus/promauto"
)

// RequestID adds a unique request ID to each request.
func RequestID(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
requestID := r.Header.Get("X-Request-ID")
if requestID == "" {
requestID = uuid.New().String()
}
w.Header().Set("X-Request-ID", requestID)
r.Header.Set("X-Request-ID", requestID)
next.ServeHTTP(w, r)
})
}

// Metrics middleware for Prometheus metrics.
type Metrics struct {
requests *prometheus.CounterVec
duration *prometheus.HistogramVec
inflight prometheus.Gauge
}

// NewMetrics creates metrics middleware.
func NewMetrics(namespace string) *Metrics {
return &Metrics{
requests: promauto.NewCounterVec(
prometheus.CounterOpts{
Namespace: namespace,
Name:      "http_requests_total",
Help:      "Total HTTP requests",
},
[]string{"method", "path", "status"},
),
duration: promauto.NewHistogramVec(
prometheus.HistogramOpts{
Namespace: namespace,
Name:      "http_request_duration_seconds",
Help:      "HTTP request duration",
Buckets:   prometheus.DefBuckets,
},
[]string{"method", "path"},
),
inflight: promauto.NewGauge(
prometheus.GaugeOpts{
Namespace: namespace,
Name:      "http_requests_inflight",
Help:      "Current in-flight requests",
},
),
}
}

// Handler wraps an http.Handler with metrics.
func (m *Metrics) Handler(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
m.inflight.Inc()
defer m.inflight.Dec()

start := time.Now()
wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

next.ServeHTTP(wrapped, r)

duration := time.Since(start).Seconds()
status := strconv.Itoa(wrapped.statusCode)

m.requests.WithLabelValues(r.Method, r.URL.Path, status).Inc()
m.duration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
})
}

// RateLimiter implements token bucket rate limiting.
type RateLimiter struct {
mu       sync.Mutex
tokens   map[string]*bucket
rate     float64
burst    int
cleanup  time.Duration
}

type bucket struct {
tokens    float64
lastCheck time.Time
}

// NewRateLimiter creates a rate limiter.
func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
rl := &RateLimiter{
tokens:  make(map[string]*bucket),
rate:    requestsPerSecond,
burst:   burst,
cleanup: 10 * time.Minute,
}
go rl.cleanupLoop()
return rl
}

func (rl *RateLimiter) cleanupLoop() {
ticker := time.NewTicker(rl.cleanup)
for range ticker.C {
rl.mu.Lock()
now := time.Now()
for key, b := range rl.tokens {
if now.Sub(b.lastCheck) > rl.cleanup {
delete(rl.tokens, key)
}
}
rl.mu.Unlock()
}
}

// Allow checks if a request is allowed.
func (rl *RateLimiter) Allow(key string) bool {
rl.mu.Lock()
defer rl.mu.Unlock()

now := time.Now()
b, ok := rl.tokens[key]
if !ok {
b = &bucket{tokens: float64(rl.burst), lastCheck: now}
rl.tokens[key] = b
}

elapsed := now.Sub(b.lastCheck).Seconds()
b.tokens += elapsed * rl.rate
if b.tokens > float64(rl.burst) {
b.tokens = float64(rl.burst)
}
b.lastCheck = now

if b.tokens >= 1 {
b.tokens--
return true
}
return false
}

// Handler returns rate limiting middleware.
func (rl *RateLimiter) Handler(keyFunc func(*http.Request) string) func(http.Handler) http.Handler {
return func(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
key := keyFunc(r)
if !rl.Allow(key) {
http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
return
}
next.ServeHTTP(w, r)
})
}
}

// IPKeyFunc extracts client IP for rate limiting.
func IPKeyFunc(r *http.Request) string {
if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
return ip
}
if ip := r.Header.Get("X-Real-IP"); ip != "" {
return ip
}
return r.RemoteAddr
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
http.ResponseWriter
statusCode int
written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
if !rw.written {
rw.statusCode = code
rw.written = true
}
rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
if !rw.written {
rw.statusCode = http.StatusOK
rw.written = true
}
return rw.ResponseWriter.Write(b)
}

// Recover middleware recovers from panics.
func Recover(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
defer func() {
if err := recover(); err != nil {
http.Error(w, "internal server error", http.StatusInternalServerError)
}
}()
next.ServeHTTP(w, r)
})
}

// CORS adds CORS headers.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
originsSet := make(map[string]bool)
for _, o := range allowedOrigins {
originsSet[o] = true
}

return func(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
origin := r.Header.Get("Origin")
if originsSet[origin] || originsSet["*"] {
w.Header().Set("Access-Control-Allow-Origin", origin)
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
w.Header().Set("Access-Control-Max-Age", "86400")
}
if r.Method == http.MethodOptions {
w.WriteHeader(http.StatusNoContent)
return
}
next.ServeHTTP(w, r)
})
}
}

// Chain chains multiple middleware together.
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
return func(final http.Handler) http.Handler {
for i := len(middlewares) - 1; i >= 0; i-- {
final = middlewares[i](final)
}
return final
}
}

// APIVersion represents an API version.
type APIVersion string

const (
	// APIVersionV1 is API version 1.
	APIVersionV1 APIVersion = "v1"
	// APIVersionV2 is API version 2.
	APIVersionV2 APIVersion = "v2"

	// APIVersionHeader is the header for specifying API version.
	APIVersionHeader = "X-API-Version"
	// APIVersionContextKey is the context key for storing API version.
	APIVersionContextKey = "api_version"
)

// APIVersioning handles API version detection and routing.
type APIVersioning struct {
	defaultVersion APIVersion
	supported      map[APIVersion]bool
	deprecated     map[APIVersion]string // version -> deprecation message
}

// NewAPIVersioning creates a new API versioning middleware.
func NewAPIVersioning(defaultVersion APIVersion, supported []APIVersion) *APIVersioning {
	av := &APIVersioning{
		defaultVersion: defaultVersion,
		supported:      make(map[APIVersion]bool),
		deprecated:     make(map[APIVersion]string),
	}
	for _, v := range supported {
		av.supported[v] = true
	}
	return av
}

// Deprecate marks a version as deprecated with a message.
func (av *APIVersioning) Deprecate(version APIVersion, message string) {
	av.deprecated[version] = message
}

// Handler returns middleware that extracts and validates API version.
func (av *APIVersioning) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check header first
		version := APIVersion(r.Header.Get(APIVersionHeader))

		// Fall back to URL path extraction (e.g., /api/v1/...)
		if version == "" {
			version = av.extractFromPath(r.URL.Path)
		}

		// Use default if not specified
		if version == "" {
			version = av.defaultVersion
		}

		// Validate version
		if !av.supported[version] {
			http.Error(w, "unsupported API version: "+string(version), http.StatusBadRequest)
			return
		}

		// Add deprecation warning if applicable
		if msg, ok := av.deprecated[version]; ok {
			w.Header().Set("X-API-Deprecated", "true")
			w.Header().Set("X-API-Deprecation-Message", msg)
		}

		// Set version in response header
		w.Header().Set(APIVersionHeader, string(version))

		// Store in context
		ctx := r.Context()
		ctx = setAPIVersion(ctx, version)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// extractFromPath extracts API version from URL path.
// Supports patterns like /api/v1/... or /v1/...
func (av *APIVersioning) extractFromPath(path string) APIVersion {
	// Simple extraction - look for v1, v2, etc.
	if len(path) < 3 {
		return ""
	}

	// Check for /api/v1/ or /v1/ patterns
	patterns := []string{"/api/v1/", "/api/v2/", "/v1/", "/v2/"}
	versions := []APIVersion{APIVersionV1, APIVersionV2, APIVersionV1, APIVersionV2}

	for i, pattern := range patterns {
		if len(path) >= len(pattern) && path[:len(pattern)] == pattern {
			return versions[i]
		}
	}

	return ""
}

type apiVersionContextKey struct{}

func setAPIVersion(ctx context.Context, version APIVersion) context.Context {
	return context.WithValue(ctx, apiVersionContextKey{}, version)
}

// GetAPIVersion retrieves the API version from context.
func GetAPIVersion(ctx context.Context) APIVersion {
	if v, ok := ctx.Value(apiVersionContextKey{}).(APIVersion); ok {
		return v
	}
	return ""
}

// Timeout adds a timeout to requests.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()

			done := make(chan struct{})
			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				// Request completed normally
			case <-ctx.Done():
				if ctx.Err() == context.DeadlineExceeded {
					http.Error(w, "request timeout", http.StatusGatewayTimeout)
				}
			}
		})
	}
}

// SecurityHeaders adds security-related headers.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

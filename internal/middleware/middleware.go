// Package middleware provides HTTP middleware for AGIS.
package middleware

import (
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

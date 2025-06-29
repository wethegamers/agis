package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"agis-bot/internal/version"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server represents the HTTP server for metrics and health checks
type Server struct {
	server *http.Server
}

// NewServer creates a new HTTP server
func NewServer() *Server {
	mux := http.NewServeMux()
	
	// Health endpoint
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/healthz", healthHandler) // K8s standard alias
	
	// Readiness endpoint
	mux.HandleFunc("/ready", readinessHandler)
	mux.HandleFunc("/readyz", readinessHandler) // K8s standard alias
	
	// Info/About endpoint
	mux.HandleFunc("/info", infoHandler)
	mux.HandleFunc("/about", infoHandler) // Alias
	
	// Version endpoint
	mux.HandleFunc("/version", versionHandler)
	
	// Metrics endpoint (Prometheus metrics)
	mux.Handle("/metrics", promhttp.Handler())
	
	// Root endpoint
	mux.HandleFunc("/", rootHandler)

	server := &http.Server{
		Addr:         ":9090", // Prometheus standard port
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{server: server}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("🌐 Starting HTTP server on port 9090 (Prometheus standard)")
	return s.server.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("🛑 Shutting down HTTP server...")
	return s.server.Shutdown(ctx)
}

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "agis-bot",
	})
}

// Readiness check handler
func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "agis-bot",
	})
}

// Info/About handler
func infoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	buildInfo := version.GetBuildInfo()
	response := map[string]interface{}{
		"service":     "agis-bot",
		"description": "WTG Agones GameServer Management Bot",
		"build":       buildInfo,
		"endpoints": map[string]string{
			"/health":  "Health check endpoint",
			"/ready":   "Readiness check endpoint", 
			"/info":    "Service information and build details",
			"/version": "Version information only",
			"/metrics": "Prometheus metrics",
		},
	}
	
	json.NewEncoder(w).Encode(response)
}

// Version handler
func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(version.GetBuildInfo())
}

// Root handler
func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": "agis-bot",
		"status":  "running",
		"endpoints": []string{
			"/health", "/healthz",
			"/ready", "/readyz", 
			"/info", "/about",
			"/version",
			"/metrics",
		},
	})
}

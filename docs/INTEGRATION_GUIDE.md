# Internal Package Integration Guide

This document describes how to integrate the new internal packages into the AGIS codebase.

## Overview

The following internal packages are available for use:

| Package | Purpose | Status |
|---------|---------|--------|
| `internal/app` | Application lifecycle management | Ready |
| `internal/config` | Environment variable configuration | Ready |
| `internal/health` | Kubernetes health probes | Ready |
| `internal/logging` | Structured logging (slog) | Ready |
| `internal/metrics` | Prometheus metrics definitions | Ready |
| `internal/middleware` | HTTP middleware (RequestID, CORS, etc.) | Ready |
| `internal/opensaas` | OpenSaaS/Wasp integration | Ready |
| `internal/tracing` | OpenTelemetry distributed tracing | Ready |
| `internal/version` | Build info and version handlers | Ready |

## Integration Strategy

Since `main.go` heavily depends on `agis-core`, we recommend a gradual migration:

### Phase 1: Add Tracing (Recommended First)

Add OpenTelemetry tracing for distributed tracing across services:

```go
import (
    "github.com/wethegamers/agis/internal/tracing"
)

func main() {
    // Initialize tracing early
    tracingCfg := tracing.DefaultConfig()
    tracingCfg.ServiceName = "agis"
    tracingCfg.Enabled = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != ""
    tracingCfg.OTLPEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
    
    provider, err := tracing.Init(context.Background(), tracingCfg)
    if err != nil {
        log.Printf("⚠️ Failed to initialize tracing: %v", err)
    } else if provider != nil {
        defer provider.Shutdown(context.Background())
        log.Println("✅ OpenTelemetry tracing initialized")
    }
    
    // ... rest of main
}
```

### Phase 2: Use internal/metrics

Replace the inline metric definitions with `internal/metrics`:

```go
import (
    "github.com/wethegamers/agis/internal/metrics"
)

// Instead of defining metrics in main.go, use:
metrics.CommandsTotal.WithLabelValues("help", "success").Inc()
metrics.GameServersTotal.WithLabelValues("minecraft", "running").Set(5)
metrics.SetBuildInfo(version.Version, version.GitCommit, version.BuildTime)
```

### Phase 3: Add Structured Logging

Replace `log` with `internal/logging` for structured JSON logs:

```go
import (
    "github.com/wethegamers/agis/internal/logging"
)

func main() {
    logCfg := logging.DefaultConfig()
    logCfg.Level = os.Getenv("LOG_LEVEL") // default: info
    logCfg.Format = "json"
    
    logger := logging.New(logCfg)
    
    // Replace log.Printf with:
    logger.Info("Bot started", "version", version.Version)
    logger.Error("Failed to connect", "error", err, "service", "discord")
}
```

### Phase 4: Use internal/version

Add version HTTP handlers to the existing HTTP server:

```go
import (
    "github.com/wethegamers/agis/internal/version"
)

func setupHTTPHandlers(mux *http.ServeMux) {
    mux.Handle("GET /version", version.Handler())
    mux.Handle("GET /version/short", version.ShortHandler())
    mux.Handle("GET /runtime", version.RuntimeHandler())
    mux.Handle("GET /info", version.FullInfoHandler())
}
```

### Phase 5: Use internal/health

Replace custom health checks with `internal/health`:

```go
import (
    "github.com/wethegamers/agis/internal/health"
)

func main() {
    checker := health.NewChecker()
    
    // Register health checks
    checker.Register("database", health.DatabaseCheck(func(ctx context.Context) error {
        return dbService.Ping(ctx)
    }))
    
    checker.Register("discord", health.DiscordCheck(func() bool {
        return session.State != nil
    }))
    
    // Create health handler
    healthHandler := health.NewHandler(checker, version.Version)
    healthHandler.RegisterRoutes(mux)
    
    // Set ready when initialization complete
    checker.SetReady(true)
}
```

## Environment Variables

New environment variables for internal packages:

| Variable | Package | Description | Default |
|----------|---------|-------------|---------|
| `LOG_LEVEL` | logging | Log level (debug/info/warn/error) | info |
| `LOG_FORMAT` | logging | Output format (json/text) | json |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | tracing | OTLP collector endpoint | (disabled) |
| `OTEL_SERVICE_NAME` | tracing | Service name for traces | agis |

## Testing

All internal packages have comprehensive tests:

```bash
# Run all internal package tests
go test ./internal/... -v

# Run with coverage
go test ./internal/... -cover
```

## Compatibility Notes

1. **agis-core compatibility**: Internal packages are designed to work alongside agis-core, not replace it entirely. Use internal packages for new features and gradually migrate existing code.

2. **Metric naming**: Internal metrics use `agis_` prefix. Existing metrics in main.go should be migrated to `internal/metrics` to avoid duplication.

3. **slog migration**: The internal logging package uses Go 1.21+ slog. Legacy `log.Printf` calls can be gradually replaced with the structured logger.

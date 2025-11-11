//go:build ignore
// +build ignore

package docs

import (
	"agis-bot/internal/api"
	"agis-bot/internal/bot/commands"
	"context"
	"log"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// This file shows the exact changes needed in main.go
// Apply these changes to integrate REST API and Scheduler

// ============================================
// STEP 1: Add imports at the top of main.go
// ============================================
// Add these to your import block (around line 13-26):

// The following import block is illustrative only and not compiled.
// import (
//     // ... existing imports ...
//     "agis-bot/internal/api"                    // ADD THIS
//     "agis-bot/internal/services/scheduler"     // ADD THIS
//     "github.com/gorilla/mux"                   // ADD THIS (for API routing)
// )

// ============================================
// STEP 2: Add scheduler metrics (after line 108, with other metrics)
// ============================================

var (
    // ... existing metrics ...
    
    // ADD THESE - Scheduler metrics
    schedulerActiveSchedules = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "agis_scheduler_active_schedules",
            Help: "Number of active server schedules",
        },
    )
    schedulerExecutionsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "agis_scheduler_executions_total",
            Help: "Total scheduler executions",
        },
        []string{"action", "status"}, // action=start/stop/restart, status=success/error
    )
    
    // ADD THESE - API metrics
    apiRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "agis_api_requests_total",
            Help: "Total REST API requests",
        },
        []string{"method", "endpoint", "status"},
    )
    apiRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "agis_api_request_duration_seconds",
            Help:    "API request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )
)

// ============================================
// STEP 3: Register new metrics (around line 143, after existing prometheus.MustRegister calls)
// ============================================

func main() {
    // ... existing code ...
    
    // Register ad conversion metrics
    prometheus.MustRegister(adConversionsTotal)
    prometheus.MustRegister(adRewardsTotal)
    prometheus.MustRegister(adFraudAttemptsTotal)
    prometheus.MustRegister(adCallbackLatency)
    prometheus.MustRegister(adConversionsByTier)
    
    // ADD THESE - Register scheduler and API metrics
    prometheus.MustRegister(schedulerActiveSchedules)
    prometheus.MustRegister(schedulerExecutionsTotal)
    prometheus.MustRegister(apiRequestsTotal)
    prometheus.MustRegister(apiRequestDuration)

// ============================================
// STEP 4: Initialize Scheduler Service
// ============================================
// Add this AFTER commandHandler initialization (around line 377)
// and BEFORE session.Open() (around line 419)

    // Initialize modular command handler
    commandHandler = commands.NewCommandHandler(cfg, dbService, loggingService)
    log.Println("✅ Modular command system initialized")
    
    // ADD THIS BLOCK - Initialize scheduler service
    var schedulerService *scheduler.SchedulerService
    if commandHandler.EnhancedService() != nil {
        schedulerService = scheduler.NewSchedulerService(
            dbService.DB(),
            commandHandler.EnhancedService(),
        )
        if err := schedulerService.Start(); err != nil {
            log.Printf("⚠️ Failed to start scheduler: %v", err)
        } else {
            log.Println("✅ Scheduler service started")
            // Update scheduler to pass to command context
            commandHandler.SetScheduler(schedulerService)
        }
    } else {
        log.Println("⚠️ Enhanced server service not available - scheduler disabled")
    }

// ============================================
// STEP 5: Initialize REST API Server  
// ============================================
// Add this AFTER scheduler initialization

    // ADD THIS BLOCK - Initialize REST API server
    if commandHandler.EnhancedService() != nil && commandHandler.Agones() != nil {
        apiPort := os.Getenv("API_PORT")
        if apiPort == "" {
            apiPort = "8080"
        }
        
        apiServer := api.NewAPIServer(
            ":"+apiPort,
            dbService,
            commandHandler.Agones(),
            commandHandler.EnhancedService(),
        )
        
        // Set metrics collectors
        apiServer.SetMetrics(apiRequestsTotal, apiRequestDuration)
        
        go func() {
            log.Printf("🚀 Starting REST API server on :%s", apiPort)
            if err := apiServer.Start(); err != nil {
                log.Printf("⚠️ Failed to start API server: %v", err)
            }
        }()
        
        log.Println("✅ REST API v1 initialized")
    } else {
        log.Println("⚠️ REST API disabled - missing required services")
    }

// ============================================
// STEP 6: Graceful Shutdown for New Services
// ============================================
// Modify the shutdown section (around line 459) to include cleanup:

    log.Println("🛑 Agis bot shutting down...")

    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // ADD THIS - Stop scheduler
    if schedulerService != nil {
        schedulerService.Stop()
        log.Println("✅ Scheduler service stopped")
    }

    // Shutdown HTTP server
    if err := httpServer.Stop(ctx); err != nil {
        log.Printf("HTTP server shutdown error: %v", err)
    }
    
    // Note: API server will shutdown automatically when main context ends
}

// ============================================
// COMPLETE! Your main.go is now integrated.
// ============================================

// Summary of changes:
// 1. Added 2 imports (api, scheduler)
// 2. Added 4 new Prometheus metrics (2 scheduler, 2 API)
// 3. Initialized scheduler service with error handling
// 4. Initialized REST API server on configurable port
// 5. Added graceful shutdown for scheduler
//
// Next steps:
// 1. Update CommandHandler to support SetScheduler() method
// 2. Run database migration
// 3. Test locally
// 4. Deploy to dev environment

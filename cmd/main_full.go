package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agis-bot/internal/config"
	"agis-bot/internal/services"
	httpserver "agis-bot/internal/http"
)

func main() {
	log.Println("🚀 Starting AGIS Bot v2.0 - Production Enhancement Edition")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	log.Printf("📊 Environment: %s", os.Getenv("SENTRY_ENVIRONMENT"))
	log.Printf("🔐 Discord Token: %s", maskToken(cfg.Discord.Token))
	log.Printf("💾 Database: %s", cfg.Database.Host)
	log.Printf("📡 Metrics Port: %d", cfg.MetricsPort)

	// Initialize Error Monitoring (Sentry)
	errorMonitor := services.NewErrorMonitor(os.Getenv("SENTRY_DSN"), os.Getenv("SENTRY_ENVIRONMENT"))
	defer errorMonitor.Flush(5 * time.Second)
	log.Println("✅ Error monitoring initialized")

	// Initialize Database
	dbService, err := services.NewDatabaseService(cfg)
	if err != nil {
		errorMonitor.CaptureError(err)
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer dbService.Close()

	// Run database migrations
	if !dbService.LocalMode() {
		log.Println("🔧 Running database migrations...")
		if err := runMigrations(dbService.DB()); err != nil {
			errorMonitor.TrackDatabaseError(err)
			log.Printf("⚠️  Migration warning: %v", err)
		}
	}

	log.Println("✅ Database initialized")

	// Ensure database indexes exist for performance
	if !dbService.LocalMode() {
		log.Println("📇 Ensuring database indexes...")
		if err := services.EnsureIndexes(dbService.DB()); err != nil {
			log.Printf("⚠️  Index creation warning: %v", err)
		}
		log.Println("✅ Database indexes ensured")
	}

	// Initialize Agones Client (if configured)
	var agonesClient *services.AgonesClient
	if cfg.Agones.AllocatorEndpoint != "" {
		agonesClient, err = services.NewAgonesClient(cfg)
		if err != nil {
			log.Printf("⚠️  Agones client not available: %v", err)
		} else {
			log.Println("✅ Agones client initialized")
		}
	} else {
		log.Println("ℹ️  Agones not configured - running in local mode")
	}

	// Initialize Notification Service
	notificationService := services.NewNotificationService(cfg)
	log.Println("✅ Notification service initialized")

	// Initialize Ad Metrics Collector
	adMetrics := services.NewAdMetrics()
	log.Println("✅ Ad metrics collector initialized")

	// Initialize Consent Service
	consentService := services.NewConsentService(dbService.DB())
	log.Println("✅ Consent service initialized")

	// Initialize Reward Algorithm
	rewardAlgorithm := services.NewRewardAlgorithm(dbService, consentService)
	log.Println("✅ Reward algorithm initialized")

	// Initialize Ad Conversion Service
	ayetAPIKey := os.Getenv("AYET_API_KEY")
	ayetCallbackToken := os.Getenv("AYET_CALLBACK_TOKEN")
	adConversionService := services.NewAdConversionService(
		dbService,
		rewardAlgorithm,
		adMetrics,
		consentService,
		ayetAPIKey,
		ayetCallbackToken,
	)
	log.Println("✅ Ad conversion service initialized")

	// Initialize A/B Testing Service
	abTestingService := services.NewABTestingService()
	log.Println("✅ A/B testing service initialized")

	// Initialize Guild Provisioning Service
	guildProvisioningService := services.NewGuildProvisioningService(
		dbService.DB(),
		agonesClient,
		dbService,
		notificationService,
	)
	log.Println("✅ Guild provisioning service initialized")

	// Initialize HTTP Server
	httpServer := httpserver.NewServer(cfg.MetricsPort)
	
	// Set up ayeT callback handler
	ayetHandler := httpserver.NewAyetHandler(adConversionService, adMetrics, errorMonitor)
	httpServer.SetAyetHandler(ayetHandler)
	log.Println("✅ HTTP server initialized")

	// Start HTTP server in goroutine
	go func() {
		log.Printf("🌐 Starting HTTP server on :%d", cfg.MetricsPort)
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			errorMonitor.CaptureError(err)
			log.Fatalf("❌ HTTP server error: %v", err)
		}
	}()

	// Initialize Discord Bot (stub for now - full implementation pending)
	log.Println("🤖 Initializing Discord bot...")
	// TODO: Initialize Discord bot with command handlers
	// bot := discord.NewBot(cfg, dbService, agonesClient, adConversionService, abTestingService, guildProvisioningService)
	// bot.Start()
	log.Println("⚠️  Discord bot initialization pending - HTTP endpoints active")

	// Wait for interrupt signal
	log.Println("✅ AGIS Bot is running - Press Ctrl+C to stop")
	
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("🛑 Shutting down gracefully...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("⚠️  HTTP server shutdown error: %v", err)
	}

	log.Println("👋 AGIS Bot stopped")
}

// runMigrations applies database migrations
func runMigrations(db *services.DB) error {
	// For now, just log - migrations should be applied via kubectl/psql
	log.Println("ℹ️  Migrations should be applied manually via deployments/migrations/v2.0-production-enhancements.sql")
	return nil
}

// maskToken masks sensitive tokens for logging
func maskToken(token string) string {
	if len(token) < 8 {
		return "***"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

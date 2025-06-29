package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agis-bot/internal/bot/commands"
	"agis-bot/internal/config"
	"agis-bot/internal/http"
	"agis-bot/internal/services"

	"github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Prometheus metrics
var (
	commandsExecuted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agis_commands_total",
			Help: "Total number of Discord commands executed",
		},
		[]string{"command", "user_id"},
	)
	serversManaged = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agis_game_servers_total",
			Help: "Number of game servers managed by Agis",
		},
		[]string{"game_type", "status"},
	)
	creditsTransactions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agis_credits_transactions_total",
			Help: "Total number of credit transactions",
		},
		[]string{"transaction_type", "user_id"},
	)
	activeUsers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "agis_active_users_total",
			Help: "Number of active users in the system",
		},
	)
	databaseOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agis_database_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "table"},
	)
)

func main() {
	// Load configuration from .env file
	cfg := config.Load()

	// Register Prometheus metrics
	prometheus.MustRegister(commandsExecuted)
	prometheus.MustRegister(serversManaged)
	prometheus.MustRegister(creditsTransactions)
	prometheus.MustRegister(activeUsers)
	prometheus.MustRegister(databaseOperations)

	// Start HTTP server with metrics, health checks, and info endpoints
	httpServer := http.NewServer()
	go func() {
		if err := httpServer.Start(); err != nil {
			log.Printf("Failed to start HTTP server: %v", err)
		}
	}()

	// Initialize Kubernetes client (optional)
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		// Running inside cluster
		config, err := rest.InClusterConfig()
		if err != nil {
			log.Printf("⚠️ Failed to get in-cluster config: %v", err)
		} else {
			_, err = kubernetes.NewForConfig(config)
			if err != nil {
				log.Printf("⚠️ Failed to create Kubernetes client: %v", err)
			} else {
				log.Println("✅ Connected to Kubernetes cluster")
			}
		}
	} else {
		// Try to use local kubeconfig
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err := kubeConfig.ClientConfig()
		if err != nil {
			log.Printf("⚠️ Failed to load Kubernetes config: %v (continuing without K8s)", err)
		} else {
			_, err = kubernetes.NewForConfig(config)
			if err != nil {
				log.Printf("⚠️ Failed to create Kubernetes client: %v", err)
			} else {
				log.Println("✅ Connected to Kubernetes cluster")
			}
		}
	}

	// Get Discord token
	token := cfg.Discord.Token
	if token == "" {
		log.Fatal("❌ DISCORD_TOKEN is required")
	}

	// Create Discord session
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Failed to create Discord session: %v", err)
	}

	// Initialize database service
	dbService, err := services.NewDatabaseService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database service: %v", err)
	}
	log.Println("✅ Database service initialized")

	// Initialize logging service
	loggingService := services.NewLoggingService(dbService, session, "") // Guild ID will be set later
	log.Println("✅ Logging service initialized")

	// Load log channels from environment variables
	loggingService.LoadChannelConfigFromEnv()

	// Start log rotation (rotate every 24 hours, keep logs for 30 days)
	loggingService.StartLogRotation(24*time.Hour, 30*24*time.Hour)

	// Initialize cleanup service
	cleanupService := services.NewCleanupService(dbService, loggingService)
	go cleanupService.Start()
	log.Println("✅ Cleanup service started")

	// Initialize modular command handler
	commandHandler := commands.NewCommandHandler(cfg, dbService, loggingService)
	log.Println("✅ Modular command system initialized")

	// Register event handlers
	session.AddHandler(commandHandler.HandleMessage)
	session.AddHandler(ready)

	// Set bot intents - include message content intent for reading message content and guild state
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentMessageContent | discordgo.IntentsGuilds

	// Open connection
	err = session.Open()
	if err != nil {
		log.Printf("⚠️ Failed to open Discord session: %v (continuing without Discord)", err)
		// Continue without Discord for testing metrics
	} else {
		// Set Discord session for notification service
		commandHandler.SetDiscordSession(session)
		defer func() {
			if err := session.Close(); err != nil {
				log.Printf("Error closing Discord session: %v", err)
			}
		}()
	}

	log.Println("🤖 Agis bot is running! Press Ctrl+C to exit.")

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("🛑 Agis bot shutting down...")
	
	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Shutdown HTTP server
	if err := httpServer.Stop(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("✅ Agis bot logged in as %s", event.User.Username)

	// Set bot status
	err := s.UpdateGameStatus(0, "🎯 Managing WTG Cluster")
	if err != nil {
		log.Printf("Failed to set bot status: %v", err)
	}
}

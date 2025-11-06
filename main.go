package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agis-bot/internal/bot"
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

// Global variables for ready handler
var (
	commandHandler *commands.CommandHandler
	cfg            *config.Config
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
	cfg = config.Load()

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
	
	// Enable state tracking for member updates (required for BeforeUpdate)
	session.StateEnabled = true
	session.State.TrackMembers = true
	session.State.TrackRoles = true

	// Initialize database service
	dbService, err := services.NewDatabaseService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database service: %v", err)
	}
	log.Println("✅ Database service initialized")

	// Initialize logging service
	loggingService := services.NewLoggingService(dbService, session, "") // Guild ID will be set later
	log.Println("✅ Logging service initialized")

	// Wire ad callback token and handler (credits reward from ayet)
	http.SetAdsCallbackToken(cfg.Ads.AyetCallbackToken)
	http.SetAdsAPIKey(cfg.Ads.AyetAPIKey)
	http.SetAdsLinks(cfg.Ads.OfferwallURL, cfg.Ads.SurveywallURL, cfg.Ads.VideoPlacementID)
	http.OnRewardWithConversion = func(uid string, amount int, conversionID, source string) error {
		err := dbService.ProcessAdConversion(uid, amount, conversionID, source)
		if errors.Is(err, services.ErrDuplicate) {
			return nil
		}
		return err
	}
	// Wire verification API config, Discord session, and logging
	http.SetVerifyAPI(cfg.Roles.VerifyAPISecret, cfg.Discord.GuildID, cfg.Roles.VerifiedRoleID)
	http.SetDiscordSessionForAPI(session)
	http.SetLoggingServiceForAPI(loggingService)

	// Load log channels from environment variables
	loggingService.LoadChannelConfigFromEnv()

	// Start log rotation (rotate every 24 hours, keep logs for 30 days)
	loggingService.StartLogRotation(24*time.Hour, 30*24*time.Hour)

	// Initialize cleanup service
	cleanupService := services.NewCleanupService(dbService, loggingService)
	go cleanupService.Start()
	log.Println("✅ Cleanup service started")

	// Initialize modular command handler
	commandHandler = commands.NewCommandHandler(cfg, dbService, loggingService)
	log.Println("✅ Modular command system initialized")

	// Wire user servers provider for WordPress dashboard API
	http.SetUserServersProvider(func(ctx context.Context, discordID string) ([]http.DashboardServer, error) {
		var (
			servers []*services.GameServer
			err     error
		)
		if commandHandler != nil && commandHandler.EnhancedService() != nil {
			servers, err = commandHandler.EnhancedService().GetUserServersEnhanced(ctx, discordID)
		} else {
			servers, err = dbService.GetUserServers(discordID)
		}
		if err != nil {
			return nil, err
		}
		out := make([]http.DashboardServer, 0, len(servers))
		for _, s := range servers {
			ds := http.DashboardServer{
				ID:        s.ID,
				Name:      s.Name,
				Game:      s.GameType,
				Address:   s.Address,
				Port:      s.Port,
				Status:    s.Status,
				Region:    "",
				Players:   http.PlayersInfo{Current: 0, Max: 0},
				ConnectURL: "",
				ManageURL:  "",
			}
			if !s.CreatedAt.IsZero() {
				ds.CreatedAt = s.CreatedAt.Format(time.RFC3339)
			}
			out = append(out, ds)
		}
		return out, nil
	})

	// Initialize event handlers for verified role protection
	eventHandlers := bot.NewEventHandlers(loggingService, cfg.Roles.VerifiedRoleID, cfg.Discord.GuildID)

	// Register event handlers (message-based)
	session.AddHandler(commandHandler.HandleMessage)
	session.AddHandler(ready)
	session.AddHandler(eventHandlers.HandleGuildMemberUpdate)

	// Register interaction handler
	session.AddHandler(commandHandler.HandleInteraction)

	// Set bot intents - include message content, guild state, and guild members for role monitoring
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentMessageContent | discordgo.IntentsGuilds | discordgo.IntentsGuildMembers

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

	// Register slash commands now that session is authenticated
	if _, err := commandHandler.RegisterSlashCommands(s, cfg.Discord.GuildID); err != nil {
		log.Printf("⚠️ Failed to register slash commands: %v", err)
	} else {
		if cfg.Discord.GuildID != "" {
			log.Println("✅ Registered guild slash commands")
		} else {
			log.Println("✅ Registered global slash commands")
		}
	}
}

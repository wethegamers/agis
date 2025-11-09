package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agis-bot/internal/bot"
	"agis-bot/internal/bot/commands"
	"agis-bot/internal/config"
	"agis-bot/internal/http"
	"agis-bot/internal/payment"
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

	// Ad conversion metrics
	adConversionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agis_ad_conversions_total",
			Help: "Total number of ad conversions processed",
		},
		[]string{"provider", "type", "status"}, // provider=ayet, type=offerwall/surveywall/video, status=completed/fraud
	)
	adRewardsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agis_ad_rewards_total",
			Help: "Total Game Credits rewarded from ad conversions",
		},
		[]string{"provider", "type"},
	)
	adFraudAttemptsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agis_ad_fraud_attempts_total",
			Help: "Total number of detected fraud attempts",
		},
		[]string{"provider", "reason"}, // reason=excessive_velocity/ip_hopping/excessive_earnings
	)
	adCallbackLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "agis_ad_callback_latency_seconds",
			Help:    "Latency of ad callback processing in seconds",
			Buckets: prometheus.DefBuckets, // 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
		},
		[]string{"provider", "status"},
	)
	adConversionsByTier = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agis_ad_conversions_by_tier_total",
			Help: "Ad conversions broken down by user tier",
		},
		[]string{"tier"}, // free/premium/premium_plus
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
	
	// Register ad conversion metrics
	prometheus.MustRegister(adConversionsTotal)
	prometheus.MustRegister(adRewardsTotal)
	prometheus.MustRegister(adFraudAttemptsTotal)
	prometheus.MustRegister(adCallbackLatency)
	prometheus.MustRegister(adConversionsByTier)

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

	// Initialize GDPR consent service (BLOCKER 7)
	consentService := services.NewConsentService(dbService)
	ctx := context.Background()
	if err := consentService.InitSchema(ctx); err != nil {
		log.Printf("⚠️ Failed to initialize consent schema: %v", err)
	}
	// Wire consent checker for HTTP endpoints
	http.SetConsentChecker(consentService)
	log.Println("✅ Consent service initialized")

	// Initialize Ad Conversion service (ayeT-Studios S2S)
	adConversionService := services.NewAdConversionService(dbService, consentService, cfg.Ads.AyetAPIKey, cfg.Ads.AyetCallbackToken)
	if err := adConversionService.InitSchema(ctx); err != nil {
		log.Printf("⚠️ Failed to initialize ad conversions schema: %v", err)
	}
	// Wire ayeT-Studios S2S callback handler
	ayetHandler := http.NewAyetHandler(adConversionService)
	http.SetAyetHandler(ayetHandler)
	
	// Initialize ad metrics collector
	adMetrics := services.NewAdMetrics(
		adConversionsTotal,
		adRewardsTotal,
		adFraudAttemptsTotal,
		adCallbackLatency,
		adConversionsByTier,
	)
	adConversionService.SetMetrics(adMetrics)
	ayetHandler.SetMetrics(adMetrics)
	log.Println("✅ Ad conversion service initialized (ayeT-Studios S2S with Prometheus metrics)")

	// Wire ad callback token and handler (credits reward from ayet - legacy fallback)
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
	// Wire Stripe payment service (BLOCKER 2: Zero-touch payments)
	if os.Getenv("STRIPE_SECRET_KEY") != "" {
		stripeSecretKey := os.Getenv("STRIPE_SECRET_KEY")
		stripeWebhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
		stripeSuccessURL := os.Getenv("STRIPE_SUCCESS_URL")
		stripeCancelURL := os.Getenv("STRIPE_CANCEL_URL")
		stripeTestMode := os.Getenv("STRIPE_TEST_MODE") == "true"

		// Import payment package
		stripeService := payment.NewStripeService(
			stripeSecretKey,
			stripeWebhookSecret,
			stripeSuccessURL,
			stripeCancelURL,
			stripeTestMode,
		)

		// Wire webhook callback for automatic WTG fulfillment
		http.SetStripeService(stripeService, func(discordID string, wtgCoins int, sessionID string, amountPaid int64) error {
			log.Printf("💰 Processing payment: User %s purchased %d WTG for $%.2f (session: %s)",
				discordID, wtgCoins, float64(amountPaid)/100, sessionID)

			// Begin database transaction
			tx, err := dbService.DB().Begin()
			if err != nil {
				return fmt.Errorf("failed to start transaction: %v", err)
			}
			defer tx.Rollback()

			// Add WTG coins to user account
			_, err = tx.Exec(`
				INSERT INTO users (discord_id, wtg_coins, credits)
				VALUES ($1, $2, 0)
				ON CONFLICT (discord_id) 
				DO UPDATE SET wtg_coins = users.wtg_coins + $2
			`, discordID, wtgCoins)

			if err != nil {
				return fmt.Errorf("failed to add WTG coins: %v", err)
			}

			// Log transaction for audit trail
			_, err = tx.Exec(`
				INSERT INTO credit_transactions (
					from_user, to_user, amount, transaction_type, description, currency_type
				) VALUES (
					'STRIPE', $1, $2, 'purchase', $3, 'WTG'
				)
			`, discordID, wtgCoins, fmt.Sprintf("Stripe payment $%.2f - Session %s", float64(amountPaid)/100, sessionID))

			if err != nil {
				return fmt.Errorf("failed to log transaction: %v", err)
			}

			// Commit transaction
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit transaction: %v", err)
			}

			log.Printf("✅ Successfully credited %d WTG to user %s", wtgCoins, discordID)

			// Send Discord DM notification (best effort, don't fail payment on DM error)
			if session != nil {
				channel, err := session.UserChannelCreate(discordID)
				if err == nil {
					session.ChannelMessageSend(channel.ID, fmt.Sprintf(
						"💎 **Payment Successful!**\\n\\n"+
							"You've received **%d WTG Coins**!\\n"+
							"Amount paid: $%.2f\\n\\n"+
							"Use `credits` to see your balance or `convert` to turn WTG into GameCredits!",
						wtgCoins, float64(amountPaid)/100,
					))
				}
			}

			return nil
		})

		log.Printf("✅ Stripe payment service initialized (Test Mode: %v)", stripeTestMode)
	} else {
		log.Println("⚠️ Stripe not configured - payments disabled (set STRIPE_SECRET_KEY to enable)")
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
	
	// Register ad analytics command (requires AdConversionService)
	adAnalyticsCmd := commands.NewAdAnalyticsCommand(adConversionService)
	commandHandler.Register(adAnalyticsCmd)
	log.Println("✅ Ad analytics command registered")

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
		
		// Initialize role sync service after Discord connection (sync every 10 minutes)
		if cfg.Roles.VerifiedRoleID != "" && cfg.Discord.GuildID != "" {
			roleSyncService := services.NewRoleSyncService(dbService.DB(), session, cfg.Discord.GuildID, cfg.Roles.VerifiedRoleID, 10*time.Minute)
			go roleSyncService.Start()
		}
		
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

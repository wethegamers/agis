// Package hotconfig provides hot-reloadable configuration for agis-bot.
// Configuration is loaded from YAML files mounted from Kubernetes ConfigMaps
// and automatically reloaded when files change.
package hotconfig

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// Config holds all hot-reloadable configuration
type Config struct {
	// Games defines available game types and their configurations
	Games map[string]GameConfig `yaml:"games"`

	// Pricing defines dynamic pricing rules
	Pricing PricingConfig `yaml:"pricing"`

	// Commands defines custom scripted commands
	Commands map[string]CustomCommand `yaml:"commands"`

	// Features toggles for feature flags
	Features FeatureFlags `yaml:"features"`

	// RateLimits defines rate limiting configuration
	RateLimits RateLimitConfig `yaml:"rate_limits"`

	// Messages defines customizable bot messages
	Messages MessageConfig `yaml:"messages"`
}

// GameConfig defines a game server type configuration
type GameConfig struct {
	// Name is the display name of the game
	Name string `yaml:"name"`

	// Enabled controls whether this game can be created
	Enabled bool `yaml:"enabled"`

	// Description shown to users
	Description string `yaml:"description"`

	// Image is the container image for this game
	Image string `yaml:"image"`

	// ImageTag overrides the default tag
	ImageTag string `yaml:"image_tag,omitempty"`

	// Resources defines K8s resource requirements
	Resources ResourceConfig `yaml:"resources"`

	// Ports defines network ports
	Ports []PortConfig `yaml:"ports"`

	// Environment variables for the container
	Environment map[string]string `yaml:"environment,omitempty"`

	// DefaultSlots is the default player count
	DefaultSlots int `yaml:"default_slots"`

	// MaxSlots is the maximum allowed player count
	MaxSlots int `yaml:"max_slots"`

	// Tier classification (solo, moderate, premium, titan)
	Tier string `yaml:"tier"`

	// BaseCostPerHour in GameCredits
	BaseCostPerHour int `yaml:"base_cost_per_hour"`

	// WarmPoolSize is how many pre-allocated servers to keep
	WarmPoolSize int `yaml:"warm_pool_size"`

	// GracePeriodMinutes before stopped server is cleaned up
	GracePeriodMinutes int `yaml:"grace_period_minutes"`

	// RequiresGuild if true, only guild treasury can pay
	RequiresGuild bool `yaml:"requires_guild"`

	// AgonesFleetName for Agones integration
	AgonesFleetName string `yaml:"agones_fleet_name,omitempty"`
}

// ResourceConfig defines Kubernetes resource requirements
type ResourceConfig struct {
	RequestsCPU    string `yaml:"requests_cpu"`
	RequestsMemory string `yaml:"requests_memory"`
	LimitsCPU      string `yaml:"limits_cpu"`
	LimitsMemory   string `yaml:"limits_memory"`
}

// PortConfig defines a network port
type PortConfig struct {
	Name       string `yaml:"name"`
	Port       int    `yaml:"port"`
	Protocol   string `yaml:"protocol"` // TCP, UDP
	TargetPort int    `yaml:"target_port,omitempty"`
}

// PricingConfig defines dynamic pricing rules
type PricingConfig struct {
	// BaseMultiplier applied to all prices
	BaseMultiplier float64 `yaml:"base_multiplier"`

	// PremiumDiscount percentage for premium users (0.0 - 1.0)
	PremiumDiscount float64 `yaml:"premium_discount"`

	// GuildDiscount percentage for guild-owned servers
	GuildDiscount float64 `yaml:"guild_discount"`

	// PeakHours defines peak pricing windows
	PeakHours []PeakHourConfig `yaml:"peak_hours,omitempty"`

	// Promotions defines active promotions
	Promotions []PromotionConfig `yaml:"promotions,omitempty"`
}

// PeakHourConfig defines peak pricing periods
type PeakHourConfig struct {
	Name       string  `yaml:"name"`
	StartHour  int     `yaml:"start_hour"` // 0-23
	EndHour    int     `yaml:"end_hour"`   // 0-23
	Multiplier float64 `yaml:"multiplier"` // e.g., 1.2 for 20% increase
	Days       []int   `yaml:"days"`       // 0=Sunday, 6=Saturday
}

// PromotionConfig defines a promotional pricing period
type PromotionConfig struct {
	Name      string    `yaml:"name"`
	Games     []string  `yaml:"games"`    // Empty = all games
	Discount  float64   `yaml:"discount"` // 0.0 - 1.0
	StartDate time.Time `yaml:"start_date"`
	EndDate   time.Time `yaml:"end_date"`
	Code      string    `yaml:"code,omitempty"` // Optional promo code
}

// CustomCommand defines a user-defined command
type CustomCommand struct {
	// Name is the command trigger
	Name string `yaml:"name"`

	// Enabled controls if command is active
	Enabled bool `yaml:"enabled"`

	// Description shown in help
	Description string `yaml:"description"`

	// Type is the command type: "embed", "text", "script"
	Type string `yaml:"type"`

	// Permission level required
	Permission string `yaml:"permission"` // user, verified, mod, admin, owner

	// Response for simple text/embed commands
	Response *CommandResponse `yaml:"response,omitempty"`

	// Script for scripted commands (Tengo)
	Script string `yaml:"script,omitempty"`

	// Cooldown in seconds
	Cooldown int `yaml:"cooldown,omitempty"`

	// Aliases for the command
	Aliases []string `yaml:"aliases,omitempty"`
}

// CommandResponse defines a command response
type CommandResponse struct {
	// Text is plain text response
	Text string `yaml:"text,omitempty"`

	// Embed defines a Discord embed
	Embed *EmbedConfig `yaml:"embed,omitempty"`
}

// EmbedConfig defines a Discord embed
type EmbedConfig struct {
	Title       string             `yaml:"title,omitempty"`
	Description string             `yaml:"description,omitempty"`
	Color       int                `yaml:"color,omitempty"`
	URL         string             `yaml:"url,omitempty"`
	Fields      []EmbedFieldConfig `yaml:"fields,omitempty"`
	Footer      string             `yaml:"footer,omitempty"`
	Thumbnail   string             `yaml:"thumbnail,omitempty"`
	Image       string             `yaml:"image,omitempty"`
}

// EmbedFieldConfig defines an embed field
type EmbedFieldConfig struct {
	Name   string `yaml:"name"`
	Value  string `yaml:"value"`
	Inline bool   `yaml:"inline,omitempty"`
}

// FeatureFlags defines toggleable features
type FeatureFlags struct {
	// Economy features
	DailyBonus     bool `yaml:"daily_bonus"`
	WorkCommand    bool `yaml:"work_command"`
	AdRewards      bool `yaml:"ad_rewards"`
	GiftCredits    bool `yaml:"gift_credits"`
	CreditTransfer bool `yaml:"credit_transfer"`

	// Server features
	ServerCreation   bool `yaml:"server_creation"`
	ServerBackups    bool `yaml:"server_backups"`
	ServerScheduling bool `yaml:"server_scheduling"`
	PublicLobby      bool `yaml:"public_lobby"`

	// Premium features
	StripePayments bool `yaml:"stripe_payments"`
	Subscriptions  bool `yaml:"subscriptions"`

	// Guild features
	GuildSystem   bool `yaml:"guild_system"`
	GuildTreasury bool `yaml:"guild_treasury"`

	// Admin features
	ClusterCommands bool `yaml:"cluster_commands"`

	// Debug/Dev
	DebugMode      bool `yaml:"debug_mode"`
	VerboseLogging bool `yaml:"verbose_logging"`
}

// RateLimitConfig defines rate limiting
type RateLimitConfig struct {
	// CommandsPerMinute per user
	CommandsPerMinute int `yaml:"commands_per_minute"`

	// ServerCreationsPerHour per user
	ServerCreationsPerHour int `yaml:"server_creations_per_hour"`

	// GiftsPerDay per user
	GiftsPerDay int `yaml:"gifts_per_day"`

	// MaxActiveServers per user
	MaxActiveServers int `yaml:"max_active_servers"`

	// PremiumMaxActiveServers for premium users
	PremiumMaxActiveServers int `yaml:"premium_max_active_servers"`
}

// MessageConfig defines customizable messages
type MessageConfig struct {
	Welcome             string `yaml:"welcome"`
	InsufficientCredits string `yaml:"insufficient_credits"`
	ServerCreated       string `yaml:"server_created"`
	ServerStopped       string `yaml:"server_stopped"`
	ServerDeleted       string `yaml:"server_deleted"`
	DailyCollected      string `yaml:"daily_collected"`
	WorkCompleted       string `yaml:"work_completed"`
	CooldownActive      string `yaml:"cooldown_active"`
	PermissionDenied    string `yaml:"permission_denied"`
	MaintenanceMode     string `yaml:"maintenance_mode"`
}

// Manager handles hot-reloadable configuration
type Manager struct {
	configPath string
	config     atomic.Value // *Config
	watcher    *fsnotify.Watcher
	mu         sync.RWMutex
	callbacks  []func(*Config)
	stopCh     chan struct{}
}

// NewManager creates a new hot config manager
func NewManager(configPath string) (*Manager, error) {
	m := &Manager{
		configPath: configPath,
		callbacks:  make([]func(*Config), 0),
		stopCh:     make(chan struct{}),
	}

	// Load initial configuration
	if err := m.load(); err != nil {
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}

	return m, nil
}

// Start begins watching for configuration changes
func (m *Manager) Start() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	m.watcher = watcher

	// Watch the config directory (for ConfigMap updates which replace the symlink)
	configDir := filepath.Dir(m.configPath)
	if err := watcher.Add(configDir); err != nil {
		return fmt.Errorf("failed to watch config directory: %w", err)
	}

	go m.watchLoop()
	log.Printf("✅ Hot config manager started, watching: %s", configDir)
	return nil
}

// Stop stops the config watcher
func (m *Manager) Stop() {
	close(m.stopCh)
	if m.watcher != nil {
		m.watcher.Close()
	}
}

// Get returns the current configuration
func (m *Manager) Get() *Config {
	return m.config.Load().(*Config)
}

// OnReload registers a callback for config reloads
func (m *Manager) OnReload(fn func(*Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, fn)
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse config YAML: %w", err)
	}

	// Apply defaults
	m.applyDefaults(cfg)

	// Validate
	if err := m.validate(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	m.config.Store(cfg)
	log.Printf("📜 Loaded hot config: %d games, %d custom commands", len(cfg.Games), len(cfg.Commands))
	return nil
}

func (m *Manager) applyDefaults(cfg *Config) {
	// Default pricing multiplier
	if cfg.Pricing.BaseMultiplier == 0 {
		cfg.Pricing.BaseMultiplier = 1.0
	}

	// Default rate limits
	if cfg.RateLimits.CommandsPerMinute == 0 {
		cfg.RateLimits.CommandsPerMinute = 30
	}
	if cfg.RateLimits.ServerCreationsPerHour == 0 {
		cfg.RateLimits.ServerCreationsPerHour = 5
	}
	if cfg.RateLimits.GiftsPerDay == 0 {
		cfg.RateLimits.GiftsPerDay = 10
	}
	if cfg.RateLimits.MaxActiveServers == 0 {
		cfg.RateLimits.MaxActiveServers = 3
	}
	if cfg.RateLimits.PremiumMaxActiveServers == 0 {
		cfg.RateLimits.PremiumMaxActiveServers = 10
	}

	// Default game values
	for name, game := range cfg.Games {
		if game.GracePeriodMinutes == 0 {
			game.GracePeriodMinutes = 120 // 2 hours default
		}
		if game.DefaultSlots == 0 {
			game.DefaultSlots = 10
		}
		if game.MaxSlots == 0 {
			game.MaxSlots = 32
		}
		cfg.Games[name] = game
	}
}

func (m *Manager) validate(cfg *Config) error {
	// Validate games
	for name, game := range cfg.Games {
		if game.Name == "" {
			return fmt.Errorf("game %s: name is required", name)
		}
		if game.BaseCostPerHour <= 0 {
			return fmt.Errorf("game %s: base_cost_per_hour must be positive", name)
		}
		if game.Image == "" {
			return fmt.Errorf("game %s: image is required", name)
		}
	}

	// Validate pricing
	if cfg.Pricing.PremiumDiscount < 0 || cfg.Pricing.PremiumDiscount > 1 {
		return fmt.Errorf("pricing: premium_discount must be between 0 and 1")
	}
	if cfg.Pricing.GuildDiscount < 0 || cfg.Pricing.GuildDiscount > 1 {
		return fmt.Errorf("pricing: guild_discount must be between 0 and 1")
	}

	// Validate custom commands
	for name, cmd := range cfg.Commands {
		if cmd.Type == "" {
			return fmt.Errorf("command %s: type is required", name)
		}
		validTypes := map[string]bool{"embed": true, "text": true, "script": true}
		if !validTypes[cmd.Type] {
			return fmt.Errorf("command %s: invalid type %s", name, cmd.Type)
		}
		if cmd.Type == "script" && cmd.Script == "" {
			return fmt.Errorf("command %s: script is required for script type", name)
		}
	}

	return nil
}

func (m *Manager) watchLoop() {
	// Debounce reloads (ConfigMap updates can trigger multiple events)
	var debounceTimer *time.Timer
	debounceDelay := 2 * time.Second

	for {
		select {
		case <-m.stopCh:
			return
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			// ConfigMap mounts use symlinks, watch for CREATE and WRITE
			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDelay, func() {
					m.reload()
				})
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("⚠️ Config watcher error: %v", err)
		}
	}
}

func (m *Manager) reload() {
	log.Printf("🔄 Reloading hot config...")
	if err := m.load(); err != nil {
		log.Printf("❌ Failed to reload config: %v (keeping previous config)", err)
		return
	}

	// Notify callbacks
	m.mu.RLock()
	callbacks := make([]func(*Config), len(m.callbacks))
	copy(callbacks, m.callbacks)
	m.mu.RUnlock()

	cfg := m.Get()
	for _, cb := range callbacks {
		go cb(cfg)
	}

	log.Printf("✅ Hot config reloaded successfully")
}

// GetGame returns a game configuration by name
func (m *Manager) GetGame(name string) (GameConfig, bool) {
	cfg := m.Get()
	game, exists := cfg.Games[name]
	return game, exists
}

// GetEnabledGames returns all enabled games
func (m *Manager) GetEnabledGames() map[string]GameConfig {
	cfg := m.Get()
	result := make(map[string]GameConfig)
	for name, game := range cfg.Games {
		if game.Enabled {
			result[name] = game
		}
	}
	return result
}

// GetCustomCommand returns a custom command by name or alias
func (m *Manager) GetCustomCommand(name string) (CustomCommand, bool) {
	cfg := m.Get()

	// Direct lookup
	if cmd, exists := cfg.Commands[name]; exists && cmd.Enabled {
		return cmd, true
	}

	// Alias lookup
	for _, cmd := range cfg.Commands {
		if !cmd.Enabled {
			continue
		}
		for _, alias := range cmd.Aliases {
			if alias == name {
				return cmd, true
			}
		}
	}

	return CustomCommand{}, false
}

// CalculatePrice calculates the effective price for a game
func (m *Manager) CalculatePrice(gameName string, isPremium, isGuild bool) int {
	cfg := m.Get()
	game, exists := cfg.Games[gameName]
	if !exists {
		return 0
	}

	price := float64(game.BaseCostPerHour) * cfg.Pricing.BaseMultiplier

	// Apply discounts
	if isPremium {
		price *= (1 - cfg.Pricing.PremiumDiscount)
	}
	if isGuild {
		price *= (1 - cfg.Pricing.GuildDiscount)
	}

	// Check for active promotions
	now := time.Now()
	for _, promo := range cfg.Pricing.Promotions {
		if now.Before(promo.StartDate) || now.After(promo.EndDate) {
			continue
		}
		// Check if promo applies to this game
		applies := len(promo.Games) == 0
		for _, g := range promo.Games {
			if g == gameName {
				applies = true
				break
			}
		}
		if applies {
			price *= (1 - promo.Discount)
		}
	}

	// Check for peak hours
	hour := now.Hour()
	weekday := int(now.Weekday())
	for _, peak := range cfg.Pricing.PeakHours {
		inHours := (peak.StartHour <= hour && hour < peak.EndHour)
		inDays := len(peak.Days) == 0
		for _, d := range peak.Days {
			if d == weekday {
				inDays = true
				break
			}
		}
		if inHours && inDays {
			price *= peak.Multiplier
		}
	}

	return int(price)
}

// IsFeatureEnabled checks if a feature flag is enabled
func (m *Manager) IsFeatureEnabled(feature string) bool {
	cfg := m.Get()
	switch feature {
	case "daily_bonus":
		return cfg.Features.DailyBonus
	case "work_command":
		return cfg.Features.WorkCommand
	case "ad_rewards":
		return cfg.Features.AdRewards
	case "gift_credits":
		return cfg.Features.GiftCredits
	case "credit_transfer":
		return cfg.Features.CreditTransfer
	case "server_creation":
		return cfg.Features.ServerCreation
	case "server_backups":
		return cfg.Features.ServerBackups
	case "server_scheduling":
		return cfg.Features.ServerScheduling
	case "public_lobby":
		return cfg.Features.PublicLobby
	case "stripe_payments":
		return cfg.Features.StripePayments
	case "subscriptions":
		return cfg.Features.Subscriptions
	case "guild_system":
		return cfg.Features.GuildSystem
	case "guild_treasury":
		return cfg.Features.GuildTreasury
	case "cluster_commands":
		return cfg.Features.ClusterCommands
	case "debug_mode":
		return cfg.Features.DebugMode
	case "verbose_logging":
		return cfg.Features.VerboseLogging
	default:
		return false
	}
}

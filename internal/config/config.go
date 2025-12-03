// Package config provides configuration management for AGIS.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration.
type Config struct {
	// Application
	App AppConfig

	// Discord
	Discord DiscordConfig

	// HTTP Server
	HTTP HTTPConfig

	// Database
	Database DatabaseConfig

	// Feature flags
	Features FeatureConfig

	// Metrics
	Metrics MetricsConfig

	// Logging
	Log LogConfig
}

// AppConfig holds application-level configuration.
type AppConfig struct {
	Name        string
	Version     string
	Environment string
	Debug       bool
}

// DiscordConfig holds Discord bot configuration.
type DiscordConfig struct {
	Token                string
	ApplicationID        string
	PublicKey            string
	AllowedGuilds        []string
	CommandPrefix        string
	ShardCount           int
	DisableWebSocket     bool
	WebSocketReconnect   bool
	WebSocketMaxRetries  int
}

// HTTPConfig holds HTTP server configuration.
type HTTPConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	EnableCORS      bool
	AllowedOrigins  []string
	RateLimitRPS    float64
	RateLimitBurst  int
}

// DatabaseConfig holds database configuration.
type DatabaseConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	Name         string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
}

// FeatureConfig holds feature flags.
type FeatureConfig struct {
	EnablePremium        bool
	EnableAnalytics      bool
	EnableWebSocket      bool
	EnableHealthChecks   bool
	EnableMetrics        bool
	EnableOpenSaaS       bool
	EnableRateLimiting   bool
}

// MetricsConfig holds metrics configuration.
type MetricsConfig struct {
	Enabled   bool
	Path      string
	Namespace string
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level       string
	Format      string // "json" or "text"
	AddSource   bool
	Development bool
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{}
	var errs []string

	// App config
	cfg.App = AppConfig{
		Name:        envString("APP_NAME", "agis-bot"),
		Version:     envString("APP_VERSION", "dev"),
		Environment: envString("APP_ENV", "development"),
		Debug:       envBool("APP_DEBUG", false),
	}

	// Discord config
	cfg.Discord = DiscordConfig{
		Token:               envString("DISCORD_TOKEN", ""),
		ApplicationID:       envString("DISCORD_APPLICATION_ID", ""),
		PublicKey:           envString("DISCORD_PUBLIC_KEY", ""),
		AllowedGuilds:       envStringSlice("DISCORD_ALLOWED_GUILDS", nil),
		CommandPrefix:       envString("DISCORD_COMMAND_PREFIX", "!"),
		ShardCount:          envInt("DISCORD_SHARD_COUNT", 1),
		DisableWebSocket:    envBool("DISCORD_DISABLE_WEBSOCKET", false),
		WebSocketReconnect:  envBool("DISCORD_WEBSOCKET_RECONNECT", true),
		WebSocketMaxRetries: envInt("DISCORD_WEBSOCKET_MAX_RETRIES", 5),
	}

	// Validate required Discord config
	if cfg.Discord.Token == "" {
		errs = append(errs, "DISCORD_TOKEN is required")
	}

	// HTTP config
	cfg.HTTP = HTTPConfig{
		Host:            envString("HTTP_HOST", "0.0.0.0"),
		Port:            envInt("HTTP_PORT", 8080),
		ReadTimeout:     envDuration("HTTP_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:    envDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:     envDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: envDuration("HTTP_SHUTDOWN_TIMEOUT", 30*time.Second),
		EnableCORS:      envBool("HTTP_ENABLE_CORS", true),
		AllowedOrigins:  envStringSlice("HTTP_ALLOWED_ORIGINS", []string{"*"}),
		RateLimitRPS:    envFloat("HTTP_RATE_LIMIT_RPS", 100),
		RateLimitBurst:  envInt("HTTP_RATE_LIMIT_BURST", 200),
	}

	// Database config
	cfg.Database = DatabaseConfig{
		Host:         envString("DB_HOST", "localhost"),
		Port:         envInt("DB_PORT", 5432),
		User:         envString("DB_USER", "agis"),
		Password:     envString("DB_PASSWORD", ""),
		Name:         envString("DB_NAME", "agis"),
		SSLMode:      envString("DB_SSL_MODE", "disable"),
		MaxOpenConns: envInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns: envInt("DB_MAX_IDLE_CONNS", 5),
		MaxLifetime:  envDuration("DB_MAX_LIFETIME", 5*time.Minute),
	}

	// Feature flags
	cfg.Features = FeatureConfig{
		EnablePremium:      envBool("FEATURE_PREMIUM", false),
		EnableAnalytics:    envBool("FEATURE_ANALYTICS", true),
		EnableWebSocket:    envBool("FEATURE_WEBSOCKET", true),
		EnableHealthChecks: envBool("FEATURE_HEALTH_CHECKS", true),
		EnableMetrics:      envBool("FEATURE_METRICS", true),
		EnableOpenSaaS:     envBool("FEATURE_OPENSAAS", true),
		EnableRateLimiting: envBool("FEATURE_RATE_LIMITING", true),
	}

	// Metrics config
	cfg.Metrics = MetricsConfig{
		Enabled:   envBool("METRICS_ENABLED", true),
		Path:      envString("METRICS_PATH", "/metrics"),
		Namespace: envString("METRICS_NAMESPACE", "agis"),
	}

	// Log config
	cfg.Log = LogConfig{
		Level:       envString("LOG_LEVEL", "info"),
		Format:      envString("LOG_FORMAT", "json"),
		AddSource:   envBool("LOG_ADD_SOURCE", false),
		Development: envBool("LOG_DEVELOPMENT", false),
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}

	return cfg, nil
}

// MustLoad loads configuration and panics on error.
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

// DatabaseDSN returns the PostgreSQL connection string.
func (c *DatabaseConfig) DatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

// HTTPAddress returns the full HTTP address.
func (c *HTTPConfig) HTTPAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IsProduction returns true if running in production.
func (c *AppConfig) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if running in development.
func (c *AppConfig) IsDevelopment() bool {
	return c.Environment == "development"
}

// Helper functions for environment variable parsing

func envString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func envInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func envFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func envBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}

func envDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

func envStringSlice(key string, defaultVal []string) []string {
	if val := os.Getenv(key); val != "" {
		parts := strings.Split(val, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultVal
}

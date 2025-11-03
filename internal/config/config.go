package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the bot
type Config struct {
	Discord  DiscordConfig
	Database DatabaseConfig
	Metrics  MetricsConfig
	WTG      WTGConfig
	Roles    RoleConfig
	Ads      AdsConfig
}

type DiscordConfig struct {
	Token    string
	ClientID string
	GuildID  string
}

type DatabaseConfig struct {
	Host     string
	Name     string
	User     string
	Password string
}

type MetricsConfig struct {
	Port string
}

type WTGConfig struct {
	DashboardURL string
}

type RoleConfig struct {
	AdminRoles []string
	ModRoles   []string
}

type AdsConfig struct {
	AyetAPIKey        string
	AyetCallbackToken string
	OfferwallURL      string
	SurveywallURL     string
	VideoPlacementID  string
}

// Load loads configuration from environment variables
func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, using environment variables: %v", err)
	}

	return &Config{
		Discord: DiscordConfig{
			Token:    getEnvOrDefault("DISCORD_TOKEN", ""),
			ClientID: getEnvOrDefault("DISCORD_CLIENT_ID", ""),
			GuildID:  getEnvOrDefault("DISCORD_GUILD_ID", ""),
		},
		Database: DatabaseConfig{
			Host:     getEnvOrDefault("DB_HOST", ""),
			Name:     getEnvOrDefault("DB_NAME", "agis"),
			User:     getEnvOrDefault("DB_USER", "root"),
			Password: getEnvOrDefault("DB_PASSWORD", ""),
		},
		Metrics: MetricsConfig{
			Port: getEnvOrDefault("METRICS_PORT", "9090"),
		},
		WTG: WTGConfig{
			DashboardURL: getEnvOrDefault("WTG_DASHBOARD_URL", "https://dashboard.wethegamers.com"),
		},
		Roles: RoleConfig{
			AdminRoles: parseRoles(getEnvOrDefault("ADMIN_ROLES", "")),
			ModRoles:   parseRoles(getEnvOrDefault("MOD_ROLES", "")),
		},
		Ads: AdsConfig{
			AyetAPIKey:        getEnvOrDefault("AYET_API_KEY", ""),
			AyetCallbackToken: getEnvOrDefault("AYET_CALLBACK_TOKEN", ""),
			OfferwallURL:      getEnvOrDefault("AYET_OFFERWALL_URL", ""),
			SurveywallURL:     getEnvOrDefault("AYET_SURVEYWALL_URL", ""),
			VideoPlacementID:  getEnvOrDefault("AYET_VIDEO_PLACEMENT_ID", ""),
		},
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseRoles(roleStr string) []string {
	if roleStr == "" {
		return []string{}
	}
	roles := strings.Split(roleStr, ",")
	for i, role := range roles {
		roles[i] = strings.TrimSpace(role)
	}
	return roles
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// Package metrics provides Prometheus metrics for AGIS.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Namespace is the prefix for all AGIS metrics.
const Namespace = "agis"

// Command metrics
var (
	CommandsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "commands_total",
			Help:      "Total number of Discord commands executed",
		},
		[]string{"command", "status"},
	)

	CommandDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Name:      "command_duration_seconds",
			Help:      "Command execution duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"command"},
	)
)

// Server metrics
var (
	GameServersTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "game_servers_total",
			Help:      "Number of game servers managed by AGIS",
		},
		[]string{"game_type", "status"},
	)

	ServerOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "server_operations_total",
			Help:      "Total server operations (create, start, stop, delete)",
		},
		[]string{"operation", "game_type", "status"},
	)
)

// Credits/Economy metrics
var (
	CreditsTransactions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "credits_transactions_total",
			Help:      "Total number of credit transactions",
		},
		[]string{"type"},
	)

	CreditsAmount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "credits_amount_total",
			Help:      "Total credits moved by transaction type",
		},
		[]string{"type", "direction"},
	)
)

// User metrics
var (
	ActiveUsers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "active_users_total",
			Help:      "Number of active users in the system",
		},
	)

	UsersByTier = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "users_by_tier",
			Help:      "Number of users by subscription tier",
		},
		[]string{"tier"},
	)
)

// Database metrics
var (
	DatabaseOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "database_operations_total",
			Help:      "Total number of database operations",
		},
		[]string{"operation", "table"},
	)

	DatabaseLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Name:      "database_latency_seconds",
			Help:      "Database operation latency in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	DatabaseConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "database_connections_active",
			Help:      "Number of active database connections",
		},
	)
)

// API metrics
var (
	APIRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "api_requests_total",
			Help:      "Total REST API requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	APIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Name:      "api_request_duration_seconds",
			Help:      "API request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
)

// Ad conversion metrics
var (
	AdConversionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "ad_conversions_total",
			Help:      "Total number of ad conversions processed",
		},
		[]string{"provider", "type", "status"},
	)

	AdRewardsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "ad_rewards_total",
			Help:      "Total Game Credits rewarded from ad conversions",
		},
		[]string{"provider", "type"},
	)

	AdFraudAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "ad_fraud_attempts_total",
			Help:      "Total number of detected fraud attempts",
		},
		[]string{"provider", "reason"},
	)

	AdCallbackLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Name:      "ad_callback_latency_seconds",
			Help:      "Latency of ad callback processing in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"provider", "status"},
	)

	AdConversionsByTier = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "ad_conversions_by_tier_total",
			Help:      "Ad conversions broken down by user tier",
		},
		[]string{"tier"},
	)
)

// Scheduler metrics
var (
	SchedulerActiveSchedules = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "scheduler_active_schedules",
			Help:      "Number of active server schedules",
		},
	)

	SchedulerExecutionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "scheduler_executions_total",
			Help:      "Total scheduler executions",
		},
		[]string{"action", "status"},
	)
)

// Build info metric
var BuildInfo = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "build_info",
		Help:      "Build information",
	},
	[]string{"version", "commit", "build_date"},
)

// SetBuildInfo sets the build info metric.
func SetBuildInfo(version, commit, buildDate string) {
	BuildInfo.WithLabelValues(version, commit, buildDate).Set(1)
}

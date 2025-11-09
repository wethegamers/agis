package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

// EnsureIndexes creates all necessary database indexes for optimal performance
func EnsureIndexes(ctx context.Context, db *sql.DB) error {
	if db == nil {
		log.Println("📄 Database indexes skipped (local mode)")
		return nil
	}

	log.Println("🔍 Ensuring database indexes for performance...")

	indexes := []struct {
		name  string
		query string
	}{
		// Users table indexes
		{
			name: "idx_users_discord_id",
			query: `CREATE INDEX IF NOT EXISTS idx_users_discord_id ON users(discord_id)`,
		},
		{
			name: "idx_users_tier",
			query: `CREATE INDEX IF NOT EXISTS idx_users_tier ON users(tier)`,
		},
		{
			name: "idx_users_join_date",
			query: `CREATE INDEX IF NOT EXISTS idx_users_join_date ON users(join_date)`,
		},

		// Game servers table indexes
		{
			name: "idx_game_servers_discord_id",
			query: `CREATE INDEX IF NOT EXISTS idx_game_servers_discord_id ON game_servers(discord_id)`,
		},
		{
			name: "idx_game_servers_status",
			query: `CREATE INDEX IF NOT EXISTS idx_game_servers_status ON game_servers(status)`,
		},
		{
			name: "idx_game_servers_user_status",
			query: `CREATE INDEX IF NOT EXISTS idx_game_servers_user_status ON game_servers(discord_id, status)`,
		},
		{
			name: "idx_game_servers_game_type",
			query: `CREATE INDEX IF NOT EXISTS idx_game_servers_game_type ON game_servers(game_type)`,
		},
		{
			name: "idx_game_servers_created_at",
			query: `CREATE INDEX IF NOT EXISTS idx_game_servers_created_at ON game_servers(created_at)`,
		},

		// Ad conversions indexes (already created in schema, but ensure they exist)
		{
			name: "idx_ad_conversions_discord_id",
			query: `CREATE INDEX IF NOT EXISTS idx_ad_conversions_discord_id ON ad_conversions(discord_id)`,
		},
		{
			name: "idx_ad_conversions_conversion_id",
			query: `CREATE INDEX IF NOT EXISTS idx_ad_conversions_conversion_id ON ad_conversions(conversion_id)`,
		},
		{
			name: "idx_ad_conversions_created_at",
			query: `CREATE INDEX IF NOT EXISTS idx_ad_conversions_created_at ON ad_conversions(created_at)`,
		},
		{
			name: "idx_ad_conversions_status",
			query: `CREATE INDEX IF NOT EXISTS idx_ad_conversions_status ON ad_conversions(status)`,
		},
		{
			name: "idx_ad_conversions_user_created",
			query: `CREATE INDEX IF NOT EXISTS idx_ad_conversions_user_created ON ad_conversions(discord_id, created_at DESC)`,
		},
		{
			name: "idx_ad_conversions_status_created",
			query: `CREATE INDEX IF NOT EXISTS idx_ad_conversions_status_created ON ad_conversions(status, created_at DESC)`,
		},

		// Consent records indexes
		{
			name: "idx_consent_records_user_id",
			query: `CREATE INDEX IF NOT EXISTS idx_consent_records_user_id ON consent_records(user_id)`,
		},
		{
			name: "idx_consent_records_consented",
			query: `CREATE INDEX IF NOT EXISTS idx_consent_records_consented ON consent_records(consented)`,
		},
		{
			name: "idx_consent_records_ip_country",
			query: `CREATE INDEX IF NOT EXISTS idx_consent_records_ip_country ON consent_records(ip_country)`,
		},
		{
			name: "idx_consent_records_consent_timestamp",
			query: `CREATE INDEX IF NOT EXISTS idx_consent_records_consent_timestamp ON consent_records(consent_timestamp)`,
		},
		{
			name: "idx_consent_records_withdrawn_timestamp",
			query: `CREATE INDEX IF NOT EXISTS idx_consent_records_withdrawn_timestamp ON consent_records(withdrawn_timestamp) WHERE withdrawn_timestamp IS NOT NULL`,
		},

		// Subscriptions indexes
		{
			name: "idx_subscriptions_discord_id",
			query: `CREATE INDEX IF NOT EXISTS idx_subscriptions_discord_id ON subscriptions(discord_id)`,
		},
		{
			name: "idx_subscriptions_status",
			query: `CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status)`,
		},
		{
			name: "idx_subscriptions_user_status",
			query: `CREATE INDEX IF NOT EXISTS idx_subscriptions_user_status ON subscriptions(discord_id, status)`,
		},
		{
			name: "idx_subscriptions_stripe_subscription_id",
			query: `CREATE INDEX IF NOT EXISTS idx_subscriptions_stripe_subscription_id ON subscriptions(stripe_subscription_id)`,
		},
		{
			name: "idx_subscriptions_current_period_end",
			query: `CREATE INDEX IF NOT EXISTS idx_subscriptions_current_period_end ON subscriptions(current_period_end)`,
		},

		// Command usage indexes
		{
			name: "idx_command_usage_discord_id",
			query: `CREATE INDEX IF NOT EXISTS idx_command_usage_discord_id ON command_usage(discord_id)`,
		},
		{
			name: "idx_command_usage_command",
			query: `CREATE INDEX IF NOT EXISTS idx_command_usage_command ON command_usage(command)`,
		},
		{
			name: "idx_command_usage_used_at",
			query: `CREATE INDEX IF NOT EXISTS idx_command_usage_used_at ON command_usage(used_at DESC)`,
		},

		// Credit transactions indexes
		{
			name: "idx_credit_transactions_from_user",
			query: `CREATE INDEX IF NOT EXISTS idx_credit_transactions_from_user ON credit_transactions(from_user)`,
		},
		{
			name: "idx_credit_transactions_to_user",
			query: `CREATE INDEX IF NOT EXISTS idx_credit_transactions_to_user ON credit_transactions(to_user)`,
		},
		{
			name: "idx_credit_transactions_type",
			query: `CREATE INDEX IF NOT EXISTS idx_credit_transactions_type ON credit_transactions(transaction_type)`,
		},
		{
			name: "idx_credit_transactions_timestamp",
			query: `CREATE INDEX IF NOT EXISTS idx_credit_transactions_timestamp ON credit_transactions(timestamp DESC)`,
		},

		// Public servers indexes
		{
			name: "idx_public_servers_game_type",
			query: `CREATE INDEX IF NOT EXISTS idx_public_servers_game_type ON public_servers(game_type)`,
		},
		{
			name: "idx_public_servers_owner_id",
			query: `CREATE INDEX IF NOT EXISTS idx_public_servers_owner_id ON public_servers(owner_id)`,
		},
		{
			name: "idx_public_servers_added_at",
			query: `CREATE INDEX IF NOT EXISTS idx_public_servers_added_at ON public_servers(added_at DESC)`,
		},

		// Bot roles indexes
		{
			name: "idx_bot_roles_role_type",
			query: `CREATE INDEX IF NOT EXISTS idx_bot_roles_role_type ON bot_roles(role_type)`,
		},
		{
			name: "idx_bot_roles_guild_id",
			query: `CREATE INDEX IF NOT EXISTS idx_bot_roles_guild_id ON bot_roles(guild_id)`,
		},

		// Guild treasury indexes (if table exists)
		{
			name: "idx_guild_treasury_guild_id",
			query: `CREATE INDEX IF NOT EXISTS idx_guild_treasury_guild_id ON guild_treasury(guild_id)`,
		},
		{
			name: "idx_guild_treasury_transactions_guild",
			query: `CREATE INDEX IF NOT EXISTS idx_guild_treasury_transactions_guild ON guild_treasury_transactions(guild_id)`,
		},
		{
			name: "idx_guild_treasury_transactions_timestamp",
			query: `CREATE INDEX IF NOT EXISTS idx_guild_treasury_transactions_timestamp ON guild_treasury_transactions(timestamp DESC)`,
		},

		// Server reviews indexes (if table exists)
		{
			name: "idx_server_reviews_server_id",
			query: `CREATE INDEX IF NOT EXISTS idx_server_reviews_server_id ON server_reviews(server_id)`,
		},
		{
			name: "idx_server_reviews_reviewer_id",
			query: `CREATE INDEX IF NOT EXISTS idx_server_reviews_reviewer_id ON server_reviews(reviewer_id)`,
		},
		{
			name: "idx_server_reviews_rating",
			query: `CREATE INDEX IF NOT EXISTS idx_server_reviews_rating ON server_reviews(rating)`,
		},
	}

	successCount := 0
	for _, idx := range indexes {
		if _, err := db.ExecContext(ctx, idx.query); err != nil {
			// Log error but continue - table might not exist yet
			log.Printf("⚠️ Failed to create index %s: %v", idx.name, err)
		} else {
			successCount++
		}
	}

	log.Printf("✅ Database indexes ensured: %d/%d successful", successCount, len(indexes))
	return nil
}

// AnalyzePerformance provides query performance analysis recommendations
func AnalyzePerformance(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return nil
	}

	log.Println("📊 Analyzing database performance...")

	// Check for missing indexes on frequently queried columns
	var unusedIndexes []string
	rows, err := db.QueryContext(ctx, `
		SELECT schemaname, tablename, indexname
		FROM pg_stat_user_indexes
		WHERE idx_scan = 0
		AND indexname NOT LIKE '%_pkey'
		ORDER BY schemaname, tablename, indexname
	`)
	if err != nil {
		log.Printf("⚠️ Could not analyze unused indexes: %v", err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table, index string
		if err := rows.Scan(&schema, &table, &index); err != nil {
			continue
		}
		unusedIndexes = append(unusedIndexes, fmt.Sprintf("%s.%s.%s", schema, table, index))
	}

	if len(unusedIndexes) > 0 {
		log.Printf("⚠️ Found %d unused indexes (consider removing if persistent)", len(unusedIndexes))
	} else {
		log.Println("✅ All indexes are being used")
	}

	return nil
}

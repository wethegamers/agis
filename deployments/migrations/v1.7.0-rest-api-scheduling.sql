-- Migration: v1.7.0 - REST API and Server Scheduling
-- Date: 2025-11-10

-- Server schedules table
CREATE TABLE IF NOT EXISTS server_schedules (
    id SERIAL PRIMARY KEY,
    server_id INTEGER NOT NULL REFERENCES game_servers(id) ON DELETE CASCADE,
    discord_id VARCHAR(32) NOT NULL REFERENCES users(discord_id),
    action VARCHAR(20) NOT NULL CHECK (action IN ('start', 'stop', 'restart')),
    cron_expression VARCHAR(100) NOT NULL,
    timezone VARCHAR(50) DEFAULT 'UTC',
    enabled BOOLEAN DEFAULT true,
    last_run TIMESTAMP,
    next_run TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for efficient schedule queries
CREATE INDEX IF NOT EXISTS idx_schedules_next_run ON server_schedules(next_run) WHERE enabled = true;
CREATE INDEX IF NOT EXISTS idx_schedules_discord_id ON server_schedules(discord_id);
CREATE INDEX IF NOT EXISTS idx_schedules_server_id ON server_schedules(server_id);

-- API keys table for REST API authentication
CREATE TABLE IF NOT EXISTS api_keys (
    id SERIAL PRIMARY KEY,
    key_hash VARCHAR(128) UNIQUE NOT NULL,
    discord_id VARCHAR(32) NOT NULL REFERENCES users(discord_id),
    name VARCHAR(100) NOT NULL,
    scopes TEXT[] DEFAULT '{"read:servers"}',
    rate_limit INTEGER DEFAULT 100,
    last_used TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_discord_id ON api_keys(discord_id);

-- Add WTG coins column to users if not exists (for shop integration)
ALTER TABLE users ADD COLUMN IF NOT EXISTS wtg_coins INTEGER DEFAULT 0;

-- Update user_stats table with more detailed analytics
CREATE TABLE IF NOT EXISTS user_stats (
    discord_id VARCHAR(32) PRIMARY KEY REFERENCES users(discord_id),
    total_servers_created INTEGER DEFAULT 0,
    total_commands_used INTEGER DEFAULT 0,
    total_credits_earned INTEGER DEFAULT 0,
    total_credits_spent INTEGER DEFAULT 0,
    total_ad_conversions INTEGER DEFAULT 0,
    total_wtg_purchased INTEGER DEFAULT 0,
    last_command_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Function to update user stats automatically
CREATE OR REPLACE FUNCTION update_user_stats_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for user_stats
DROP TRIGGER IF EXISTS trigger_update_user_stats ON user_stats;
CREATE TRIGGER trigger_update_user_stats
BEFORE UPDATE ON user_stats
FOR EACH ROW
EXECUTE FUNCTION update_user_stats_timestamp();

-- Add indexes for analytics queries
CREATE INDEX IF NOT EXISTS idx_users_tier ON users(tier);
CREATE INDEX IF NOT EXISTS idx_users_credits ON users(credits DESC);
CREATE INDEX IF NOT EXISTS idx_users_servers_used ON users(servers_used DESC);

-- Comments for documentation
COMMENT ON TABLE server_schedules IS 'Stores cron-based schedules for automatic server management';
COMMENT ON TABLE api_keys IS 'API keys for REST API v1.7.0 authentication';
COMMENT ON TABLE user_stats IS 'Detailed analytics for user activity and spending';

-- Grant permissions (adjust based on your user)
GRANT SELECT, INSERT, UPDATE, DELETE ON server_schedules TO agis_dev_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON api_keys TO agis_dev_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON user_stats TO agis_dev_user;
GRANT USAGE, SELECT ON SEQUENCE server_schedules_id_seq TO agis_dev_user;
GRANT USAGE, SELECT ON SEQUENCE api_keys_id_seq TO agis_dev_user;

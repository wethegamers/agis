-- Migration v2.0: Production Enhancement Features
-- Adds support for guild provisioning, A/B testing, and enhanced ad conversions

-- ============================================================================
-- Guild Provisioning Tables
-- ============================================================================

-- Guild treasury table (if not already exists from v4.0)
CREATE TABLE IF NOT EXISTS guild_treasury (
    guild_id VARCHAR(32) PRIMARY KEY,
    balance INTEGER DEFAULT 0 CHECK (balance >= 0),
    total_earned INTEGER DEFAULT 0,
    total_spent INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Treasury transactions log
CREATE TABLE IF NOT EXISTS treasury_transactions (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(32) NOT NULL,
    amount INTEGER NOT NULL,
    transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('credit', 'debit')),
    description TEXT,
    source VARCHAR(50), -- 'member_contribution', 'server_provisioning', 'admin_grant', etc.
    created_by VARCHAR(32), -- Discord user ID
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (guild_id) REFERENCES guild_treasury(guild_id) ON DELETE CASCADE
);

-- Index for treasury transaction queries
CREATE INDEX IF NOT EXISTS idx_treasury_transactions_guild_id ON treasury_transactions(guild_id);
CREATE INDEX IF NOT EXISTS idx_treasury_transactions_created_at ON treasury_transactions(created_at DESC);

-- Server provisioning requests
CREATE TABLE IF NOT EXISTS server_provision_requests (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(32) NOT NULL,
    requested_by VARCHAR(32) NOT NULL, -- Discord user ID
    template_id VARCHAR(50) NOT NULL,
    server_name VARCHAR(100) NOT NULL,
    duration_hours INTEGER NOT NULL CHECK (duration_hours > 0),
    auto_renew BOOLEAN DEFAULT FALSE,
    requested_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    approved_at TIMESTAMP,
    approved_by VARCHAR(32), -- Admin who approved
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'provisioning', 'active', 'terminated', 'failed')),
    server_id VARCHAR(100), -- Agones GameServer ID
    termination_scheduled_at TIMESTAMP,
    total_cost INTEGER DEFAULT 0,
    notes TEXT,
    FOREIGN KEY (guild_id) REFERENCES guild_treasury(guild_id) ON DELETE CASCADE
);

-- Indexes for provisioning queries
CREATE INDEX IF NOT EXISTS idx_provision_requests_guild_id ON server_provision_requests(guild_id);
CREATE INDEX IF NOT EXISTS idx_provision_requests_status ON server_provision_requests(status);
CREATE INDEX IF NOT EXISTS idx_provision_requests_server_id ON server_provision_requests(server_id);
CREATE INDEX IF NOT EXISTS idx_provision_requests_requested_at ON server_provision_requests(requested_at DESC);

-- ============================================================================
-- A/B Testing Tables
-- ============================================================================

-- Experiment configurations
CREATE TABLE IF NOT EXISTS ab_experiments (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    traffic_alloc DECIMAL(3,2) NOT NULL CHECK (traffic_alloc >= 0 AND traffic_alloc <= 1),
    target_metric VARCHAR(50) NOT NULL,
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'running', 'paused', 'completed', 'archived')),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(32) -- Admin who created experiment
);

-- Experiment variants
CREATE TABLE IF NOT EXISTS ab_variants (
    id SERIAL PRIMARY KEY,
    experiment_id VARCHAR(100) NOT NULL,
    variant_id VARCHAR(50) NOT NULL,
    variant_name VARCHAR(100) NOT NULL,
    allocation DECIMAL(3,2) NOT NULL CHECK (allocation >= 0 AND allocation <= 1),
    config JSONB NOT NULL, -- Variant-specific configuration
    description TEXT,
    FOREIGN KEY (experiment_id) REFERENCES ab_experiments(id) ON DELETE CASCADE,
    UNIQUE(experiment_id, variant_id)
);

-- User assignments (sticky)
CREATE TABLE IF NOT EXISTS ab_assignments (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(32) NOT NULL,
    experiment_id VARCHAR(100) NOT NULL,
    variant_id VARCHAR(50) NOT NULL,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    sticky BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (experiment_id) REFERENCES ab_experiments(id) ON DELETE CASCADE,
    UNIQUE(user_id, experiment_id)
);

-- Experiment events (for metrics)
CREATE TABLE IF NOT EXISTS ab_events (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(32) NOT NULL,
    experiment_id VARCHAR(100) NOT NULL,
    variant_id VARCHAR(50) NOT NULL,
    event_type VARCHAR(50) NOT NULL, -- 'conversion', 'revenue', 'reward', 'fraud', etc.
    event_value DECIMAL(10,2) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (experiment_id) REFERENCES ab_experiments(id) ON DELETE CASCADE
);

-- Indexes for A/B testing queries
CREATE INDEX IF NOT EXISTS idx_ab_experiments_status ON ab_experiments(status);
CREATE INDEX IF NOT EXISTS idx_ab_experiments_dates ON ab_experiments(start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_ab_assignments_user_experiment ON ab_assignments(user_id, experiment_id);
CREATE INDEX IF NOT EXISTS idx_ab_events_experiment_variant ON ab_events(experiment_id, variant_id);
CREATE INDEX IF NOT EXISTS idx_ab_events_created_at ON ab_events(created_at DESC);

-- ============================================================================
-- Enhanced Ad Conversions (upgrade existing table if needed)
-- ============================================================================

-- Add columns if they don't exist (for backwards compatibility)
DO $$ 
BEGIN
    -- Check if ad_conversions table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'ad_conversions') THEN
        -- Add new columns if they don't exist
        ALTER TABLE ad_conversions 
            ADD COLUMN IF NOT EXISTS provider VARCHAR(50) DEFAULT 'ayet',
            ADD COLUMN IF NOT EXISTS type VARCHAR(50),
            ADD COLUMN IF NOT EXISTS multiplier DECIMAL(3,2) DEFAULT 1.0,
            ADD COLUMN IF NOT EXISTS ip_address VARCHAR(45),
            ADD COLUMN IF NOT EXISTS user_agent TEXT,
            ADD COLUMN IF NOT EXISTS processed_at TIMESTAMP,
            ADD COLUMN IF NOT EXISTS fraud_reason TEXT;
        
        -- Add indexes if they don't exist
        CREATE INDEX IF NOT EXISTS idx_ad_conversions_discord_id ON ad_conversions(discord_id);
        CREATE INDEX IF NOT EXISTS idx_ad_conversions_created_at ON ad_conversions(created_at DESC);
        CREATE INDEX IF NOT EXISTS idx_ad_conversions_status ON ad_conversions(status);
        CREATE INDEX IF NOT EXISTS idx_ad_conversions_provider_type ON ad_conversions(provider, type);
    END IF;
END $$;

-- ============================================================================
-- Consent Management (GDPR compliance)
-- ============================================================================

CREATE TABLE IF NOT EXISTS consent_records (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(32) NOT NULL,
    consent_type VARCHAR(50) NOT NULL, -- 'ads', 'analytics', 'personalization'
    consented BOOLEAN NOT NULL DEFAULT FALSE,
    ip_address VARCHAR(45),
    ip_country VARCHAR(2), -- ISO country code
    consent_text TEXT, -- What the user agreed to
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for consent queries
CREATE INDEX IF NOT EXISTS idx_consent_user_type ON consent_records(user_id, consent_type);
CREATE INDEX IF NOT EXISTS idx_consent_updated_at ON consent_records(updated_at DESC);

-- ============================================================================
-- Subscriptions (Premium/Premium Plus tiers)
-- ============================================================================

CREATE TABLE IF NOT EXISTS subscriptions (
    id SERIAL PRIMARY KEY,
    discord_id VARCHAR(32) NOT NULL,
    tier VARCHAR(20) NOT NULL CHECK (tier IN ('premium', 'premium_plus')),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'canceled', 'expired', 'paused')),
    stripe_subscription_id VARCHAR(100) UNIQUE,
    stripe_customer_id VARCHAR(100),
    current_period_start TIMESTAMP NOT NULL,
    current_period_end TIMESTAMP NOT NULL,
    cancel_at_period_end BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (discord_id) REFERENCES users(discord_id) ON DELETE CASCADE
);

-- Indexes for subscription queries
CREATE INDEX IF NOT EXISTS idx_subscriptions_discord_id ON subscriptions(discord_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_subscriptions_stripe_subscription_id ON subscriptions(stripe_subscription_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_period_end ON subscriptions(current_period_end);

-- ============================================================================
-- Server Templates (for guild provisioning)
-- ============================================================================

CREATE TABLE IF NOT EXISTS server_templates (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    game_type VARCHAR(50) NOT NULL,
    size VARCHAR(20) NOT NULL, -- 'small', 'medium', 'large'
    cost_per_hour INTEGER NOT NULL CHECK (cost_per_hour > 0),
    setup_cost INTEGER NOT NULL CHECK (setup_cost >= 0),
    max_players INTEGER NOT NULL,
    cpu_request VARCHAR(20) NOT NULL, -- e.g., '1000m'
    memory_request VARCHAR(20) NOT NULL, -- e.g., '2Gi'
    description TEXT,
    docker_image VARCHAR(200),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default templates
INSERT INTO server_templates (id, name, game_type, size, cost_per_hour, setup_cost, max_players, cpu_request, memory_request, description, docker_image)
VALUES 
    ('minecraft-small', 'Minecraft (Small)', 'minecraft', 'small', 100, 500, 10, '1000m', '2Gi', 'Small Minecraft server for up to 10 players', 'itzg/minecraft-server:latest'),
    ('minecraft-medium', 'Minecraft (Medium)', 'minecraft', 'medium', 200, 1000, 25, '2000m', '4Gi', 'Medium Minecraft server for up to 25 players', 'itzg/minecraft-server:latest'),
    ('minecraft-large', 'Minecraft (Large)', 'minecraft', 'large', 400, 2000, 50, '4000m', '8Gi', 'Large Minecraft server for up to 50 players', 'itzg/minecraft-server:latest'),
    ('valheim-small', 'Valheim (Small)', 'valheim', 'small', 150, 750, 10, '1500m', '3Gi', 'Small Valheim server for up to 10 players', 'mbround18/valheim:latest'),
    ('palworld-small', 'Palworld (Small)', 'palworld', 'small', 200, 1000, 16, '2000m', '4Gi', 'Small Palworld server for up to 16 players', 'thijsvanloef/palworld-server-docker:latest')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- Views for Analytics
-- ============================================================================

-- View: Guild treasury summary
CREATE OR REPLACE VIEW guild_treasury_summary AS
SELECT 
    gt.guild_id,
    gt.balance,
    gt.total_earned,
    gt.total_spent,
    COUNT(DISTINCT spr.id) AS active_servers,
    SUM(CASE WHEN spr.status = 'active' THEN st.cost_per_hour ELSE 0 END) AS hourly_cost,
    gt.updated_at
FROM guild_treasury gt
LEFT JOIN server_provision_requests spr ON gt.guild_id = spr.guild_id AND spr.status = 'active'
LEFT JOIN server_templates st ON spr.template_id = st.id
GROUP BY gt.guild_id, gt.balance, gt.total_earned, gt.total_spent, gt.updated_at;

-- View: A/B experiment results summary
CREATE OR REPLACE VIEW ab_experiment_results AS
SELECT 
    e.id AS experiment_id,
    e.name AS experiment_name,
    e.status,
    v.variant_id,
    v.variant_name,
    COUNT(DISTINCT a.user_id) AS sample_size,
    COUNT(CASE WHEN ev.event_type = 'conversion' THEN 1 END) AS conversions,
    AVG(CASE WHEN ev.event_type = 'reward' THEN ev.event_value END) AS avg_reward,
    SUM(CASE WHEN ev.event_type = 'revenue' THEN ev.event_value ELSE 0 END) AS total_revenue,
    COUNT(CASE WHEN ev.event_type = 'fraud' THEN 1 END) AS fraud_count
FROM ab_experiments e
JOIN ab_variants v ON e.id = v.experiment_id
LEFT JOIN ab_assignments a ON e.id = a.experiment_id AND v.variant_id = a.variant_id
LEFT JOIN ab_events ev ON a.user_id = ev.user_id AND a.experiment_id = ev.experiment_id AND a.variant_id = ev.variant_id
GROUP BY e.id, e.name, e.status, v.variant_id, v.variant_name;

-- View: Ad conversion analytics
CREATE OR REPLACE VIEW ad_conversion_analytics AS
SELECT 
    DATE_TRUNC('day', created_at) AS conversion_date,
    provider,
    type,
    status,
    COUNT(*) AS conversion_count,
    SUM(amount) AS total_amount,
    AVG(amount) AS avg_amount,
    COUNT(DISTINCT discord_id) AS unique_users,
    COUNT(CASE WHEN status = 'fraud' THEN 1 END) AS fraud_count
FROM ad_conversions
GROUP BY DATE_TRUNC('day', created_at), provider, type, status;

-- ============================================================================
-- Functions
-- ============================================================================

-- Function: Update treasury balance
CREATE OR REPLACE FUNCTION update_treasury_balance()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE guild_treasury
    SET 
        balance = balance + (CASE WHEN NEW.transaction_type = 'credit' THEN NEW.amount ELSE -NEW.amount END),
        total_earned = total_earned + (CASE WHEN NEW.transaction_type = 'credit' THEN NEW.amount ELSE 0 END),
        total_spent = total_spent + (CASE WHEN NEW.transaction_type = 'debit' THEN NEW.amount ELSE 0 END),
        updated_at = NOW()
    WHERE guild_id = NEW.guild_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger: Update treasury on transaction insert
DROP TRIGGER IF EXISTS treasury_transaction_trigger ON treasury_transactions;
CREATE TRIGGER treasury_transaction_trigger
AFTER INSERT ON treasury_transactions
FOR EACH ROW
EXECUTE FUNCTION update_treasury_balance();

-- Function: Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at trigger to relevant tables
DROP TRIGGER IF EXISTS update_ab_experiments_updated_at ON ab_experiments;
CREATE TRIGGER update_ab_experiments_updated_at
BEFORE UPDATE ON ab_experiments
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_guild_treasury_updated_at ON guild_treasury;
CREATE TRIGGER update_guild_treasury_updated_at
BEFORE UPDATE ON guild_treasury
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_subscriptions_updated_at ON subscriptions;
CREATE TRIGGER update_subscriptions_updated_at
BEFORE UPDATE ON subscriptions
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- Migration Complete
-- ============================================================================

-- Version tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(20) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO schema_migrations (version) VALUES ('v2.0-production-enhancements')
ON CONFLICT (version) DO NOTHING;

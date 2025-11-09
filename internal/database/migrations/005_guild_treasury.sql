-- Guild Treasury System Migration
-- Enables shared guild wallets for server funding (BLOCKER 4)
-- Blue Ocean Strategy: Guild pooling enables Titan servers impossible for competitors

-- Guild Treasury: Shared wallet for guild server funding
CREATE TABLE IF NOT EXISTS guild_treasury (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) UNIQUE NOT NULL,  -- Discord Guild ID or custom identifier
    guild_name VARCHAR(255) NOT NULL,
    owner_id VARCHAR(255) NOT NULL,         -- Discord ID of guild creator
    balance INT DEFAULT 0 CHECK (balance >= 0),  -- GameCredits balance (non-refundable)
    total_deposits INT DEFAULT 0,           -- All-time deposits for analytics
    total_spent INT DEFAULT 0,              -- All-time spending for analytics
    member_count INT DEFAULT 0,             -- Current member count
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(discord_id) ON DELETE CASCADE
);

CREATE INDEX idx_guild_treasury_guild_id ON guild_treasury(guild_id);
CREATE INDEX idx_guild_treasury_owner_id ON guild_treasury(owner_id);

-- Guild Members: Tracks individual contributions and roles
CREATE TABLE IF NOT EXISTS guild_members (
    guild_id VARCHAR(255) NOT NULL,
    discord_id VARCHAR(255) NOT NULL,
    total_deposits INT DEFAULT 0,           -- Lifetime contributions to treasury
    last_deposit TIMESTAMP,                 -- Last deposit timestamp
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    role VARCHAR(20) DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member')),
    PRIMARY KEY (guild_id, discord_id),
    FOREIGN KEY (guild_id) REFERENCES guild_treasury(guild_id) ON DELETE CASCADE,
    FOREIGN KEY (discord_id) REFERENCES users(discord_id) ON DELETE CASCADE
);

CREATE INDEX idx_guild_members_discord_id ON guild_members(discord_id);
CREATE INDEX idx_guild_members_total_deposits ON guild_members(total_deposits DESC);

-- Guild Servers: Tracks servers funded by guild treasury
CREATE TABLE IF NOT EXISTS guild_servers (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    server_id INT NOT NULL UNIQUE,          -- References game_servers(id)
    created_by VARCHAR(255) NOT NULL,       -- Discord ID who created server
    cost_per_hour INT NOT NULL,
    hours_funded INT DEFAULT 0,             -- Total hours funded from treasury
    total_spent INT DEFAULT 0,              -- Total GC spent from treasury
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (guild_id) REFERENCES guild_treasury(guild_id) ON DELETE CASCADE,
    FOREIGN KEY (server_id) REFERENCES game_servers(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(discord_id) ON DELETE CASCADE
);

CREATE INDEX idx_guild_servers_guild_id ON guild_servers(guild_id);
CREATE INDEX idx_guild_servers_server_id ON guild_servers(server_id);

-- Comments for documentation
COMMENT ON TABLE guild_treasury IS 'Shared guild wallets for pooled server funding (Blue Ocean strategy)';
COMMENT ON COLUMN guild_treasury.balance IS 'Non-refundable GameCredits balance. Deposits cannot be withdrawn.';
COMMENT ON COLUMN guild_treasury.total_deposits IS 'All-time deposits for contribution leaderboards';
COMMENT ON TABLE guild_members IS 'Guild membership and individual contribution tracking';
COMMENT ON COLUMN guild_members.total_deposits IS 'Lifetime contributions for fairness metrics';
COMMENT ON TABLE guild_servers IS 'Servers funded by guild treasury (e.g., Titan tier ARK servers)';

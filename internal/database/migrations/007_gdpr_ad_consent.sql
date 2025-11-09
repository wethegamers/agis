-- Migration: GDPR Ad Consent Tracking
-- Purpose: Store user consent for ad viewing in compliance with GDPR
-- GDPR Requirements:
--   - Freely given, specific, informed, unambiguous consent
--   - Ability to withdraw consent at any time
--   - Records must prove consent was obtained
--   - Users must be able to access their consent status

CREATE TABLE IF NOT EXISTS user_ad_consent (
    user_id BIGINT PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    consented BOOLEAN NOT NULL DEFAULT FALSE,
    consent_timestamp TIMESTAMPTZ,
    withdrawn_timestamp TIMESTAMPTZ,
    ip_country VARCHAR(2), -- ISO 3166-1 alpha-2 country code
    gdpr_version VARCHAR(20) NOT NULL DEFAULT 'v1.0', -- track consent policy version
    consent_method VARCHAR(50), -- 'discord_command', 'web_dashboard', etc.
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for quick consent checks (hot path for ad commands)
CREATE INDEX idx_user_ad_consent_status ON user_ad_consent(user_id, consented) WHERE consented = TRUE;

-- Index for EU users requiring consent
CREATE INDEX idx_user_ad_consent_country ON user_ad_consent(ip_country);

-- Index for compliance reporting (consent trends over time)
CREATE INDEX idx_user_ad_consent_timestamp ON user_ad_consent(consent_timestamp DESC);

-- Index for analytics (withdrawal tracking)
CREATE INDEX idx_user_ad_consent_withdrawn ON user_ad_consent(withdrawn_timestamp) WHERE withdrawn_timestamp IS NOT NULL;

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_user_ad_consent_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_user_ad_consent_timestamp
BEFORE UPDATE ON user_ad_consent
FOR EACH ROW
EXECUTE FUNCTION update_user_ad_consent_timestamp();

-- Add comment for documentation
COMMENT ON TABLE user_ad_consent IS 'GDPR-compliant ad consent tracking. Users in EEA/UK/CH must explicitly consent before viewing ads.';
COMMENT ON COLUMN user_ad_consent.consented IS 'TRUE if user has given consent, FALSE if explicitly rejected or withdrawn';
COMMENT ON COLUMN user_ad_consent.consent_timestamp IS 'When user gave consent (NULL if never consented or withdrawn)';
COMMENT ON COLUMN user_ad_consent.withdrawn_timestamp IS 'When user withdrew consent (NULL if never withdrawn)';
COMMENT ON COLUMN user_ad_consent.ip_country IS 'Country detected from IP at time of consent (for audit trail)';
COMMENT ON COLUMN user_ad_consent.gdpr_version IS 'Version of consent policy user agreed to (for policy updates)';

#!/bin/bash
set -e

# AGIS Bot v2.0 - Vault Secrets Setup Script
# This script adds all required secrets to Vault for the specified environment

VAULT_ADDR="${VAULT_ADDR:-http://vault.vault.svc.cluster.local:8200}"
VAULT_TOKEN="${VAULT_TOKEN:-hvs.kjP6fT17rS8dnnW7NTZqUOgm}"
ENVIRONMENT="${1:-development}"

echo "🔐 AGIS Bot v2.0 - Vault Secrets Setup"
echo "Environment: $ENVIRONMENT"
echo "Vault Address: $VAULT_ADDR"
echo ""

# Vault path based on environment
VAULT_PATH="secret/$ENVIRONMENT/agis-bot"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Export Vault env vars
export VAULT_ADDR
export VAULT_TOKEN

# Function to write secret to Vault
write_secret() {
    local key=$1
    local value=$2
    local optional=$3
    
    if [ -z "$value" ] || [ "$value" = "PLACEHOLDER" ]; then
        if [ "$optional" = "true" ]; then
            echo -e "${YELLOW}⚠️  Skipping optional secret: $key${NC}"
            return 0
        else
            echo -e "${RED}❌ Missing required secret: $key${NC}"
            return 1
        fi
    fi
    
    # Write to Vault
    vault kv put "$VAULT_PATH" "$key=$value" 2>&1 | grep -q "Success" || {
        echo -e "${RED}❌ Failed to write: $key${NC}"
        return 1
    }
    
    echo -e "${GREEN}✅ Added: $key${NC}"
}

# Function to get secret from Vault (for verification)
get_secret() {
    local key=$1
    vault kv get -field="$key" "$VAULT_PATH" 2>/dev/null || echo ""
}

# Check Vault connectivity
echo "🔍 Checking Vault connectivity..."
if ! vault status &>/dev/null; then
    echo -e "${RED}❌ Cannot connect to Vault at $VAULT_ADDR${NC}"
    echo "Please ensure:"
    echo "  1. Vault is running"
    echo "  2. VAULT_ADDR is correct"
    echo "  3. VAULT_TOKEN is valid"
    exit 1
fi
echo -e "${GREEN}✅ Connected to Vault${NC}"
echo ""

# Prompt for secrets interactively
echo "📝 Please provide the following secrets:"
echo "   (Press Enter to skip optional secrets marked with *)"
echo ""

# === Core Discord & Database ===
echo -e "${YELLOW}=== Core Discord & Database ===${NC}"

read -p "DISCORD_TOKEN (Bot token): " DISCORD_TOKEN
read -p "DISCORD_CLIENT_ID: " DISCORD_CLIENT_ID
read -p "DISCORD_GUILD_ID: " DISCORD_GUILD_ID
read -p "DB_HOST (e.g., postgresql.database.svc.cluster.local): " DB_HOST
read -p "DB_USER (default: root): " DB_USER
DB_USER=${DB_USER:-root}
read -sp "DB_PASSWORD: " DB_PASSWORD
echo ""
read -p "DB_NAME (default: agis): " DB_NAME
DB_NAME=${DB_NAME:-agis}

# === ayeT-Studios Ad Network ===
echo ""
echo -e "${YELLOW}=== ayeT-Studios Ad Network ===${NC}"

read -p "AYET_API_KEY (Production API key): " AYET_API_KEY
read -p "AYET_CALLBACK_TOKEN (Shared secret): " AYET_CALLBACK_TOKEN
read -p "AYET_OFFERWALL_URL*: " AYET_OFFERWALL_URL
read -p "AYET_SURVEYWALL_URL*: " AYET_SURVEYWALL_URL
read -p "AYET_VIDEO_PLACEMENT_ID*: " AYET_VIDEO_PLACEMENT_ID

# === Sentry Error Monitoring ===
echo ""
echo -e "${YELLOW}=== Sentry Error Monitoring ===${NC}"

read -p "SENTRY_DSN (https://...@sentry.io/...): " SENTRY_DSN

# === Discord Webhooks for Alerts ===
echo ""
echo -e "${YELLOW}=== Discord Webhooks for Alerts ===${NC}"
echo "Create 8 webhooks in Discord (Server Settings → Integrations → Webhooks)"
echo "Recommended channels: #alerts-payments, #alerts-ads, #alerts-infra, #alerts-security, #alerts-performance, #alerts-revenue, #alerts-critical, #alerts-compliance"
echo ""

read -p "DISCORD_WEBHOOK_PAYMENTS: " DISCORD_WEBHOOK_PAYMENTS
read -p "DISCORD_WEBHOOK_ADS: " DISCORD_WEBHOOK_ADS
read -p "DISCORD_WEBHOOK_INFRA: " DISCORD_WEBHOOK_INFRA
read -p "DISCORD_WEBHOOK_SECURITY: " DISCORD_WEBHOOK_SECURITY
read -p "DISCORD_WEBHOOK_PERFORMANCE: " DISCORD_WEBHOOK_PERFORMANCE
read -p "DISCORD_WEBHOOK_REVENUE: " DISCORD_WEBHOOK_REVENUE
read -p "DISCORD_WEBHOOK_CRITICAL: " DISCORD_WEBHOOK_CRITICAL
read -p "DISCORD_WEBHOOK_COMPLIANCE: " DISCORD_WEBHOOK_COMPLIANCE

# === Agones Configuration ===
echo ""
echo -e "${YELLOW}=== Agones Configuration (Optional) ===${NC}"

read -p "AGONES_ALLOCATOR_ENDPOINT*: " AGONES_ALLOCATOR_ENDPOINT
read -p "AGONES_ALLOCATOR_TLS* (cert content): " AGONES_ALLOCATOR_TLS
read -p "AGONES_NAMESPACE* (default: game-servers): " AGONES_NAMESPACE
AGONES_NAMESPACE=${AGONES_NAMESPACE:-game-servers}

# === Discord Logging Channels ===
echo ""
echo -e "${YELLOW}=== Discord Logging Channels (Optional) ===${NC}"

read -p "LOG_CHANNEL_GENERAL*: " LOG_CHANNEL_GENERAL
read -p "LOG_CHANNEL_USER*: " LOG_CHANNEL_USER
read -p "LOG_CHANNEL_MOD*: " LOG_CHANNEL_MOD
read -p "LOG_CHANNEL_ERROR*: " LOG_CHANNEL_ERROR
read -p "LOG_CHANNEL_CLEANUP*: " LOG_CHANNEL_CLEANUP
read -p "LOG_CHANNEL_CLUSTER*: " LOG_CHANNEL_CLUSTER
read -p "LOG_CHANNEL_EXPORT*: " LOG_CHANNEL_EXPORT
read -p "LOG_CHANNEL_AUDIT*: " LOG_CHANNEL_AUDIT

# === Additional Optional Secrets ===
echo ""
echo -e "${YELLOW}=== Additional Secrets (Optional) ===${NC}"

read -p "VERIFIED_ROLE_ID*: " VERIFIED_ROLE_ID
read -p "VERIFY_API_SECRET*: " VERIFY_API_SECRET

echo ""
echo "📤 Writing secrets to Vault path: $VAULT_PATH"
echo ""

# Track failures
FAILED_SECRETS=()

# Write all secrets to Vault (combining into single KV)
# Note: Vault KV v2 stores all key-value pairs together

cat > /tmp/vault-secrets.json <<EOF
{
  "DISCORD_TOKEN": "$DISCORD_TOKEN",
  "DISCORD_CLIENT_ID": "$DISCORD_CLIENT_ID",
  "DISCORD_GUILD_ID": "$DISCORD_GUILD_ID",
  "DB_HOST": "$DB_HOST",
  "DB_USER": "$DB_USER",
  "DB_PASSWORD": "$DB_PASSWORD",
  "DB_NAME": "$DB_NAME",
  "AYET_API_KEY": "$AYET_API_KEY",
  "AYET_CALLBACK_TOKEN": "$AYET_CALLBACK_TOKEN",
  "AYET_OFFERWALL_URL": "$AYET_OFFERWALL_URL",
  "AYET_SURVEYWALL_URL": "$AYET_SURVEYWALL_URL",
  "AYET_VIDEO_PLACEMENT_ID": "$AYET_VIDEO_PLACEMENT_ID",
  "SENTRY_DSN": "$SENTRY_DSN",
  "DISCORD_WEBHOOK_PAYMENTS": "$DISCORD_WEBHOOK_PAYMENTS",
  "DISCORD_WEBHOOK_ADS": "$DISCORD_WEBHOOK_ADS",
  "DISCORD_WEBHOOK_INFRA": "$DISCORD_WEBHOOK_INFRA",
  "DISCORD_WEBHOOK_SECURITY": "$DISCORD_WEBHOOK_SECURITY",
  "DISCORD_WEBHOOK_PERFORMANCE": "$DISCORD_WEBHOOK_PERFORMANCE",
  "DISCORD_WEBHOOK_REVENUE": "$DISCORD_WEBHOOK_REVENUE",
  "DISCORD_WEBHOOK_CRITICAL": "$DISCORD_WEBHOOK_CRITICAL",
  "DISCORD_WEBHOOK_COMPLIANCE": "$DISCORD_WEBHOOK_COMPLIANCE",
  "AGONES_ALLOCATOR_ENDPOINT": "$AGONES_ALLOCATOR_ENDPOINT",
  "AGONES_ALLOCATOR_TLS": "$AGONES_ALLOCATOR_TLS",
  "AGONES_NAMESPACE": "$AGONES_NAMESPACE",
  "LOG_CHANNEL_GENERAL": "$LOG_CHANNEL_GENERAL",
  "LOG_CHANNEL_USER": "$LOG_CHANNEL_USER",
  "LOG_CHANNEL_MOD": "$LOG_CHANNEL_MOD",
  "LOG_CHANNEL_ERROR": "$LOG_CHANNEL_ERROR",
  "LOG_CHANNEL_CLEANUP": "$LOG_CHANNEL_CLEANUP",
  "LOG_CHANNEL_CLUSTER": "$LOG_CHANNEL_CLUSTER",
  "LOG_CHANNEL_EXPORT": "$LOG_CHANNEL_EXPORT",
  "LOG_CHANNEL_AUDIT": "$LOG_CHANNEL_AUDIT",
  "VERIFIED_ROLE_ID": "$VERIFIED_ROLE_ID",
  "VERIFY_API_SECRET": "$VERIFY_API_SECRET"
}
EOF

# Write all secrets at once
echo "Writing all secrets to Vault..."
if vault kv put "$VAULT_PATH" @/tmp/vault-secrets.json 2>&1 | grep -q "Success"; then
    echo -e "${GREEN}✅ All secrets written successfully!${NC}"
else
    echo -e "${RED}❌ Failed to write secrets${NC}"
    rm /tmp/vault-secrets.json
    exit 1
fi

# Clean up temp file
rm /tmp/vault-secrets.json

echo ""
echo "✅ Vault setup complete!"
echo ""
echo "📋 Summary:"
echo "  Environment: $ENVIRONMENT"
echo "  Vault Path: $VAULT_PATH"
echo "  Secrets: 33 total"
echo ""
echo "🔍 Verify secrets with:"
echo "  vault kv get $VAULT_PATH"
echo ""
echo "📝 Next steps:"
echo "  1. Verify secrets: vault kv get $VAULT_PATH"
echo "  2. Apply database migrations"
echo "  3. Deploy to Kubernetes: helm upgrade --install agis-bot ./charts/agis-bot -n $ENVIRONMENT"
echo ""

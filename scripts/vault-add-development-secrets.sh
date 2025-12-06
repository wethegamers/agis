#!/bin/bash
set -e

# Quick script to add development secrets to Vault
# Run with: kubectl port-forward -n vault svc/vault 8200:8200 (in another terminal)

export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="${VAULT_TOKEN:-}"  # Set via environment variable for security

echo "🔐 Adding development secrets to Vault"
echo "Vault: $VAULT_ADDR"
echo ""

# Check connectivity
if ! vault status &>/dev/null; then
    echo "❌ Cannot connect to Vault"
    echo "Run in another terminal: kubectl port-forward -n vault svc/vault 8200:8200"
    exit 1
fi

echo "✅ Connected to Vault"

# Development secrets (using placeholders - you'll need to update these)
vault kv put secret/development/agis-bot \
  DISCORD_TOKEN="YOUR_DEV_BOT_TOKEN" \
  DISCORD_CLIENT_ID="YOUR_CLIENT_ID" \
  DISCORD_GUILD_ID="YOUR_GUILD_ID" \
  DB_HOST="postgresql.database.svc.cluster.local" \
  DB_USER="root" \
  DB_PASSWORD="your_db_password" \
  DB_NAME="agis" \
  AYET_API_KEY="sandbox_key_here" \
  AYET_CALLBACK_TOKEN="callback_token_here" \
  AYET_OFFERWALL_URL="https://offerwall-sandbox.example.com" \
  AYET_SURVEYWALL_URL="https://surveywall-sandbox.example.com" \
  AYET_VIDEO_PLACEMENT_ID="placement_id" \
  SENTRY_DSN="https://your_sentry@sentry.io/project" \
  DISCORD_WEBHOOK_PAYMENTS="https://discord.com/api/webhooks/..." \
  DISCORD_WEBHOOK_ADS="https://discord.com/api/webhooks/..." \
  DISCORD_WEBHOOK_INFRA="https://discord.com/api/webhooks/..." \
  DISCORD_WEBHOOK_SECURITY="https://discord.com/api/webhooks/..." \
  DISCORD_WEBHOOK_PERFORMANCE="https://discord.com/api/webhooks/..." \
  DISCORD_WEBHOOK_REVENUE="https://discord.com/api/webhooks/..." \
  DISCORD_WEBHOOK_CRITICAL="https://discord.com/api/webhooks/..." \
  DISCORD_WEBHOOK_COMPLIANCE="https://discord.com/api/webhooks/..." \
  AGONES_ALLOCATOR_ENDPOINT="" \
  AGONES_ALLOCATOR_TLS="" \
  AGONES_NAMESPACE="game-servers" \
  LOG_CHANNEL_GENERAL="" \
  LOG_CHANNEL_USER="" \
  LOG_CHANNEL_MOD="" \
  LOG_CHANNEL_ERROR="" \
  LOG_CHANNEL_CLEANUP="" \
  LOG_CHANNEL_CLUSTER="" \
  LOG_CHANNEL_EXPORT="" \
  LOG_CHANNEL_AUDIT="" \
  VERIFIED_ROLE_ID="" \
  VERIFY_API_SECRET=""

echo ""
echo "✅ Development secrets added!"
echo ""
echo "🔍 Verify with: vault kv get secret/development/agis-bot"
echo ""
echo "⚠️  NOTE: Update placeholder values in Vault UI or re-run with real values"
echo "     Vault UI: http://localhost:8200"
echo ""

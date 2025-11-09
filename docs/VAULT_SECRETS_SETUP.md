# Vault Secrets Setup Guide

This guide walks you through adding all required secrets to Vault for AGIS Bot v2.0.

## Quick Start

### Step 1: Port-Forward to Vault

Open a new terminal and run:

```bash
kubectl port-forward -n vault svc/vault 8200:8200
```

Keep this terminal running.

### Step 2: Add Secrets

In another terminal, cd to the agis-bot directory and run ONE of these options:

#### Option A: Interactive Script (Recommended)

```bash
./scripts/vault-setup-secrets.sh development
```

This will prompt you for all secrets interactively.

#### Option B: Quick Development Setup (Placeholders)

```bash
./scripts/vault-add-development-secrets.sh
```

Then update values in Vault UI at http://localhost:8200

#### Option C: Manual via Vault CLI

```bash
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="hvs.kjP6fT17rS8dnnW7NTZqUOgm"

vault kv put kubefirst/development/agis-bot \
  DISCORD_TOKEN="your_token" \
  DISCORD_CLIENT_ID="your_client_id" \
  # ... etc
```

## Secrets Checklist

### Required Secrets (9)

- [ ] `DISCORD_TOKEN` - Bot token from Discord Developer Portal
- [ ] `DISCORD_CLIENT_ID` - Application ID
- [ ] `DISCORD_GUILD_ID` - Your Discord server ID
- [ ] `DB_HOST` - PostgreSQL host (e.g., `postgresql.database.svc.cluster.local`)
- [ ] `DB_USER` - Database user (default: `root`)
- [ ] `DB_PASSWORD` - Database password
- [ ] `DB_NAME` - Database name (default: `agis`)
- [ ] `AYET_API_KEY` - ayeT-Studios API key
- [ ] `AYET_CALLBACK_TOKEN` - ayeT-Studios callback token

### Monitoring Secrets (9)

- [ ] `SENTRY_DSN` - Sentry.io DSN for error monitoring
- [ ] `DISCORD_WEBHOOK_PAYMENTS` - Discord webhook URL
- [ ] `DISCORD_WEBHOOK_ADS` - Discord webhook URL
- [ ] `DISCORD_WEBHOOK_INFRA` - Discord webhook URL
- [ ] `DISCORD_WEBHOOK_SECURITY` - Discord webhook URL
- [ ] `DISCORD_WEBHOOK_PERFORMANCE` - Discord webhook URL
- [ ] `DISCORD_WEBHOOK_REVENUE` - Discord webhook URL
- [ ] `DISCORD_WEBHOOK_CRITICAL` - Discord webhook URL
- [ ] `DISCORD_WEBHOOK_COMPLIANCE` - Discord webhook URL

### Optional Secrets (15+)

- [ ] `AYET_OFFERWALL_URL` - ayeT offerwall URL
- [ ] `AYET_SURVEYWALL_URL` - ayeT surveywall URL
- [ ] `AYET_VIDEO_PLACEMENT_ID` - ayeT video placement ID
- [ ] `AGONES_ALLOCATOR_ENDPOINT` - Agones allocator endpoint
- [ ] `AGONES_ALLOCATOR_TLS` - Agones TLS cert
- [ ] `AGONES_NAMESPACE` - Kubernetes namespace (default: `game-servers`)
- [ ] `LOG_CHANNEL_GENERAL` - Discord channel ID
- [ ] `LOG_CHANNEL_USER` - Discord channel ID
- [ ] `LOG_CHANNEL_MOD` - Discord channel ID
- [ ] `LOG_CHANNEL_ERROR` - Discord channel ID
- [ ] `LOG_CHANNEL_CLEANUP` - Discord channel ID
- [ ] `LOG_CHANNEL_CLUSTER` - Discord channel ID
- [ ] `LOG_CHANNEL_EXPORT` - Discord channel ID
- [ ] `LOG_CHANNEL_AUDIT` - Discord channel ID
- [ ] `VERIFIED_ROLE_ID` - Discord role ID
- [ ] `VERIFY_API_SECRET` - API secret for verification

## Getting Secret Values

### Discord Token

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Select your application
3. Go to "Bot" section
4. Click "Reset Token" and copy the new token
5. **IMPORTANT**: Save this immediately - you can't see it again!

### Discord Webhooks

Create webhooks for each alert channel:

1. Open Discord server
2. Go to Server Settings → Integrations → Webhooks
3. Create webhook for each channel:
   - `#alerts-payments`
   - `#alerts-ads`
   - `#alerts-infra`
   - `#alerts-security`
   - `#alerts-performance`
   - `#alerts-revenue`
   - `#alerts-critical`
   - `#alerts-compliance`
4. Copy webhook URLs

### Sentry DSN

1. Go to [Sentry.io](https://sentry.io)
2. Create new project (or select existing)
3. Go to Settings → Projects → [Your Project] → Client Keys (DSN)
4. Copy the DSN (format: `https://...@sentry.io/...`)

### Database Connection

For local k3d cluster:

```bash
# Check if PostgreSQL is running
kubectl get pods -n database

# If not, you may need to deploy PostgreSQL
# Or use external database
```

For staging/production, get values from your DBA or cloud provider.

### ayeT-Studios

1. Sign up at [ayeT-Studios](https://www.ayet-studios.com)
2. Create application
3. Get API key and callback token from dashboard
4. Configure S2S callback URL: `https://bot-api.wethegamers.org/ads/ayet/s2s`

## Verifying Secrets

### Via Vault CLI

```bash
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="hvs.kjP6fT17rS8dnnW7NTZqUOgm"

# List all secrets
vault kv get kubefirst/development/agis-bot

# Get specific secret
vault kv get -field=DISCORD_TOKEN kubefirst/development/agis-bot
```

### Via Vault UI

1. Open http://localhost:8200 (with port-forward running)
2. Login with token: `hvs.kjP6fT17rS8dnnW7NTZqUOgm`
3. Navigate to: `kubefirst` → `data` → `development` → `agis-bot`
4. View/edit secrets

## Updating Secrets

### Update Single Secret

```bash
# Get current secrets
vault kv get -format=json kubefirst/development/agis-bot > /tmp/secrets.json

# Edit the secret you want to change
# Then patch (this adds/updates without removing other secrets)
vault kv patch kubefirst/development/agis-bot DISCORD_TOKEN="new_token"
```

### Update Multiple Secrets

```bash
vault kv patch kubefirst/development/agis-bot \
  DISCORD_TOKEN="new_token" \
  DB_PASSWORD="new_password"
```

## Troubleshooting

### "Connection refused" error

**Cause**: Vault port-forward not running

**Fix**:
```bash
kubectl port-forward -n vault svc/vault 8200:8200
```

### "Permission denied" error

**Cause**: Invalid or expired token

**Fix**: Get root token from cluster:
```bash
kubectl -n vault get secrets/vault-unseal-secret \
  --template='{{index .data "root-token"}}' | base64 -d
```

### "Path not found" error

**Cause**: Wrong vault path or secret not created yet

**Fix**: Verify path structure:
- Development: `kubefirst/development/agis-bot`
- Staging: `kubefirst/staging/agis-bot`
- Production: `kubefirst/production/agis-bot`

### Secrets not appearing in pods

**Cause**: ExternalSecrets not syncing

**Fix**:
```bash
# Check ExternalSecrets status
kubectl get externalsecrets -n development

# Check if secret was created
kubectl get secrets -n development agis-bot-secrets

# Force refresh
kubectl annotate externalsecret agis-bot-secrets -n development \
  force-sync="$(date +%s)" --overwrite
```

## Environment-Specific Paths

| Environment | Vault Path |
|-------------|------------|
| Development | `kubefirst/development/agis-bot` |
| Staging | `kubefirst/staging/agis-bot` |
| Production | `kubefirst/production/agis-bot` |

## Security Best Practices

1. **Never commit secrets to git**
   - All secrets are in Vault only
   - `.env` files are gitignored

2. **Use different secrets per environment**
   - Development uses sandbox/test keys
   - Production uses real keys

3. **Rotate secrets regularly**
   - Discord bot token: Every 90 days
   - Database passwords: Every 90 days
   - API keys: As needed

4. **Limit Vault token access**
   - Don't share root token
   - Create per-user tokens with limited scope
   - Use Kubernetes auth for pods

5. **Monitor secret access**
   - Check Vault audit logs
   - Alert on unusual access patterns

## Next Steps

After adding secrets:

1. ✅ Verify secrets in Vault
2. → Apply database migrations (next step)
3. → Deploy to Kubernetes
4. → Verify ExternalSecrets sync
5. → Test bot startup

## Quick Reference

```bash
# Port-forward to Vault
kubectl port-forward -n vault svc/vault 8200:8200

# Set env vars
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="hvs.kjP6fT17rS8dnnW7NTZqUOgm"

# Add secrets
./scripts/vault-add-development-secrets.sh

# Verify
vault kv get kubefirst/development/agis-bot

# Update single secret
vault kv patch kubefirst/development/agis-bot KEY="value"
```

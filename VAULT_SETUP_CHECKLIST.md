# Vault Setup Checklist - Week 1 Step 1

## Current Status

✅ Port-forward to Vault is running (PID 287920)
✅ Vault is accessible at http://localhost:8200
✅ Vault is initialized and unsealed
❌ Need valid Vault token with write access

## Action Required

You need the Vault root token to proceed. Check these locations:

1. **Your password manager** (most likely location)
2. **Initial Vault setup notes** (when cluster was first created)
3. **kubefirst installation output** (saved somewhere)
4. **Ask team member** who initially set up the cluster

## Quick Setup (Once You Have Token)

### Option 1: Use Vault UI (Easiest)

1. Open http://localhost:8200 in browser
2. Login with your root token
3. Navigate to: **Secrets** → **kubefirst** → **+ Create secret**
4. Path: `development/agis-bot`
5. Click **Add** for each secret below
6. Click **Save** when done

### Option 2: Use Script

```bash
cd /home/seb/wtg/agis-bot

# Set your actual token
export VAULT_TOKEN="your_actual_root_token_here"
export VAULT_ADDR="http://localhost:8200"

# Run the script
./scripts/vault-add-development-secrets.sh
```

### Option 3: Manual CLI Commands

```bash
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="your_actual_root_token_here"

# Add all secrets at once
vault kv put kubefirst/development/agis-bot \
  DISCORD_TOKEN="PLACEHOLDER_UPDATE_ME" \
  DISCORD_CLIENT_ID="PLACEHOLDER_UPDATE_ME" \
  DISCORD_GUILD_ID="PLACEHOLDER_UPDATE_ME" \
  DB_HOST="postgresql.database.svc.cluster.local:5432" \
  DB_USER="root" \
  DB_PASSWORD="PLACEHOLDER_UPDATE_ME" \
  DB_NAME="agis" \
  AYET_API_KEY="PLACEHOLDER_UPDATE_ME" \
  AYET_CALLBACK_TOKEN="PLACEHOLDER_UPDATE_ME" \
  AYET_OFFERWALL_URL="https://www.ayet-studios.com/offerwall" \
  AYET_SURVEYWALL_URL="https://www.ayet-studios.com/surveywall" \
  AYET_VIDEO_PLACEMENT_ID="PLACEHOLDER_UPDATE_ME" \
  SENTRY_DSN="PLACEHOLDER_UPDATE_ME" \
  DISCORD_WEBHOOK_PAYMENTS="PLACEHOLDER_UPDATE_ME" \
  DISCORD_WEBHOOK_ADS="PLACEHOLDER_UPDATE_ME" \
  DISCORD_WEBHOOK_INFRA="PLACEHOLDER_UPDATE_ME" \
  DISCORD_WEBHOOK_SECURITY="PLACEHOLDER_UPDATE_ME" \
  DISCORD_WEBHOOK_PERFORMANCE="PLACEHOLDER_UPDATE_ME" \
  DISCORD_WEBHOOK_REVENUE="PLACEHOLDER_UPDATE_ME" \
  DISCORD_WEBHOOK_CRITICAL="PLACEHOLDER_UPDATE_ME" \
  DISCORD_WEBHOOK_COMPLIANCE="PLACEHOLDER_UPDATE_ME" \
  AGONES_ALLOCATOR_ENDPOINT="agones-allocator.agones-system.svc.cluster.local:443" \
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
  VERIFY_API_SECRET="PLACEHOLDER_UPDATE_ME"
```

## Required Secrets to Update

### Priority 1 (Critical - needed for bot to start)

- [ ] `DISCORD_TOKEN` - Get from https://discord.com/developers/applications
- [ ] `DISCORD_CLIENT_ID` - Same place as token
- [ ] `DISCORD_GUILD_ID` - Right-click your Discord server → Copy ID
- [ ] `DB_PASSWORD` - Check your database setup

### Priority 2 (Needed for new features)

- [ ] `AYET_API_KEY` - ayeT-Studios dashboard
- [ ] `AYET_CALLBACK_TOKEN` - ayeT-Studios dashboard
- [ ] `AYET_VIDEO_PLACEMENT_ID` - ayeT-Studios dashboard

### Priority 3 (Monitoring - can be added later)

- [ ] `SENTRY_DSN` - sentry.io project settings
- [ ] `DISCORD_WEBHOOK_*` (8 webhooks) - Discord server settings

### Priority 4 (Optional - can leave empty for now)

- [ ] Discord channel IDs (8 channels)
- [ ] `VERIFIED_ROLE_ID`
- [ ] `VERIFY_API_SECRET`

## Verify Setup

After adding secrets:

```bash
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="your_token"

# List all secrets
vault kv get kubefirst/development/agis-bot

# Check specific secret
vault kv get -field=DISCORD_TOKEN kubefirst/development/agis-bot
```

## Next Steps After Vault Setup

Once secrets are in Vault, proceed with:

```bash
# 1. Apply database migrations
kubectl -n database exec -it postgresql-0 -- psql -U root -d agis -f /path/to/migration.sql

# 2. Deploy to development
helm upgrade --install agis-bot charts/agis-bot \
  -n development --create-namespace \
  -f charts/agis-bot/values.yaml

# 3. Verify deployment
kubectl -n development get pods
kubectl -n development logs -f deployment/agis-bot
```

## Troubleshooting

### Can't Find Root Token?

If you truly can't find the root token, you may need to:

1. **Regenerate root token** (requires unseal keys):
   ```bash
   # Get unseal keys from k8s secret
   kubectl -n vault get secret vault-unseal-keys -o json
   
   # Use unseal keys to generate new root token
   kubectl -n vault exec -it vault-0 -- vault operator generate-root -init
   ```

2. **Use kubefirst CLI** (if available):
   ```bash
   kubefirst vault root-token
   ```

3. **Check kubefirst GitOps repo** - may have initial secrets

### Port-Forward Died?

Restart it:

```bash
kubectl port-forward -n vault svc/vault 8200:8200
```

### Permission Denied?

Your token may have limited permissions. You need a token with policy allowing:

```hcl
path "kubefirst/data/development/agis-bot" {
  capabilities = ["create", "update", "read"]
}
```

## Summary

**You're here**: Week 1, Step 1 - Adding secrets to Vault
**Blocker**: Need valid Vault root token
**Next**: Once secrets are added, continue to database migrations

**All infrastructure is ready**:
- ✅ Database migration SQL written
- ✅ Helm charts updated
- ✅ Vault scripts created
- ✅ Documentation complete
- ✅ Port-forward running

Just need the token to proceed! 🚀

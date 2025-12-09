# Week 1, Step 4: Sentry Alert Configuration - SETUP GUIDE

**Status**: Ready for configuration  
**Date**: 2025-11-09  
**Environment**: development  
**Alert Channels**: 8 Discord webhooks

## Overview

This guide covers setting up Sentry error monitoring and Discord webhook alerts for AGIS Bot v2.0.

## Current Status

### ✅ Prerequisites Met
- [x] Sentry DSN placeholder in Vault
- [x] 8 Discord webhooks configured in Vault
- [x] Pod running and logging errors
- [x] Alert rule templates created

### ⏳ Pending Actions
- [ ] Create Sentry project (if not exists)
- [ ] Get real Sentry DSN
- [ ] Create Discord webhooks
- [ ] Configure alert rules
- [ ] Test error capture

## Step 1: Create Sentry Project

### Option A: Sentry.io (SaaS - Recommended)

```bash
# 1. Go to https://sentry.io
# 2. Sign up or log in
# 3. Create organization: "We The Gamers"
# 4. Create project:
#    - Name: agis-bot
#    - Platform: Go
#    - Team: DevOps
#    - Alert Settings: Custom
# 5. Copy DSN from Settings → Client Keys (DSN)
```

### Option B: Self-Hosted Sentry

```bash
# If using self-hosted Sentry at sentry.your-domain.com
# 1. Create organization
# 2. Create project (Go platform)
# 3. Copy DSN
```

## Step 2: Update Vault with Real Credentials

```bash
# Port-forward to Vault
kubectl port-forward -n vault svc/vault 8200:8200 &

# Set environment variables
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="<redacted>"

# Update Sentry DSN
vault kv patch secret/development/agis-bot \
  SENTRY_DSN="https://your-key@sentry.io/your-project-id"

# Verify
vault kv get secret/development/agis-bot | grep SENTRY_DSN
```

## Step 3: Create Discord Webhooks

### Create Webhook for Each Channel

In your Discord server:

```
1. Server Settings → Integrations → Webhooks
2. Click "New Webhook"
3. Name: "Sentry Alerts - Payments"
4. Select channel: #alerts-payments
5. Copy webhook URL
6. Repeat for each channel
```

### Webhook Channels

| Channel | Webhook Variable | Purpose |
|---------|------------------|---------|
| #alerts-payments | DISCORD_WEBHOOK_PAYMENTS | Payment failures |
| #alerts-ads | DISCORD_WEBHOOK_ADS | Ad conversion errors |
| #alerts-infra | DISCORD_WEBHOOK_INFRA | Infrastructure issues |
| #alerts-security | DISCORD_WEBHOOK_SECURITY | Auth failures |
| #alerts-performance | DISCORD_WEBHOOK_PERFORMANCE | Performance degradation |
| #alerts-revenue | DISCORD_WEBHOOK_REVENUE | Revenue processing |
| #alerts-critical | DISCORD_WEBHOOK_CRITICAL | Panics/critical errors |
| #alerts-compliance | DISCORD_WEBHOOK_COMPLIANCE | GDPR/compliance issues |

### Update Vault with Webhooks

```bash
vault kv patch secret/development/agis-bot \
  DISCORD_WEBHOOK_PAYMENTS="https://discord.com/api/webhooks/..." \
  DISCORD_WEBHOOK_ADS="https://discord.com/api/webhooks/..." \
  DISCORD_WEBHOOK_INFRA="https://discord.com/api/webhooks/..." \
  DISCORD_WEBHOOK_SECURITY="https://discord.com/api/webhooks/..." \
  DISCORD_WEBHOOK_PERFORMANCE="https://discord.com/api/webhooks/..." \
  DISCORD_WEBHOOK_REVENUE="https://discord.com/api/webhooks/..." \
  DISCORD_WEBHOOK_CRITICAL="https://discord.com/api/webhooks/..." \
  DISCORD_WEBHOOK_COMPLIANCE="https://discord.com/api/webhooks/..."
```

## Step 4: Configure Alert Rules

### Manual Setup (Recommended for Testing)

1. Go to Sentry: https://sentry.io/settings/your-org/agis-bot/alerts/
2. Click "Create Alert Rule"
3. Choose "Issue Alert"
4. Set conditions:
   - **When**: An event is first seen
   - **And**: Tags match (category:payment)
5. Set actions:
   - **Then**: Send a Discord notification
   - **To**: Select webhook
6. Save

### Automated Setup (API)

```bash
# Set environment variables
export SENTRY_ORG="your-org"
export SENTRY_PROJECT="agis-bot"
export SENTRY_AUTH_TOKEN="your-auth-token"

# Get auth token from: https://sentry.io/settings/account/api/auth-tokens/

# Run setup script
cd /home/seb/wtg/agis-bot
chmod +x scripts/setup-sentry-alerts.sh

./scripts/setup-sentry-alerts.sh
```

## Alert Rules Configuration

### Rule 1: Payment Processing Failures
- **Trigger**: Any error with tag `category:payment`
- **Threshold**: 1 error in 5 minutes
- **Severity**: Critical
- **Action**: Discord → #alerts-payments

### Rule 2: Ad Conversion Errors
- **Trigger**: Any error with tag `category:ad_conversion`
- **Threshold**: 5 errors in 10 minutes
- **Severity**: High
- **Action**: Discord → #alerts-ads

### Rule 3: Database Connection Errors
- **Trigger**: Any error with tag `category:database`
- **Threshold**: 1 error in 5 minutes
- **Severity**: Critical
- **Action**: Discord → #alerts-infra

### Rule 4: Authentication Failures
- **Trigger**: Any error with tag `category:auth`
- **Threshold**: 3 errors in 10 minutes
- **Severity**: High
- **Action**: Discord → #alerts-security

### Rule 5: Performance Degradation
- **Trigger**: Transaction duration > 5000ms
- **Threshold**: 10% of transactions in 15 minutes
- **Severity**: Medium
- **Action**: Discord → #alerts-performance

### Rule 6: Revenue Processing Errors
- **Trigger**: Any error with tag `category:revenue`
- **Threshold**: 1 error in 5 minutes
- **Severity**: Critical
- **Action**: Discord → #alerts-revenue

### Rule 7: Critical Errors (Panics)
- **Trigger**: Any error with level `fatal`
- **Threshold**: 1 error immediately
- **Severity**: Critical
- **Action**: Discord → #alerts-critical

### Rule 8: Compliance Issues
- **Trigger**: Any error with tag `category:compliance`
- **Threshold**: 1 error immediately
- **Severity**: Critical
- **Action**: Discord → #alerts-compliance

## Step 5: Test Error Capture

### Trigger Test Error

```bash
# Port-forward to bot
kubectl port-forward -n development svc/agis-bot 9090:9090 &

# Trigger test error (if endpoint exists)
curl -X POST http://localhost:9090/api/test-error

# Or check logs for any errors
kubectl logs -n development agis-bot-xxx --tail=20
```

### Verify in Sentry

1. Go to Sentry: https://sentry.io/organizations/your-org/issues/
2. Look for test error
3. Verify error details captured
4. Check Discord webhook received notification

## Step 6: Verify Discord Webhooks

### Test Webhook Manually

```bash
# Get webhook URL from Vault
export WEBHOOK_URL=$(vault kv get -field=DISCORD_WEBHOOK_PAYMENTS secret/development/agis-bot)

# Send test message
curl -X POST "$WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "🧪 Test message from Sentry setup",
    "embeds": [{
      "title": "Test Alert",
      "description": "This is a test alert from Sentry",
      "color": 16711680
    }]
  }'
```

### Verify in Discord

1. Check #alerts-payments channel
2. Verify test message received
3. Repeat for other webhooks

## Step 7: Monitor and Adjust

### Review Alert Performance

```bash
# Check Sentry alert rules
# https://sentry.io/settings/your-org/agis-bot/alerts/

# Monitor Discord channels for alerts
# Adjust thresholds if too noisy or missing issues
```

### Common Adjustments

| Issue | Solution |
|-------|----------|
| Too many alerts | Increase threshold or add more specific conditions |
| Missing alerts | Decrease threshold or broaden conditions |
| Wrong channel | Update webhook routing in alert rule |
| No Discord messages | Verify webhook URL and permissions |

## Troubleshooting

### Sentry DSN Not Working

```bash
# Check DSN in pod
kubectl exec -n development agis-bot-xxx -- env | grep SENTRY_DSN

# Check pod logs
kubectl logs -n development agis-bot-xxx | grep -i sentry

# Restart pod to pick up new DSN
kubectl rollout restart deployment/agis-bot -n development
```

### Discord Webhooks Not Receiving Alerts

```bash
# Verify webhook URL
vault kv get secret/development/agis-bot | grep DISCORD_WEBHOOK

# Test webhook manually
curl -X POST "https://discord.com/api/webhooks/..." \
  -H "Content-Type: application/json" \
  -d '{"content": "Test"}'

# Check webhook permissions in Discord
# Server Settings → Integrations → Webhooks → Select webhook
```

### Errors Not Being Captured

1. Verify SENTRY_DSN is set in pod
2. Check error tagging in code
3. Verify error level (must be error or higher)
4. Check Sentry project settings for sampling

## Implementation Checklist

- [ ] Create Sentry project
- [ ] Get DSN and update Vault
- [ ] Create 8 Discord webhooks
- [ ] Update Vault with webhook URLs
- [ ] Configure 8 alert rules
- [ ] Test error capture
- [ ] Verify Discord notifications
- [ ] Monitor for 24 hours
- [ ] Adjust thresholds as needed

## Files Created

- `docs/SENTRY_SETUP_GUIDE.md` - Comprehensive setup guide
- `scripts/setup-sentry-alerts.sh` - Automated alert rule creation
- `WEEK1_STEP4_SENTRY_SETUP.md` - This file

## Next Steps

1. **Immediate** (Today):
   - Create Sentry project
   - Get DSN
   - Create Discord webhooks
   - Update Vault

2. **Short-term** (Tomorrow):
   - Configure alert rules
   - Test error capture
   - Verify Discord notifications

3. **Ongoing**:
   - Monitor alert frequency
   - Adjust thresholds
   - Review error patterns

## Resources

- [Sentry Go SDK](https://docs.sentry.io/platforms/go/)
- [Sentry Alert Rules](https://docs.sentry.io/product/alerts/)
- [Discord Webhooks](https://discord.com/developers/docs/resources/webhook)
- [AGIS Bot Documentation](../README.md)

## Support

For issues or questions:
1. Check troubleshooting section above
2. Review Sentry documentation
3. Check Discord webhook permissions
4. Review pod logs: `kubectl logs -n development agis-bot-xxx`

---

**Status**: Ready for implementation  
**Estimated Time**: 1-2 hours  
**Difficulty**: Medium  
**Dependencies**: Sentry account, Discord server

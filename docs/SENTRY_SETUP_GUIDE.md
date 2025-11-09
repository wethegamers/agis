# Sentry Setup Guide for AGIS Bot v2.0

## Overview

Sentry provides real-time error tracking and performance monitoring for the AGIS Bot. This guide covers:
- Creating a Sentry project
- Configuring the DSN
- Setting up alert rules
- Integrating Discord webhooks
- Testing error capture

## Step 1: Create Sentry Project

### Option A: Self-Hosted Sentry

If using self-hosted Sentry:

```bash
# Access your Sentry instance
https://sentry.your-domain.com

# Create new organization (if needed)
# Create new project
# - Platform: Go
# - Alert Settings: Custom
```

### Option B: Sentry.io (SaaS)

1. Go to https://sentry.io
2. Sign up or log in
3. Create new organization: "We The Gamers"
4. Create new project:
   - **Name**: agis-bot
   - **Platform**: Go
   - **Team**: DevOps
   - **Alert Settings**: Custom

## Step 2: Get DSN

After creating the project:

1. Navigate to **Settings** → **Client Keys (DSN)**
2. Copy the DSN (format: `https://key@sentry.io/project-id`)
3. Update Vault:

```bash
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="your-token"

vault kv patch secret/development/agis-bot \
  SENTRY_DSN="https://your-key@sentry.io/your-project-id"
```

## Step 3: Configure Discord Webhooks

### Create Discord Webhooks

In your Discord server, create webhooks for each alert channel:

```bash
# For each channel, go to:
# Server Settings → Integrations → Webhooks → New Webhook

# Recommended channels and webhooks:
1. #alerts-payments → DISCORD_WEBHOOK_PAYMENTS
2. #alerts-ads → DISCORD_WEBHOOK_ADS
3. #alerts-infra → DISCORD_WEBHOOK_INFRA
4. #alerts-security → DISCORD_WEBHOOK_SECURITY
5. #alerts-performance → DISCORD_WEBHOOK_PERFORMANCE
6. #alerts-revenue → DISCORD_WEBHOOK_REVENUE
7. #alerts-critical → DISCORD_WEBHOOK_CRITICAL
8. #alerts-compliance → DISCORD_WEBHOOK_COMPLIANCE
```

### Update Vault with Webhook URLs

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

## Step 4: Configure Sentry Alert Rules

### Alert Rule 1: Payment Processing Failures

**Trigger**: Any error with tag `category:payment`  
**Severity**: Critical  
**Action**: Discord webhook to #alerts-payments

```
Condition: event.type:error AND tags.category:payment
Threshold: 1 error in 5 minutes
Action: Send to DISCORD_WEBHOOK_PAYMENTS
```

### Alert Rule 2: Ad Conversion Errors

**Trigger**: Ad conversion processing failures  
**Severity**: High  
**Action**: Discord webhook to #alerts-ads

```
Condition: event.type:error AND tags.category:ad_conversion
Threshold: 5 errors in 10 minutes
Action: Send to DISCORD_WEBHOOK_ADS
```

### Alert Rule 3: Database Connection Errors

**Trigger**: Database connectivity issues  
**Severity**: Critical  
**Action**: Discord webhook to #alerts-infra

```
Condition: event.type:error AND tags.category:database
Threshold: 1 error in 5 minutes
Action: Send to DISCORD_WEBHOOK_INFRA
```

### Alert Rule 4: Authentication Failures

**Trigger**: Discord bot authentication failures  
**Severity**: High  
**Action**: Discord webhook to #alerts-security

```
Condition: event.type:error AND tags.category:auth
Threshold: 3 errors in 10 minutes
Action: Send to DISCORD_WEBHOOK_SECURITY
```

### Alert Rule 5: Performance Degradation

**Trigger**: Response time exceeds threshold  
**Severity**: Medium  
**Action**: Discord webhook to #alerts-performance

```
Condition: transaction.duration > 5000ms
Threshold: 10% of transactions in 15 minutes
Action: Send to DISCORD_WEBHOOK_PERFORMANCE
```

### Alert Rule 6: Revenue Processing Errors

**Trigger**: Payment/revenue processing failures  
**Severity**: Critical  
**Action**: Discord webhook to #alerts-revenue

```
Condition: event.type:error AND tags.category:revenue
Threshold: 1 error in 5 minutes
Action: Send to DISCORD_WEBHOOK_REVENUE
```

### Alert Rule 7: Critical Errors (Panics)

**Trigger**: Any panic or critical error  
**Severity**: Critical  
**Action**: Discord webhook to #alerts-critical

```
Condition: event.type:error AND level:fatal
Threshold: 1 error immediately
Action: Send to DISCORD_WEBHOOK_CRITICAL
```

### Alert Rule 8: Compliance Issues

**Trigger**: GDPR/compliance-related errors  
**Severity**: Critical  
**Action**: Discord webhook to #alerts-compliance

```
Condition: event.type:error AND tags.category:compliance
Threshold: 1 error immediately
Action: Send to DISCORD_WEBHOOK_COMPLIANCE
```

## Step 5: Create Alert Rules in Sentry UI

### Manual Setup (Recommended for Testing)

1. Go to **Alerts** → **Create Alert Rule**
2. Choose **Issue Alert**
3. Set conditions:
   - **When**: An event is first seen
   - **And**: Tags match (category:payment)
4. Set actions:
   - **Then**: Send a Discord notification
   - **To**: Select webhook
5. Save alert rule

### Automated Setup (API)

```bash
#!/bin/bash
# scripts/setup-sentry-alerts.sh

SENTRY_ORG="your-org"
SENTRY_PROJECT="agis-bot"
SENTRY_AUTH_TOKEN="your-auth-token"
SENTRY_API="https://sentry.io/api/0"

# Function to create alert rule
create_alert_rule() {
  local name=$1
  local query=$2
  local webhook=$3
  
  curl -X POST \
    "${SENTRY_API}/projects/${SENTRY_ORG}/${SENTRY_PROJECT}/alert-rules/" \
    -H "Authorization: Bearer ${SENTRY_AUTH_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"${name}\",
      \"environment\": \"development\",
      \"dataset\": \"events\",
      \"query\": \"${query}\",
      \"aggregate\": \"count()\",
      \"timeWindow\": 5,
      \"triggers\": [
        {
          \"label\": \"critical\",
          \"alertThreshold\": 1,
          \"actions\": [
            {
              \"type\": \"discord\",
              \"targetIdentifier\": \"${webhook}\"
            }
          ]
        }
      ]
    }"
}

# Create alert rules
create_alert_rule \
  "Payment Processing Failure" \
  "event.type:error tags.category:payment" \
  "${DISCORD_WEBHOOK_PAYMENTS}"

create_alert_rule \
  "Ad Conversion Error" \
  "event.type:error tags.category:ad_conversion" \
  "${DISCORD_WEBHOOK_ADS}"

create_alert_rule \
  "Database Connection Error" \
  "event.type:error tags.category:database" \
  "${DISCORD_WEBHOOK_INFRA}"

echo "✅ Alert rules created successfully"
```

## Step 6: Test Error Capture

### Trigger a Test Error

```bash
# Port-forward to the bot
kubectl port-forward -n development svc/agis-bot 9090:9090

# Trigger an error (example - depends on bot implementation)
curl -X POST http://localhost:9090/api/test-error
```

### Verify in Sentry

1. Go to **Issues** in Sentry
2. Look for the test error
3. Verify error details are captured
4. Check Discord webhook received notification

## Step 7: Configure Error Tagging

In the bot code, tag errors for better routing:

```go
import "github.com/getsentry/sentry-go"

// Example: Payment error
sentry.CaptureException(err, func(scope *sentry.Scope) {
  scope.SetTag("category", "payment")
  scope.SetTag("severity", "critical")
  scope.SetContext("payment", map[string]interface{}{
    "user_id": userID,
    "amount": amount,
  })
})

// Example: Ad conversion error
sentry.CaptureException(err, func(scope *sentry.Scope) {
  scope.SetTag("category", "ad_conversion")
  scope.SetTag("severity", "high")
})
```

## Step 8: Monitor and Adjust

### Review Alert Performance

1. Check **Alerts** → **Alert Rules**
2. Review trigger frequency
3. Adjust thresholds if too noisy or missing issues
4. Monitor Discord channel for false positives

### Common Adjustments

- **Too many alerts**: Increase threshold or add more specific conditions
- **Missing alerts**: Decrease threshold or broaden conditions
- **Wrong channel**: Update webhook routing in alert rule

## Troubleshooting

### Sentry DSN Not Working

```bash
# Verify DSN in pod
kubectl exec -n development agis-bot-xxx -- env | grep SENTRY

# Check pod logs for Sentry errors
kubectl logs -n development agis-bot-xxx | grep -i sentry
```

### Discord Webhooks Not Receiving Alerts

1. Verify webhook URL is correct
2. Check webhook permissions in Discord
3. Test webhook manually:
   ```bash
   curl -X POST "https://discord.com/api/webhooks/..." \
     -H "Content-Type: application/json" \
     -d '{"content": "Test message"}'
   ```
4. Check Sentry alert rule is enabled

### Errors Not Being Captured

1. Verify SENTRY_DSN is set in pod
2. Check error tagging in code
3. Verify error level (must be error or higher)
4. Check Sentry project settings for sampling

## Next Steps

- [ ] Create Sentry project
- [ ] Get DSN and update Vault
- [ ] Create Discord webhooks
- [ ] Configure alert rules
- [ ] Test error capture
- [ ] Monitor for 24 hours
- [ ] Adjust thresholds as needed

## References

- [Sentry Go SDK](https://docs.sentry.io/platforms/go/)
- [Sentry Alert Rules](https://docs.sentry.io/product/alerts/)
- [Discord Webhooks](https://discord.com/developers/docs/resources/webhook)
- [AGIS Bot Error Handling](../internal/bot/error_handling.go)

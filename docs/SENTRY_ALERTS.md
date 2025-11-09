# Sentry Alert Configuration

This guide covers setting up Sentry alerts for payment failures, ad conversion errors, and performance degradation.

## Prerequisites

- Sentry project created (e.g., `agis-bot`)
- `SENTRY_DSN` configured in deployment
- Discord webhooks created for alert routing
- Optional: PagerDuty integration for critical alerts

## Alert Configuration Methods

### Method 1: Sentry UI (Manual)

1. Navigate to **Alerts** → **Create Alert Rule**
2. Choose alert type:
   - **Issue Alert**: Triggered when error events match conditions
   - **Metric Alert**: Triggered by aggregated metrics (count, rate, percentiles)
3. Copy conditions from `deployments/sentry/alert-rules.yaml`
4. Configure actions (Discord, PagerDuty, email)

### Method 2: Sentry API (Automated)

```bash
#!/bin/bash
# apply-sentry-alerts.sh

SENTRY_AUTH_TOKEN="{{SENTRY_AUTH_TOKEN}}"
SENTRY_ORG="your-org"
SENTRY_PROJECT="agis-bot"

# Create metric alert for payment failures
curl -X POST "https://sentry.io/api/0/projects/${SENTRY_ORG}/${SENTRY_PROJECT}/alert-rules/" \
  -H "Authorization: Bearer ${SENTRY_AUTH_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Payment Processing Failure - Critical",
    "environment": "production",
    "dataset": "events",
    "query": "event.type:error event.tags.category:payment",
    "aggregate": "count()",
    "timeWindow": 5,
    "triggers": [
      {
        "label": "critical",
        "alertThreshold": 5,
        "actions": [
          {
            "type": "discord",
            "targetIdentifier": "{{DISCORD_WEBHOOK_PAYMENTS}}"
          }
        ]
      }
    ]
  }'
```

### Method 3: Terraform (GitOps)

```hcl
resource "sentry_alert_rule" "payment_failure" {
  organization = "your-org"
  project      = "agis-bot"
  name         = "Payment Processing Failure - Critical"
  environment  = "production"

  conditions {
    id          = "sentry.rules.conditions.event_attribute.EventAttributeCondition"
    attribute   = "tags.category"
    match       = "eq"
    value       = "payment"
  }

  actions {
    id                  = "sentry.integrations.discord.notify_action.DiscordNotifyServiceAction"
    workspace           = var.discord_workspace_id
    channel_id          = var.discord_channel_payments
  }

  action_match = "all"
  frequency    = 5
}
```

## Alert Severity Levels

| Severity | Response Time | Escalation | Example |
|----------|---------------|------------|---------|
| **Critical** | Immediate (PagerDuty) | On-call engineer | Payment failures, zero conversions, panics |
| **High** | 15 minutes | Engineering team | Ad signature verification failures, DB errors |
| **Medium** | 1 hour | Engineering team | High fraud rate, conversion processing errors |
| **Low** | Next business day | Team review | Subscription errors, performance warnings |

## Discord Webhook Setup

Create separate webhooks for alert routing:

```bash
# In Discord Server Settings → Integrations → Webhooks
DISCORD_WEBHOOK_PAYMENTS=https://discord.com/api/webhooks/...     # #alerts-payments
DISCORD_WEBHOOK_ADS=https://discord.com/api/webhooks/...          # #alerts-ads
DISCORD_WEBHOOK_INFRA=https://discord.com/api/webhooks/...        # #alerts-infra
DISCORD_WEBHOOK_SECURITY=https://discord.com/api/webhooks/...     # #alerts-security
DISCORD_WEBHOOK_PERFORMANCE=https://discord.com/api/webhooks/...  # #alerts-performance
DISCORD_WEBHOOK_REVENUE=https://discord.com/api/webhooks/...      # #alerts-revenue
DISCORD_WEBHOOK_CRITICAL=https://discord.com/api/webhooks/...     # #alerts-critical
DISCORD_WEBHOOK_COMPLIANCE=https://discord.com/api/webhooks/...   # #alerts-compliance
```

Store in Kubernetes Secret or ExternalSecret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: sentry-webhooks
  namespace: development
type: Opaque
stringData:
  discord-webhook-payments: "https://discord.com/api/webhooks/..."
  discord-webhook-ads: "https://discord.com/api/webhooks/..."
  # ... etc
```

## Critical Alert Rules

### 1. Payment Processing Failure
**Trigger**: 5+ payment errors in 5 minutes  
**Action**: Discord (#alerts-payments) + PagerDuty  
**Query**: `event.type:error AND event.tags.category:payment`

### 2. Ad Signature Verification Failure
**Trigger**: 10+ signature failures in 10 minutes  
**Action**: Discord (#alerts-ads)  
**Query**: `event.message:"signature verification failed" AND event.tags.provider:ayet`

### 3. Database Connection Failure
**Trigger**: 3+ connection errors in 5 minutes  
**Action**: Discord (#alerts-infra) + Email  
**Query**: `event.tags.category:database AND event.message:("connection refused" OR "timeout")`

### 4. Zero Successful Conversions
**Trigger**: <1 conversion in 30 minutes  
**Action**: Discord (#alerts-revenue) + PagerDuty  
**Query**: `event.tags.status:completed` (inverted threshold)

### 5. Unhandled Panic
**Trigger**: Any fatal-level event  
**Action**: PagerDuty + Discord (#alerts-critical)  
**Query**: `event.level:fatal`

## Performance Alerts

### High Callback Latency
**Metric**: `p95(transaction.duration) > 2000ms`  
**Filter**: `transaction:"POST /ads/ayet/s2s"`  
**Action**: Discord (#alerts-performance)

### High Error Rate
**Metric**: `failure_rate() > 5%`  
**Filter**: `transaction:/ads/*`  
**Action**: Discord (#alerts-ads)

### Low Apdex Score
**Metric**: `apdex(300) < 0.8`  
**Filter**: `transaction:/ads/ayet/s2s`  
**Action**: Discord (#alerts-performance)

## Error Tagging Best Practices

Ensure errors are tagged correctly in code:

```go
// Payment error
sentry.WithScope(func(scope *sentry.Scope) {
    scope.SetTag("category", "payment")
    scope.SetTag("payment_method", "stripe")
    scope.SetContext("payment", map[string]interface{}{
        "amount":   amount,
        "currency": "usd",
        "user_id":  userID,
    })
    errorMonitor.TrackPaymentError(err)
})

// Ad callback error
sentry.WithScope(func(scope *sentry.Scope) {
    scope.SetTag("provider", "ayet")
    scope.SetTag("operation", "process_conversion")
    scope.SetContext("conversion", map[string]interface{}{
        "conversion_id": conversionID,
        "amount":        amount,
        "currency":      currency,
    })
    errorMonitor.TrackAdCallbackError(err)
})

// Database error
sentry.WithScope(func(scope *sentry.Scope) {
    scope.SetTag("category", "database")
    scope.SetTag("operation", query)
    errorMonitor.TrackDatabaseError(err)
})
```

## Testing Alerts

### Local Testing (Staging Environment)

```bash
# Trigger test alert via API
curl -X POST "https://{{AGIS_BOT_STAGING}}/internal/test-alert" \
  -H "Authorization: Bearer {{ADMIN_TOKEN}}" \
  -d '{"type": "payment_failure", "count": 5}'
```

### Manual Sentry Event

```go
// In test file or admin command
sentry.CaptureException(errors.New("TEST: Payment processing failure"))
sentry.WithScope(func(scope *sentry.Scope) {
    scope.SetTag("category", "payment")
    scope.SetLevel(sentry.LevelError)
    sentry.CaptureMessage("TEST ALERT: Payment failure")
})
```

### Verify Alert Delivery

1. Check Sentry **Alerts** → **History** for triggered rules
2. Verify Discord webhook received message
3. Confirm PagerDuty incident created (for critical alerts)

## Alert Tuning

### Reduce False Positives

- Increase thresholds for noisy alerts
- Add additional filters (e.g., exclude test users)
- Use resolve thresholds to auto-close

### Example Tuning

```yaml
# Before (too sensitive)
alert_threshold: 1  # fires on single error
time_window: 60     # checks every hour

# After (tuned)
alert_threshold: 5  # requires 5 errors
time_window: 10     # checks every 10 minutes
resolve_threshold: 2  # auto-resolves when < 2
```

## Monitoring Alert Health

Track alert metrics:

- **Alert fatigue**: Alerts triggered per day
- **Resolution time**: Time to acknowledge/resolve
- **False positive rate**: Alerts without action taken

Review monthly and adjust thresholds.

## Integration with Incident Management

### PagerDuty Setup

1. Create PagerDuty service for AGIS Bot
2. Generate integration key
3. Add to Sentry: **Settings** → **Integrations** → **PagerDuty**
4. Configure escalation policy (on-call rotation)

### Runbook Links

Add runbook links to alert descriptions:

```yaml
actions:
  - type: discord
    target_identifier: "{{DISCORD_WEBHOOK_PAYMENTS}}"
    description: |
      🚨 Payment Processing Failure
      
      Runbook: https://wiki.example.com/runbooks/payment-failure
      Dashboard: https://grafana.example.com/d/payments
      Logs: https://loki.example.com/?query={app="agis-bot",category="payment"}
```

## Compliance and Audit

### GDPR Consent Failures

Separate channel for compliance alerts:

```yaml
- name: "GDPR Consent Check Failure"
  actions:
    - type: email
      target_identifier: legal-team@example.com
    - type: discord
      target_identifier: "{{DISCORD_WEBHOOK_COMPLIANCE}}"
```

### Audit Log

Sentry automatically logs all alert triggers. Export via API for compliance:

```bash
curl "https://sentry.io/api/0/organizations/${SENTRY_ORG}/events/" \
  -H "Authorization: Bearer ${SENTRY_AUTH_TOKEN}" \
  -G -d "query=type:alert_triggered" \
  -d "start=$(date -u -d '30 days ago' +%Y-%m-%dT%H:%M:%S)" \
  -d "end=$(date -u +%Y-%m-%dT%H:%M:%S)"
```

## Troubleshooting

### Alerts Not Firing

- Verify `SENTRY_DSN` is set and reachable
- Check Sentry **Alerts** → **Alert Rule** → **Alert History**
- Confirm events match conditions (test with sample error)
- Verify environment filter matches deployment (`production`)

### Discord Webhook Not Working

- Test webhook directly: `curl -X POST {{WEBHOOK_URL}} -H "Content-Type: application/json" -d '{"content":"test"}'`
- Check webhook hasn't been deleted in Discord
- Verify webhook URL in Sentry integration settings

### Too Many Alerts

- Increase thresholds or time windows
- Add additional filters to narrow scope
- Implement alert grouping/fingerprinting
- Use "Ignore" rules for known issues

## Production Checklist

- [ ] All 8 critical alerts configured
- [ ] Discord webhooks tested and verified
- [ ] PagerDuty integration enabled for critical alerts
- [ ] On-call rotation scheduled in PagerDuty
- [ ] Alert thresholds tuned based on baseline metrics
- [ ] Runbook links added to alert descriptions
- [ ] Team trained on alert response procedures
- [ ] Alert history reviewed weekly
- [ ] False positive rate < 10%

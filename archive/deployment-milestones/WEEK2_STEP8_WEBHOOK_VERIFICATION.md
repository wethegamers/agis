# Week 2, Step 8: Webhook Verification

**Status**: Implementation Phase  
**Date**: 2025-11-10  
**Objective**: Verify all alert channels and webhook integrations  
**Timeline**: 1-2 hours

## Overview

This step verifies that all Discord webhooks and Sentry alerts are working correctly and routing to the correct channels.

## Current Status

### ✅ Discord Webhooks Configured
- 8 webhooks in Vault
- Ready for real webhook URLs
- Routing configured per alert type

### ✅ Sentry Alerts Prepared
- 8 alert rules prepared
- Automated setup script created
- Discord webhook routing configured

### ✅ Monitoring Infrastructure
- ServiceMonitor active
- Prometheus scraping metrics
- Grafana dashboard ready

## Step 8 Tasks

### Task 1: Verify Discord Webhook URLs

**Objective**: Confirm all webhook URLs are valid and accessible

**Steps**:
1. Get webhook URLs from Vault
2. Test each webhook with curl
3. Verify Discord receives test message

**Commands**:
```bash
# Get webhook URLs
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="hvs.kjP6fT17rS8dnnW7NTZqUOgm"

# Get each webhook
vault kv get -field=DISCORD_WEBHOOK_PAYMENTS secret/development/agis-bot
vault kv get -field=DISCORD_WEBHOOK_ADS secret/development/agis-bot
vault kv get -field=DISCORD_WEBHOOK_INFRA secret/development/agis-bot
vault kv get -field=DISCORD_WEBHOOK_SECURITY secret/development/agis-bot
vault kv get -field=DISCORD_WEBHOOK_PERFORMANCE secret/development/agis-bot
vault kv get -field=DISCORD_WEBHOOK_REVENUE secret/development/agis-bot
vault kv get -field=DISCORD_WEBHOOK_CRITICAL secret/development/agis-bot
vault kv get -field=DISCORD_WEBHOOK_COMPLIANCE secret/development/agis-bot
```

**Test Webhook**:
```bash
curl -X POST "https://discord.com/api/webhooks/[webhook-id]/[webhook-token]" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "🧪 Test message from AGIS Bot",
    "embeds": [{
      "title": "Webhook Test",
      "description": "This is a test message",
      "color": 3066993
    }]
  }'
```

**Expected Result**:
- Message appears in Discord channel
- No errors returned
- HTTP 204 response

### Task 2: Test Payment Alerts

**Objective**: Verify payment failure alerts work

**Steps**:
1. Trigger a payment error in the bot
2. Verify Sentry captures error
3. Verify Discord webhook receives alert
4. Check message in #alerts-payments

**Expected Result**:
- Error logged in Sentry
- Discord message in #alerts-payments
- Message includes error details
- Timestamp is current

### Task 3: Test Ad Conversion Alerts

**Objective**: Verify ad conversion error alerts work

**Steps**:
1. Trigger an ad conversion error
2. Verify Sentry captures error
3. Verify Discord webhook receives alert
4. Check message in #alerts-ads

**Expected Result**:
- Error logged in Sentry
- Discord message in #alerts-ads
- Message includes conversion details
- Alert threshold respected

### Task 4: Test Infrastructure Alerts

**Objective**: Verify infrastructure error alerts work

**Steps**:
1. Trigger a database connection error
2. Verify Sentry captures error
3. Verify Discord webhook receives alert
4. Check message in #alerts-infra

**Expected Result**:
- Error logged in Sentry
- Discord message in #alerts-infra
- Message includes error details
- Severity level indicated

### Task 5: Test Security Alerts

**Objective**: Verify authentication failure alerts work

**Steps**:
1. Trigger an authentication error
2. Verify Sentry captures error
3. Verify Discord webhook receives alert
4. Check message in #alerts-security

**Expected Result**:
- Error logged in Sentry
- Discord message in #alerts-security
- Message includes security details
- Alert severity high

### Task 6: Test Performance Alerts

**Objective**: Verify performance degradation alerts work

**Steps**:
1. Simulate slow response times
2. Verify Prometheus detects degradation
3. Verify alert triggers
4. Check message in #alerts-performance

**Expected Result**:
- Prometheus alert fires
- Discord message in #alerts-performance
- Message includes latency metrics
- Threshold exceeded indicated

### Task 7: Test Revenue Alerts

**Objective**: Verify revenue processing error alerts work

**Steps**:
1. Trigger a revenue processing error
2. Verify Sentry captures error
3. Verify Discord webhook receives alert
4. Check message in #alerts-revenue

**Expected Result**:
- Error logged in Sentry
- Discord message in #alerts-revenue
- Message includes revenue details
- Alert severity critical

### Task 8: Test Critical Error Alerts

**Objective**: Verify panic/critical error alerts work

**Steps**:
1. Trigger a panic or critical error
2. Verify Sentry captures error
3. Verify Discord webhook receives alert
4. Check message in #alerts-critical

**Expected Result**:
- Error logged in Sentry
- Discord message in #alerts-critical
- Message includes stack trace
- Alert severity critical

### Task 9: Test Compliance Alerts

**Objective**: Verify GDPR/compliance error alerts work

**Steps**:
1. Trigger a compliance error
2. Verify Sentry captures error
3. Verify Discord webhook receives alert
4. Check message in #alerts-compliance

**Expected Result**:
- Error logged in Sentry
- Discord message in #alerts-compliance
- Message includes compliance details
- Alert severity critical

## Testing Checklist

### Webhook Connectivity
- [ ] All 8 webhooks accessible
- [ ] Test messages received
- [ ] No permission errors
- [ ] Webhooks not expired

### Alert Routing
- [ ] Payments → #alerts-payments
- [ ] Ads → #alerts-ads
- [ ] Infrastructure → #alerts-infra
- [ ] Security → #alerts-security
- [ ] Performance → #alerts-performance
- [ ] Revenue → #alerts-revenue
- [ ] Critical → #alerts-critical
- [ ] Compliance → #alerts-compliance

### Message Formatting
- [ ] Embeds display correctly
- [ ] Colors are appropriate
- [ ] Timestamps are accurate
- [ ] Error details included
- [ ] Severity level indicated

### Sentry Integration
- [ ] Errors captured in Sentry
- [ ] Error details complete
- [ ] Stack traces included
- [ ] Tags applied correctly
- [ ] Context information present

### Alert Thresholds
- [ ] Alerts fire at correct threshold
- [ ] No false positives
- [ ] No missed alerts
- [ ] Threshold adjustable

## SQL Queries for Verification

### Check Webhook Configuration
```bash
vault kv get secret/development/agis-bot | grep DISCORD_WEBHOOK
```

### Check Sentry Configuration
```bash
vault kv get -field=SENTRY_DSN secret/development/agis-bot
```

### Check Alert Rules (in Sentry UI)
```
https://sentry.io/settings/[org]/[project]/alerts/
```

## Expected Results

### Webhook Tests
- All 8 webhooks respond with 204 No Content
- Test messages appear in Discord
- No rate limiting errors
- Webhooks remain active

### Alert Routing
- Each alert type routes to correct channel
- No cross-channel alerts
- Correct severity levels
- Proper formatting

### Message Content
- Error type clearly indicated
- Relevant details included
- Timestamps accurate
- Links to Sentry included

### Sentry Integration
- All errors captured
- Error details complete
- Proper categorization
- Correct severity levels

## Troubleshooting

### Webhook Not Responding
- Verify webhook URL is correct
- Check webhook permissions in Discord
- Verify webhook is not expired
- Check Discord server is accessible
- Review Discord API status

### Messages Not Appearing
- Verify webhook URL is correct
- Check Discord channel permissions
- Verify bot has send message permission
- Check message format is valid
- Review Discord logs

### Sentry Not Capturing Errors
- Verify SENTRY_DSN is correct
- Check Sentry project exists
- Verify error level is high enough
- Check sampling settings
- Review Sentry logs

### Alerts Not Firing
- Verify alert rules are enabled
- Check alert conditions
- Verify thresholds are correct
- Check webhook routing
- Review Sentry alert logs

## Success Criteria

- [ ] All 8 webhooks tested
- [ ] All webhooks accessible
- [ ] Test messages received
- [ ] Payment alerts working
- [ ] Ad conversion alerts working
- [ ] Infrastructure alerts working
- [ ] Security alerts working
- [ ] Performance alerts working
- [ ] Revenue alerts working
- [ ] Critical alerts working
- [ ] Compliance alerts working
- [ ] Correct routing verified
- [ ] Message formatting correct
- [ ] Sentry integration working

## Files Involved

### Configuration
- Vault: `secret/development/agis-bot`
- Sentry: Alert rules configuration

### Code
- `internal/services/error_monitoring.go` - Error monitoring
- `internal/services/ad_metrics.go` - Metrics collection

### Documentation
- `docs/SENTRY_SETUP_GUIDE.md` - Sentry setup
- `docs/SENTRY_ALERTS.md` - Alert configuration

## Next Steps

After webhook verification:
1. Complete Week 2 summary
2. Prepare for Week 3 production deployment
3. Review all changes
4. Plan production rollout

## Estimated Time

- Verify webhook URLs: 15 minutes
- Test payment alerts: 10 minutes
- Test ad conversion alerts: 10 minutes
- Test infrastructure alerts: 10 minutes
- Test security alerts: 10 minutes
- Test performance alerts: 10 minutes
- Test revenue alerts: 10 minutes
- Test critical alerts: 10 minutes
- Test compliance alerts: 10 minutes
- **Total: 95 minutes (1.6 hours)**

---

**Step 8 Ready to Execute!** 🚀

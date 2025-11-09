# AGIS Bot v2.0 - Production Deployment Guide

This guide covers deploying all production enhancement features to your Kubernetes cluster.

## Prerequisites

- Kubernetes cluster with Agones installed
- Helm 3.x
- PostgreSQL 15+ database
- Prometheus Operator (for ServiceMonitor)
- Grafana (for dashboards)
- External Secrets Operator with Vault
- kubectl access to cluster

## Deployment Steps

### 1. Prepare Secrets in Vault

Add the following secrets to your Vault path (`development/agis-bot` or `production/agis-bot`):

```bash
# Core Discord & Database
DISCORD_TOKEN=<your_bot_token>
DISCORD_CLIENT_ID=<your_client_id>
DISCORD_GUILD_ID=<your_guild_id>
DB_HOST=<postgres_host>
DB_USER=<postgres_user>
DB_PASSWORD=<postgres_password>
DB_NAME=agis

# ayeT-Studios Ad Network
AYET_API_KEY=<production_api_key>
AYET_CALLBACK_TOKEN=<shared_secret>
AYET_OFFERWALL_URL=https://offerwall.ayet-studios.com/...
AYET_SURVEYWALL_URL=https://surveywall.ayet-studios.com/...
AYET_VIDEO_PLACEMENT_ID=<placement_id>

# Sentry Error Monitoring
SENTRY_DSN=https://...@sentry.io/...

# Discord Webhooks for Alerts (create 8 webhooks)
DISCORD_WEBHOOK_PAYMENTS=https://discord.com/api/webhooks/...
DISCORD_WEBHOOK_ADS=https://discord.com/api/webhooks/...
DISCORD_WEBHOOK_INFRA=https://discord.com/api/webhooks/...
DISCORD_WEBHOOK_SECURITY=https://discord.com/api/webhooks/...
DISCORD_WEBHOOK_PERFORMANCE=https://discord.com/api/webhooks/...
DISCORD_WEBHOOK_REVENUE=https://discord.com/api/webhooks/...
DISCORD_WEBHOOK_CRITICAL=https://discord.com/api/webhooks/...
DISCORD_WEBHOOK_COMPLIANCE=https://discord.com/api/webhooks/...

# Agones Configuration
AGONES_ALLOCATOR_ENDPOINT=<allocator_endpoint>
AGONES_ALLOCATOR_TLS=<tls_cert>
AGONES_NAMESPACE=game-servers

# Discord Logging Channels
LOG_CHANNEL_GENERAL=<channel_id>
LOG_CHANNEL_USER=<channel_id>
LOG_CHANNEL_MOD=<channel_id>
LOG_CHANNEL_ERROR=<channel_id>
LOG_CHANNEL_CLEANUP=<channel_id>
LOG_CHANNEL_CLUSTER=<channel_id>
LOG_CHANNEL_EXPORT=<channel_id>
LOG_CHANNEL_AUDIT=<channel_id>
```

### 2. Apply Database Migrations

**Connect to your PostgreSQL database**:

```bash
# Option 1: Via kubectl port-forward
kubectl port-forward -n database svc/postgresql 5432:5432

# Option 2: Direct connection
psql -h <postgres-host> -U <user> -d agis
```

**Apply migrations**:

```bash
# From repository root
psql -h localhost -U root -d agis -f deployments/migrations/v2.0-production-enhancements.sql
```

**Verify migrations**:

```sql
-- Check tables were created
SELECT table_name FROM information_schema.tables 
WHERE table_schema = 'public' 
ORDER BY table_name;

-- Check server templates
SELECT id, name, cost_per_hour FROM server_templates;

-- Check schema version
SELECT * FROM schema_migrations;
```

Expected output:
```
 version                       | applied_at          
-------------------------------+---------------------
 v2.0-production-enhancements | 2025-01-09 20:45:00
```

### 3. Update Helm Values

Create environment-specific values file:

**`values-production.yaml`**:
```yaml
replicaCount: 2

image:
  repository: ghcr.io/wethegamers/agis-bot
  tag: "v2.0.0"  # Update after building new image
  pullPolicy: Always

environment: production
wtgDashboardUrl: https://wethegamers.org

vaultSecretPath: production/agis-bot

monitoring:
  serviceMonitor:
    enabled: true
    interval: 15s
    scrapeTimeout: 10s
    labels:
      prometheus: kube-prometheus
  grafanaDashboard:
    enabled: true

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

ingress:
  enabled: true
  hosts:
    - host: bot-api.wethegamers.org
      paths:
        - path: /
          pathType: Prefix
```

**`values-staging.yaml`**:
```yaml
replicaCount: 1

image:
  tag: "latest"

environment: staging
vaultSecretPath: staging/agis-bot

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 250m
    memory: 256Mi
```

### 4. Deploy via Helm

**Development**:
```bash
helm upgrade --install agis-bot ./charts/agis-bot \
  -n development --create-namespace \
  -f values-development.yaml
```

**Staging**:
```bash
helm upgrade --install agis-bot ./charts/agis-bot \
  -n staging --create-namespace \
  -f values-staging.yaml
```

**Production**:
```bash
# Dry-run first
helm upgrade --install agis-bot ./charts/agis-bot \
  -n production --create-namespace \
  -f values-production.yaml \
  --dry-run --debug

# Deploy
helm upgrade --install agis-bot ./charts/agis-bot \
  -n production --create-namespace \
  -f values-production.yaml
```

### 5. Verify Deployment

**Check pod status**:
```bash
kubectl get pods -n production
kubectl logs -n production deployment/agis-bot -f
```

Expected log output:
```
🚀 Starting AGIS Bot v2.0 - Production Enhancement Edition
📊 Environment: production
🔐 Discord Token: Bot ****
💾 Database: postgresql.database.svc.cluster.local
📡 Metrics Port: 9090
✅ Error monitoring initialized
✅ Database initialized
📇 Ensuring database indexes...
✅ Database indexes ensured
✅ Agones client initialized
✅ Notification service initialized
✅ Ad metrics collector initialized
✅ Consent service initialized
✅ Reward algorithm initialized
✅ Ad conversion service initialized
✅ A/B testing service initialized
✅ Guild provisioning service initialized
✅ HTTP server initialized
🌐 Starting HTTP server on :9090
✅ AGIS Bot is running - Press Ctrl+C to stop
```

**Check endpoints**:
```bash
kubectl port-forward -n production svc/agis-bot 9090:9090

# Health checks
curl http://localhost:9090/healthz
curl http://localhost:9090/readyz

# Metrics
curl http://localhost:9090/metrics | grep agis_ad

# Version info
curl http://localhost:9090/version
```

### 6. Verify Prometheus Scraping

**Check ServiceMonitor**:
```bash
kubectl get servicemonitor -n production agis-bot -o yaml
```

**Verify in Prometheus UI**:
```bash
# Port-forward to Prometheus
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090

# Open http://localhost:9090
# Query: agis_ad_conversions_total
```

Expected: Metrics should appear within 15-30 seconds

### 7. Import Grafana Dashboard

**Option 1: Automatic (via ConfigMap)**

The dashboard is automatically provisioned if `monitoring.grafanaDashboard.enabled: true`.

Verify:
```bash
kubectl get configmap -n production agis-bot-grafana-dashboard
```

**Option 2: Manual Import**

1. Open Grafana UI
2. Navigate to **Dashboards** → **Import**
3. Upload `deployments/grafana/ad-metrics-dashboard.json`
4. Select Prometheus data source
5. Click **Import**

Dashboard URL: `http://grafana.example.com/d/agis-ad-metrics`

### 8. Configure Sentry Alerts

**Option 1: Sentry UI**

1. Log into Sentry: https://sentry.io
2. Navigate to **Alerts** → **Create Alert**
3. Follow configurations in `deployments/sentry/alert-rules.yaml`
4. Add Discord webhook integrations

**Option 2: Terraform**

```hcl
# terraform/sentry-alerts.tf
module "sentry_alerts" {
  source = "./modules/sentry-alerts"
  
  organization = "your-org"
  project      = "agis-bot"
  
  discord_webhook_payments = var.discord_webhook_payments
  discord_webhook_ads      = var.discord_webhook_ads
  # ... etc
}
```

**Test alerts**:
```bash
# Trigger test error
curl -X POST http://localhost:9090/internal/test-sentry \
  -H "Content-Type: application/json" \
  -d '{"type": "payment_error"}'
```

### 9. Run Integration Tests

**Set up GitHub Secrets**:

In GitHub repository settings → Secrets → Actions:

```
AYET_API_KEY_SANDBOX=<sandbox_key>
AYET_CALLBACK_TOKEN_SANDBOX=<sandbox_token>
DISCORD_TOKEN_TEST=<test_bot_token>
SENTRY_DSN_TEST=<test_sentry_dsn>
DISCORD_WEBHOOK_CI=<ci_webhook_url>
```

**Trigger workflow**:
```bash
# Manual trigger
gh workflow run integration-tests.yml

# Or push to main branch (runs automatically)
git push origin main
```

**Check results**:
```bash
gh run list --workflow=integration-tests.yml
gh run view <run_id>
```

### 10. Create Discord Webhooks

**For each alert channel, create a webhook**:

1. Discord Server → Edit Channel → Integrations → Webhooks
2. Create webhook for each channel:
   - `#alerts-payments`
   - `#alerts-ads`
   - `#alerts-infra`
   - `#alerts-security`
   - `#alerts-performance`
   - `#alerts-revenue`
   - `#alerts-critical`
   - `#alerts-compliance`
3. Copy webhook URLs and add to Vault

**Test webhooks**:
```bash
curl -X POST "https://discord.com/api/webhooks/..." \
  -H "Content-Type: application/json" \
  -d '{"content": "✅ Alert webhook configured successfully"}'
```

## Post-Deployment Verification

### Database Health

```sql
-- Check ad conversions table
SELECT COUNT(*) FROM ad_conversions;

-- Check guild treasury
SELECT guild_id, balance FROM guild_treasury;

-- Check A/B experiments
SELECT id, name, status FROM ab_experiments;

-- Check server templates
SELECT id, name, cost_per_hour FROM server_templates;

-- Check performance
EXPLAIN ANALYZE SELECT * FROM ad_conversions 
WHERE discord_id = 'test' AND created_at > NOW() - INTERVAL '24 hours';
```

### Metrics Verification

```bash
# Check all ad metrics are being exported
curl http://localhost:9090/metrics | grep -E "agis_ad_(conversions|rewards|fraud|callback|conversions_by_tier)"
```

Expected output:
```
agis_ad_conversions_total{provider="ayet",type="offerwall",status="completed"} 0
agis_ad_rewards_total{provider="ayet",type="offerwall"} 0
agis_ad_fraud_attempts_total{provider="ayet",reason="velocity"} 0
agis_ad_callback_latency_seconds_bucket{provider="ayet",status="completed",le="0.5"} 0
agis_ad_conversions_by_tier_total{tier="free"} 0
```

### Integration Test Checklist

- [ ] Unit tests passing: `go test ./...`
- [ ] Integration tests passing: `go test -tags=integration ./internal/services`
- [ ] Database migrations applied successfully
- [ ] All pods running and healthy
- [ ] Prometheus scraping metrics (check ServiceMonitor)
- [ ] Grafana dashboard visible
- [ ] Sentry receiving errors (test with manual error)
- [ ] Discord webhooks working (test each channel)
- [ ] Health endpoints responding
- [ ] ayeT S2S callback endpoint accessible

## Rollback Procedure

If deployment fails:

```bash
# Rollback Helm release
helm rollback agis-bot -n production

# Rollback database migrations (if needed)
psql -h <host> -U <user> -d agis -c "
  DROP TABLE IF EXISTS ab_experiments CASCADE;
  DROP TABLE IF EXISTS ab_variants CASCADE;
  DROP TABLE IF EXISTS ab_assignments CASCADE;
  DROP TABLE IF EXISTS ab_events CASCADE;
  DROP TABLE IF EXISTS server_provision_requests CASCADE;
  DROP TABLE IF EXISTS treasury_transactions CASCADE;
  DROP TABLE IF EXISTS consent_records CASCADE;
  DROP TABLE IF EXISTS subscriptions CASCADE;
  DROP TABLE IF EXISTS server_templates CASCADE;
  DELETE FROM schema_migrations WHERE version = 'v2.0-production-enhancements';
"
```

## Monitoring & Alerts

### Key Metrics to Watch

1. **Ad Conversions**:
   - Rate: `rate(agis_ad_conversions_total{status="completed"}[5m])`
   - Fraud rate: `rate(agis_ad_fraud_attempts_total[5m])`

2. **Performance**:
   - Callback latency P95: `histogram_quantile(0.95, rate(agis_ad_callback_latency_seconds_bucket[5m]))`
   - Error rate: `rate(agis_ad_conversions_total{status="error"}[5m])`

3. **Revenue**:
   - Total rewards: `sum(agis_ad_rewards_total)`
   - Revenue by type: `sum(rate(agis_ad_rewards_total[1h])) by (type)`

### Alert Thresholds

| Metric | Warning | Critical |
|--------|---------|----------|
| Conversion rate | <50% | <20% |
| Fraud rate | >5% | >10% |
| Callback latency P95 | >1s | >2s |
| Error rate | >2% | >5% |
| Zero conversions | 15min | 30min |

## Troubleshooting

### Pod CrashLoopBackOff

```bash
kubectl logs -n production deployment/agis-bot --previous
kubectl describe pod -n production -l app=agis-bot
```

Common causes:
- Missing Vault secrets
- Database connection failure
- Invalid Discord token

### Metrics Not Scraping

```bash
# Check ServiceMonitor
kubectl get servicemonitor -n production agis-bot

# Check Prometheus targets
kubectl port-forward -n monitoring svc/prometheus 9090:9090
# Open http://localhost:9090/targets
```

### Database Migration Failed

```bash
# Check current schema version
psql -h <host> -U <user> -d agis -c "SELECT * FROM schema_migrations;"

# Check for migration errors
psql -h <host> -U <user> -d agis -c "
  SELECT table_name, column_name, data_type 
  FROM information_schema.columns 
  WHERE table_name IN ('ab_experiments', 'server_provision_requests')
  ORDER BY table_name, ordinal_position;
"
```

### Sentry Alerts Not Firing

```bash
# Check Sentry DSN is set
kubectl get secret -n production agis-bot-secrets -o jsonpath='{.data.SENTRY_DSN}' | base64 -d

# Test Sentry integration
kubectl exec -n production deployment/agis-bot -- curl -X POST http://localhost:9090/internal/test-sentry
```

## Environment-Specific Notes

### Development
- Single replica
- Local mode database supported
- Integration tests run on every PR

### Staging
- 1 replica
- Uses staging Vault path
- Integration tests run nightly
- ayeT sandbox API keys

### Production
- 2+ replicas for HA
- Production Vault secrets
- Real ayeT production keys
- PagerDuty integration for critical alerts
- Backup job scheduled

## Next Steps

After successful deployment:

1. **Week 1**: Monitor Grafana dashboard daily, tune alert thresholds
2. **Week 2**: Launch first A/B experiment (10% traffic, 1.2x multiplier)
3. **Week 3**: Enable guild provisioning for beta guilds
4. **Month 1**: Review metrics, optimize fraud detection thresholds
5. **Month 2**: Add more server templates based on demand
6. **Month 3**: Implement cost optimization (spot instances, auto-scaling)

## Support

- Documentation: `docs/PRODUCTION_ENHANCEMENTS.md`
- Integration tests: `docs/INTEGRATION_TESTS.md`
- Grafana setup: `docs/GRAFANA_SETUP.md`
- Sentry alerts: `docs/SENTRY_ALERTS.md`

For issues, check logs:
```bash
kubectl logs -n production deployment/agis-bot -f --tail=100
```

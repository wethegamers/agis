> ⚠️ **NOTICE**: This document has been consolidated into the Master Documentation.
> 
> **See**: [operations/docs/manuals/](https://github.com/wethegamers/operations/tree/main/docs/manuals)
>
> This file is kept for reference but may be outdated. The master manuals are the authoritative source.

---


# AGIS Bot - Operations & Maintenance Manual

**Version:** 1.7.0  
**Last Updated:** 2025-01-09  
**Audience:** DevOps Engineers, Site Reliability Engineers, System Administrators

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Deployment](#deployment)
3. [Configuration](#configuration)
4. [Database Management](#database-management)
5. [Monitoring & Alerting](#monitoring--alerting)
6. [Backup & Recovery](#backup--recovery)
7. [Scaling](#scaling)
8. [Security](#security)
9. [Troubleshooting](#troubleshooting)
10. [Maintenance Procedures](#maintenance-procedures)
11. [Incident Response](#incident-response)

---

## Architecture Overview

### System Components

```
┌─────────────────────────────────────────────────────────┐
│                      Discord API                         │
└────────────────┬────────────────────────────────────────┘
                 │
         ┌───────▼────────┐
         │   AGIS Bot     │
         │  (Go Service)  │
         └────┬───┬───┬───┘
              │   │   │
    ┌─────────┘   │   └─────────────┐
    │             │                  │
┌───▼────┐   ┌───▼────┐      ┌─────▼─────┐
│ PostgreSQL │  │  Agones  │      │   Minio   │
│ (Database) │  │(K8s CRD) │      │ (Storage) │
└────────────┘  └──────────┘      └───────────┘
```

### Technology Stack

- **Runtime**: Go 1.23+
- **Database**: PostgreSQL 14+
- **Container Orchestration**: Kubernetes 1.27+
- **Game Server Manager**: Agones 1.35+
- **Object Storage**: Minio (S3-compatible)
- **Payment Processing**: Stripe API
- **Secrets Management**: HashiCorp Vault + ExternalSecrets
- **CI/CD**: GitHub Actions + Argo Workflows
- **Monitoring**: Prometheus + Grafana (recommended)

### Infrastructure Requirements

**Minimum Production Setup**:
- **Bot Service**: 2 replicas, 512MB RAM, 0.5 CPU each
- **PostgreSQL**: 1 instance, 2GB RAM, 1 CPU, 20GB storage
- **Minio**: 1 instance, 1GB RAM, 0.5 CPU, 100GB storage
- **Agones Fleet**: Auto-scaling (0-100 game servers)

**Network Requirements**:
- Ingress for HTTP server (port 9090)
- PostgreSQL port 5432 (internal)
- Minio port 9000 (internal)
- Discord API access (outbound HTTPS)
- Stripe API access (outbound HTTPS)

---

## Deployment

### Prerequisites

1. **Kubernetes Cluster**: 1.27+ with Agones installed
2. **Helm**: 3.0+
3. **kubectl**: Configured with cluster access
4. **External Secrets Operator**: Installed (for Vault integration)
5. **PostgreSQL**: Available (in-cluster or external)

### Initial Deployment

#### 1. Clone Repository

```bash
git clone https://github.com/wethegamers/agis-bot.git
cd agis-bot
```

#### 2. Configure Secrets in Vault

Store secrets in Vault at path: `secret/agis-bot/<environment>`

**Required Secrets**:
```bash
# Discord
DISCORD_TOKEN=<bot-token>
DISCORD_CLIENT_ID=<client-id>
DISCORD_GUILD_ID=<guild-id>

# Database
DB_HOST=<postgres-host>
DB_NAME=agis
DB_USER=agisbot
DB_PASSWORD=<secure-password>

# Stripe
STRIPE_SECRET_KEY=<sk_live_...>
STRIPE_WEBHOOK_SECRET=<whsec_...>
STRIPE_SUCCESS_URL=https://wethegamers.org/payment/success
STRIPE_CANCEL_URL=https://wethegamers.org/payment/cancel

# Minio
S3_ENDPOINT=<minio-endpoint>
S3_ACCESS_KEY=<access-key>
S3_SECRET_KEY=<secret-key>
S3_BUCKET=agis-backups
S3_USE_SSL=true

# Backup Encryption
BACKUP_ENCRYPTION_KEY=<32-char-passphrase>

# Metrics
METRICS_PORT=9090

# WTG Dashboard
WTG_DASHBOARD_URL=https://wethegamers.org

# Admin Roles (comma-separated Discord role IDs)
ADMIN_ROLES=<role-id-1>,<role-id-2>
MOD_ROLES=<role-id-3>,<role-id-4>
```

#### 3. Deploy with Helm

**Development**:
```bash
helm upgrade --install agis-bot charts/agis-bot \
  -n development --create-namespace \
  --set image.tag=latest \
  --set replicaCount=1
```

**Staging**:
```bash
helm upgrade --install agis-bot charts/agis-bot \
  -n staging --create-namespace \
  --set image.tag=v1.7.0 \
  --set replicaCount=2
```

**Production**:
```bash
helm upgrade --install agis-bot charts/agis-bot \
  -n production --create-namespace \
  --set image.tag=v1.7.0 \
  --set replicaCount=3 \
  --set resources.requests.memory=512Mi \
  --set resources.limits.memory=1Gi
```

#### 4. Verify Deployment

```bash
# Check pod status
kubectl get pods -n production

# Check logs
kubectl logs -n production -l app=agis-bot --tail=100

# Check service
kubectl get svc -n production

# Test health endpoint
kubectl port-forward -n production svc/agis-bot 9090:9090
curl http://localhost:9090/health
```

### CI/CD Pipeline

**Automated Deployment Flow**:
1. Push to `main` branch triggers GitHub Actions
2. GitHub Actions submits Argo Workflow
3. Argo Workflow builds multi-arch image (linux/amd64, linux/arm64)
4. Image pushed to GHCR (ghcr.io/wethegamers/agis-bot)
5. Argo Workflow deploys to Development
6. Manual approval for Staging
7. Manual approval for Production
8. Discord notifications at each stage

**Manual Trigger**:
```bash
# Via GitHub CLI
gh workflow run build-and-push.yml

# Via Argo
argo submit .argo/publish.yaml
```

---

## Configuration

### Environment Variables

See [Deployment Prerequisites](#2-configure-secrets-in-vault) for all environment variables.

**Critical Variables**:
- `DISCORD_TOKEN`: Bot authentication (rotate quarterly)
- `DB_PASSWORD`: Database password (rotate monthly)
- `STRIPE_SECRET_KEY`: Payment processing (do not rotate)
- `STRIPE_WEBHOOK_SECRET`: Webhook verification (do not rotate)

### Helm Chart Configuration

**Key Values** (`charts/agis-bot/values.yaml`):

```yaml
# Replica count (2-3 for HA)
replicaCount: 2

# Image settings
image:
  repository: ghcr.io/wethegamers/agis-bot
  tag: v1.7.0
  pullPolicy: IfNotPresent

# Resource limits
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "1Gi"
    cpu: "1000m"

# Health checks
livenessProbe:
  httpGet:
    path: /health
    port: 9090
  initialDelaySeconds: 30
  periodSeconds: 10

# Autoscaling (optional)
autoscaling:
  enabled: false
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80
```

### Database Configuration

**Connection Pooling** (Go application):
```go
// Recommended settings for production
MaxOpenConns: 25
MaxIdleConns: 5
ConnMaxLifetime: 5 * time.Minute
```

**PostgreSQL Tuning**:
```sql
-- postgresql.conf
max_connections = 100
shared_buffers = 256MB
effective_cache_size = 1GB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
```

---

## Database Management

### Schema Migrations

**Location**: `internal/database/migrations/`

**Migration Files**:
```
001_initial_schema.sql
002_public_servers.sql
003_achievements.sql
004_shop_system.sql
005_guild_treasury.sql
006_server_reviews.sql
```

**Apply Migrations**:

```bash
# Manual application
psql $DATABASE_URL -f internal/database/migrations/001_initial_schema.sql

# Or use migration tool (recommended)
migrate -path internal/database/migrations \
        -database "postgres://user:pass@host/db?sslmode=require" up
```

**Rollback** (if needed):
```sql
-- Manually drop tables in reverse order
DROP TABLE IF EXISTS server_reviews;
DROP TABLE IF EXISTS guild_servers;
DROP TABLE IF EXISTS guild_members;
DROP TABLE IF EXISTS guild_treasury;
-- etc.
```

### Seed Data

**Pricing Configuration**:
```bash
psql $DATABASE_URL -f internal/database/seeds/pricing_seed.sql
```

This seeds 16 game types with accurate pricing. Safe to re-run (uses `ON CONFLICT DO UPDATE`).

**Shop Items**:
```bash
psql $DATABASE_URL -f scripts/seed-wtg-shop.sql
```

### Database Backups

**Automated Backups** (daily via cron):
```bash
#!/bin/bash
# /opt/scripts/backup-agis-db.sh

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="agis_backup_${TIMESTAMP}.sql.gz"

pg_dump "$DATABASE_URL" | gzip > "/backups/${BACKUP_FILE}"

# Upload to S3
aws s3 cp "/backups/${BACKUP_FILE}" s3://agis-backups/database/

# Retain 30 days
find /backups/ -name "agis_backup_*.sql.gz" -mtime +30 -delete
```

**Restore from Backup**:
```bash
# Download from S3
aws s3 cp s3://agis-backups/database/agis_backup_20250109_120000.sql.gz .

# Restore (WARNING: drops existing database)
gunzip < agis_backup_20250109_120000.sql.gz | psql $DATABASE_URL
```

### Database Maintenance

**Vacuum** (weekly):
```sql
VACUUM ANALYZE;
```

**Reindex** (monthly):
```sql
REINDEX DATABASE agis;
```

**Check Table Sizes**:
```sql
SELECT 
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

---

## Monitoring & Alerting

### Health Endpoints

**HTTP Server** (port 9090):
- `/health`, `/healthz` - Basic health check
- `/ready`, `/readyz` - Readiness check (database connection)
- `/info`, `/about`, `/version` - Build information
- `/metrics` - Prometheus metrics

**Health Check**:
```bash
curl http://agis-bot.production.svc:9090/health
# Expected: {"status":"ok"}

curl http://agis-bot.production.svc:9090/ready
# Expected: {"status":"ready","database":"connected"}
```

### Prometheus Metrics

**Exposed Metrics**:
- `agis_bot_commands_total` - Total commands executed (by command name)
- `agis_bot_errors_total` - Total errors encountered
- `agis_bot_servers_total` - Active game servers (by game type)
- `agis_bot_users_total` - Registered users
- `agis_bot_premium_subscriptions` - Active premium subscriptions
- `agis_bot_revenue_cents` - Monthly recurring revenue

**Prometheus Scrape Config**:
```yaml
scrape_configs:
  - job_name: 'agis-bot'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - production
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: keep
        regex: agis-bot
      - source_labels: [__meta_kubernetes_pod_container_port_number]
        action: keep
        regex: "9090"
```

### Recommended Alerts

**Critical Alerts**:

```yaml
# AlertManager rules
groups:
  - name: agis-bot
    rules:
      - alert: AGISBotDown
        expr: up{job="agis-bot"} == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "AGIS Bot is down"
          description: "AGIS Bot has been down for 5 minutes"

      - alert: DatabaseConnectionFailed
        expr: agis_bot_database_connected == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Database connection failed"
          description: "AGIS Bot cannot connect to PostgreSQL"

      - alert: HighErrorRate
        expr: rate(agis_bot_errors_total[5m]) > 1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors/sec"

      - alert: PaymentWebhookFailures
        expr: rate(agis_bot_stripe_webhook_failures_total[1h]) > 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Stripe webhook failures detected"
          description: "Payment processing may be disrupted"
```

### Logging

**Log Levels**:
- `INFO`: Normal operations (server created, payment processed)
- `WARN`: Recoverable errors (rate limit hit, timeout)
- `ERROR`: Failures requiring attention (database error, API failure)

**Log Aggregation**:
```bash
# Stream logs from all replicas
kubectl logs -n production -l app=agis-bot -f --tail=100

# Search logs
kubectl logs -n production -l app=agis-bot | grep ERROR

# Export to file
kubectl logs -n production deploy/agis-bot --since=24h > agis-logs-$(date +%Y%m%d).log
```

**Recommended**: Use ELK Stack or Loki for centralized logging.

---

## Backup & Recovery

### Application Backups

**Server Saves** (Minio):
- Automatic: User-initiated via `export` command
- Storage: 30-day retention in Minio bucket
- Encryption: AES-256-GCM
- Compression: gzip

**Monitor Backup Storage**:
```bash
# Check Minio bucket size
mc du production/agis-backups

# List recent backups
mc ls production/agis-backups --recursive | tail -20
```

### Database Backups

See [Database Backups](#database-backups) section.

**Backup Verification** (monthly):
```bash
# Restore to test environment
./scripts/restore-test-db.sh

# Run smoke tests
./scripts/verify-backup.sh
```

### Disaster Recovery

**RTO (Recovery Time Objective)**: 30 minutes  
**RPO (Recovery Point Objective)**: 24 hours

**DR Procedure**:

1. **Total Cluster Loss**:
   ```bash
   # 1. Provision new Kubernetes cluster
   # 2. Install Agones, Vault, ExternalSecrets
   # 3. Restore database from latest backup
   gunzip < latest_backup.sql.gz | psql $NEW_DATABASE_URL
   
   # 4. Deploy AGIS Bot
   helm install agis-bot charts/agis-bot -n production \
     --set database.host=$NEW_DB_HOST
   
   # 5. Verify health
   kubectl get pods -n production
   curl http://agis-bot/health
   ```

2. **Database Corruption**:
   ```bash
   # 1. Stop bot to prevent writes
   kubectl scale deploy/agis-bot -n production --replicas=0
   
   # 2. Restore from backup
   gunzip < backup.sql.gz | psql $DATABASE_URL
   
   # 3. Verify data integrity
   psql $DATABASE_URL -c "SELECT COUNT(*) FROM users;"
   
   # 4. Restart bot
   kubectl scale deploy/agis-bot -n production --replicas=3
   ```

---

## Scaling

### Horizontal Scaling

**Manual Scaling**:
```bash
# Scale up
kubectl scale deploy/agis-bot -n production --replicas=5

# Scale down
kubectl scale deploy/agis-bot -n production --replicas=2
```

**Auto-Scaling** (HPA):
```yaml
# values.yaml
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80
```

**Recommended Scaling Thresholds**:
- **2 replicas**: Up to 1,000 concurrent users
- **5 replicas**: 1,000-5,000 concurrent users
- **10 replicas**: 5,000+ concurrent users

### Database Scaling

**Read Replicas** (PostgreSQL):
```bash
# Route read-only queries to replica
export DB_READ_HOST=postgres-replica.production.svc
```

**Connection Pooling** (PgBouncer):
```yaml
# Recommended for >1000 concurrent connections
apiVersion: v1
kind: Service
metadata:
  name: pgbouncer
spec:
  selector:
    app: pgbouncer
  ports:
    - port: 5432
```

### Game Server Scaling

**Agones Fleet Autoscaling**:
```yaml
apiVersion: autoscaling.agones.dev/v1
kind: FleetAutoscaler
metadata:
  name: agis-gameservers
spec:
  fleetName: agis-fleet
  policy:
    type: Buffer
    buffer:
      bufferSize: 5
      minReplicas: 0
      maxReplicas: 100
```

---

## Security

### Access Control

**RBAC** (Kubernetes):
```yaml
# charts/agis-bot/templates/rbac.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: agis-bot
rules:
  - apiGroups: ["agones.dev"]
    resources: ["gameservers", "fleets"]
    verbs: ["get", "list", "create", "delete"]
```

**Discord Permissions**:
- Bot requires: `SEND_MESSAGES`, `EMBED_LINKS`, `READ_MESSAGE_HISTORY`
- Slash commands: `applications.commands` scope

### Secrets Management

**Vault Integration**:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: agis-bot-secrets
spec:
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: agis-bot-secrets
  data:
    - secretKey: DISCORD_TOKEN
      remoteRef:
        key: secret/agis-bot/production
        property: DISCORD_TOKEN
```

**Secret Rotation**:
- **Discord Token**: Rotate quarterly
- **Database Password**: Rotate monthly
- **Stripe Keys**: Never rotate (breaks webhooks)
- **Backup Encryption Key**: Rotate annually

### Network Security

**Network Policies**:
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: agis-bot
spec:
  podSelector:
    matchLabels:
      app: agis-bot
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: prometheus
      ports:
        - port: 9090
  egress:
    - to:
        - podSelector:
            matchLabels:
              app: postgresql
      ports:
        - port: 5432
    - to:  # Discord API
        - namespaceSelector: {}
      ports:
        - port: 443
```

### Vulnerability Scanning

**Container Scanning**:
```bash
# Trivy scan
trivy image ghcr.io/wethegamers/agis-bot:v1.7.0

# Grype scan
grype ghcr.io/wethegamers/agis-bot:v1.7.0
```

**Dependency Audit**:
```bash
# Go modules
go list -json -m all | nancy sleuth

# Or use govulncheck
govulncheck ./...
```

---

## Troubleshooting

### Common Issues

#### Bot Not Responding

**Symptoms**: Commands don't work, bot shows offline

**Diagnosis**:
```bash
# Check pod status
kubectl get pods -n production -l app=agis-bot

# Check logs
kubectl logs -n production -l app=agis-bot --tail=100

# Check Discord API status
curl https://status.discord.com/api/v2/status.json
```

**Solutions**:
1. Verify DISCORD_TOKEN is valid
2. Check bot has proper permissions in Discord
3. Ensure pod is running (`CrashLoopBackOff` = config issue)
4. Verify network connectivity to Discord API

#### Database Connection Errors

**Symptoms**: "failed to connect to database" errors

**Diagnosis**:
```bash
# Test from bot pod
kubectl exec -it -n production deploy/agis-bot -- /bin/sh
nc -zv $DB_HOST 5432

# Check PostgreSQL
kubectl logs -n production -l app=postgresql
```

**Solutions**:
1. Verify DB_HOST, DB_USER, DB_PASSWORD secrets
2. Check PostgreSQL is running
3. Verify network policies allow connection
4. Check PostgreSQL max_connections limit

#### Payment Webhooks Failing

**Symptoms**: Payments don't credit users

**Diagnosis**:
```bash
# Check Stripe webhook logs
kubectl logs -n production -l app=agis-bot | grep "stripe webhook"

# Verify webhook secret
kubectl get secret agis-bot-secrets -n production -o jsonpath='{.data.STRIPE_WEBHOOK_SECRET}' | base64 -d
```

**Solutions**:
1. Verify STRIPE_WEBHOOK_SECRET matches Stripe dashboard
2. Check webhook endpoint is accessible (Stripe dashboard > Webhooks > Recent Deliveries)
3. Ensure signature verification logic is correct
4. Check transaction logs in database

#### High Memory Usage

**Symptoms**: Pods getting OOMKilled

**Diagnosis**:
```bash
# Check resource usage
kubectl top pods -n production -l app=agis-bot

# Check memory profile
kubectl exec -it -n production deploy/agis-bot -- curl http://localhost:9090/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

**Solutions**:
1. Increase memory limits in Helm values
2. Check for memory leaks (profiling)
3. Reduce cache sizes (pricing cache, etc.)
4. Scale horizontally instead

### Debug Mode

**Enable Verbose Logging**:
```bash
# Temporarily (restart required)
kubectl set env -n production deploy/agis-bot LOG_LEVEL=DEBUG

# Or update Helm values
helm upgrade agis-bot charts/agis-bot -n production \
  --set env.LOG_LEVEL=DEBUG
```

### Support Escalation

**Severity Levels**:
- **P0 (Critical)**: Bot down, payments broken - Page on-call engineer
- **P1 (High)**: Major feature broken - Respond within 1 hour
- **P2 (Medium)**: Minor feature broken - Respond within 4 hours
- **P3 (Low)**: Cosmetic issue - Respond within 24 hours

---

## Maintenance Procedures

### Routine Maintenance Schedule

**Daily**:
- [ ] Check health endpoints
- [ ] Review error logs
- [ ] Monitor subscription stats

**Weekly**:
- [ ] Database VACUUM
- [ ] Review Prometheus alerts
- [ ] Check backup success
- [ ] Review resource usage

**Monthly**:
- [ ] Database REINDEX
- [ ] Rotate database password
- [ ] Review and update pricing
- [ ] Verify backup restoration
- [ ] Update dependencies

**Quarterly**:
- [ ] Rotate Discord token
- [ ] Review and optimize queries
- [ ] Load testing
- [ ] Disaster recovery drill

### Updating the Bot

**Zero-Downtime Deployment**:

```bash
# 1. Deploy new version (rolling update)
helm upgrade agis-bot charts/agis-bot -n production \
  --set image.tag=v1.7.1 \
  --wait

# 2. Verify new pods are healthy
kubectl get pods -n production -l app=agis-bot

# 3. Monitor for errors
kubectl logs -n production -l app=agis-bot -f

# 4. Rollback if issues
helm rollback agis-bot -n production
```

**Database Schema Updates**:

```bash
# 1. Scale down to 1 replica (prevents concurrent migrations)
kubectl scale deploy/agis-bot -n production --replicas=1

# 2. Apply migration
kubectl exec -it -n production deploy/agis-bot -- \
  psql $DATABASE_URL -f /app/migrations/007_new_feature.sql

# 3. Scale back up
kubectl scale deploy/agis-bot -n production --replicas=3
```

### Pricing Updates

**Update Game Pricing** (zero downtime):

```bash
# Via admin command in Discord
@AGIS pricing update rust 250

# Or via SQL
psql $DATABASE_URL -c "UPDATE pricing_config SET cost_per_hour = 250 WHERE game_type = 'rust';"
```

No bot restart required - pricing cache updates within 5 minutes.

---

## Incident Response

### Incident Response Plan

**Step 1: Detection**
- Alert via Prometheus/AlertManager
- User reports in Discord #support
- Automated monitoring

**Step 2: Assessment**
- Check severity (P0-P3)
- Determine impact (users affected)
- Estimate resolution time

**Step 3: Communication**
- Post in Discord #status channel
- Update status page (if available)
- Notify stakeholders

**Step 4: Mitigation**
- Apply immediate fix or rollback
- Document actions taken
- Verify resolution

**Step 5: Post-Mortem**
- Write incident report
- Identify root cause
- Create action items
- Schedule follow-up

### Example Incidents

#### Database Outage

**Runbook**:
1. Scale bot to 0 replicas (stop new writes)
2. Investigate database (disk full? connection limit?)
3. Fix root cause (expand disk, increase connections)
4. Verify database health
5. Scale bot back up
6. Monitor for errors

#### Payment Webhook Failure

**Runbook**:
1. Check Stripe dashboard for failed webhooks
2. Retry failed webhooks manually if <50
3. If >50, write script to reconcile payments:
   ```sql
   -- Find users who paid but didn't get credited
   SELECT * FROM credit_transactions 
   WHERE transaction_type = 'purchase' 
   AND created_at > NOW() - INTERVAL '24 hours';
   ```
4. Manual credit application if needed
5. Fix webhook issue (secret, endpoint)
6. Monitor for 24 hours

---

## Appendices

### A. Useful Commands

```bash
# Quick health check
kubectl get pods -n production && \
curl http://agis-bot.production.svc:9090/health

# Tail logs across all pods
kubectl logs -n production -l app=agis-bot -f --tail=50

# Execute SQL query
kubectl exec -it -n production deploy/agis-bot -- \
  psql $DATABASE_URL -c "SELECT COUNT(*) FROM users;"

# Port forward for local debugging
kubectl port-forward -n production svc/agis-bot 9090:9090

# Get current pricing config
kubectl exec -it -n production deploy/agis-bot -- \
  psql $DATABASE_URL -c "SELECT * FROM pricing_config ORDER BY cost_per_hour;"
```

### B. Monitoring Dashboards

**Grafana Dashboard JSON**: Available at `docs/grafana-dashboard.json` (TODO)

**Key Panels**:
- Active Users (last 24h)
- Commands/second
- Error Rate
- Active Game Servers
- Premium Subscriptions
- Monthly Revenue

### C. Contact Information

**On-Call Rotation**: PagerDuty schedule  
**Slack Channel**: #agis-bot-ops  
**Documentation**: https://docs.wethegamers.org  
**GitHub**: https://github.com/wethegamers/agis-bot  

---

**Document Version**: 1.0  
**Maintained By**: DevOps Team  
**Next Review**: 2025-04-09

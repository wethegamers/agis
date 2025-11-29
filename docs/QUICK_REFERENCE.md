> ⚠️ **NOTICE**: This document has been consolidated into the Master Documentation.
> 
> **See**: [operations/docs/manuals/](https://github.com/wethegamers/operations/tree/main/docs/manuals)
>
> This file is kept for reference but may be outdated. The master manuals are the authoritative source.

---


# AGIS Bot - Quick Reference Card

**Version:** 1.7.0  
**Print-Ready Reference for Operations**

---

## Critical Contacts

| Role | Contact |
|------|---------|
| **On-Call** | PagerDuty rotation |
| **Slack** | #agis-bot-ops |
| **GitHub** | github.com/wethegamers/agis-bot |
| **Docs** | docs.wethegamers.org |

---

## Health Check (30 seconds)

```bash
# 1. Pod status
kubectl get pods -n production -l app=agis-bot

# 2. Health endpoint
kubectl port-forward -n production svc/agis-bot 9090:9090 &
curl http://localhost:9090/health

# 3. Error logs
kubectl logs -n production -l app=agis-bot --tail=20 | grep ERROR

# 4. User count
kubectl exec -it -n production deploy/agis-bot -- \
  psql $DATABASE_URL -c "SELECT COUNT(*) FROM users;"
```

**Expected**:
- ✅ 2-3 pods `Running`
- ✅ `/health` returns `{"status":"ok"}`
- ✅ No recent ERROR logs
- ✅ User count increasing

---

## Emergency Procedures

### Bot Down (P0)

```bash
# 1. Check pods
kubectl get pods -n production -l app=agis-bot

# 2. If CrashLoopBackOff, check logs
kubectl logs -n production -l app=agis-bot --tail=100

# 3. Common fix: restart
kubectl rollout restart deploy/agis-bot -n production

# 4. If DB issue, verify connection
kubectl exec -it -n production deploy/agis-bot -- nc -zv $DB_HOST 5432

# 5. Escalate if not resolved in 5 minutes
```

### Database Down (P0)

```bash
# 1. Stop bot (prevent connection spam)
kubectl scale deploy/agis-bot -n production --replicas=0

# 2. Check PostgreSQL
kubectl get pods -n production -l app=postgresql
kubectl logs -n production -l app=postgresql --tail=100

# 3. Restore database if corrupted
gunzip < latest_backup.sql.gz | psql $DATABASE_URL

# 4. Restart bot
kubectl scale deploy/agis-bot -n production --replicas=3

# 5. Monitor logs
kubectl logs -n production -l app=agis-bot -f
```

### Payment Webhooks Failing (P0)

```bash
# 1. Check Stripe dashboard
# https://dashboard.stripe.com/webhooks

# 2. Verify webhook secret
kubectl get secret agis-bot-secrets -n production \
  -o jsonpath='{.data.STRIPE_WEBHOOK_SECRET}' | base64 -d

# 3. Check webhook logs
kubectl logs -n production -l app=agis-bot | grep "stripe webhook"

# 4. Manual credit application (if needed)
kubectl exec -it -n production deploy/agis-bot -- \
  psql $DATABASE_URL -c \
  "UPDATE users SET wtg_coins = wtg_coins + 5 WHERE discord_id = '123456789';"

# 5. Notify users in Discord #status
```

---

## Common Operations

### Deploy New Version

```bash
# Zero-downtime rolling update
helm upgrade agis-bot charts/agis-bot -n production \
  --set image.tag=v1.7.1 \
  --wait

# Verify
kubectl get pods -n production -l app=agis-bot
kubectl logs -n production -l app=agis-bot --tail=50

# Rollback if issues
helm rollback agis-bot -n production
```

### Scale Up/Down

```bash
# Scale up
kubectl scale deploy/agis-bot -n production --replicas=5

# Scale down
kubectl scale deploy/agis-bot -n production --replicas=2

# Auto-scale (edit Helm values)
helm upgrade agis-bot charts/agis-bot -n production \
  --set autoscaling.enabled=true \
  --set autoscaling.minReplicas=2 \
  --set autoscaling.maxReplicas=10
```

### Update Game Pricing

```bash
# Via Discord admin command
@AGIS pricing update rust 250

# Or via SQL (no restart needed)
kubectl exec -it -n production deploy/agis-bot -- \
  psql $DATABASE_URL -c \
  "UPDATE pricing_config SET cost_per_hour = 250 WHERE game_type = 'rust';"

# Verify
kubectl exec -it -n production deploy/agis-bot -- \
  psql $DATABASE_URL -c \
  "SELECT game_type, cost_per_hour FROM pricing_config ORDER BY cost_per_hour;"
```

### Database Backup

```bash
# Manual backup
kubectl exec -it -n production deploy/postgres -- \
  pg_dump -U agisbot agis | gzip > backup_$(date +%Y%m%d).sql.gz

# Upload to S3
aws s3 cp backup_$(date +%Y%m%d).sql.gz s3://agis-backups/database/

# Verify backup
gunzip -t backup_$(date +%Y%m%d).sql.gz
```

### Database Restore

```bash
# WARNING: This drops existing data!

# 1. Stop bot
kubectl scale deploy/agis-bot -n production --replicas=0

# 2. Download backup
aws s3 cp s3://agis-backups/database/backup_20250109.sql.gz .

# 3. Restore
gunzip < backup_20250109.sql.gz | \
  kubectl exec -i -n production deploy/postgres -- \
  psql -U agisbot agis

# 4. Verify
kubectl exec -it -n production deploy/postgres -- \
  psql -U agisbot agis -c "SELECT COUNT(*) FROM users;"

# 5. Restart bot
kubectl scale deploy/agis-bot -n production --replicas=3
```

### View Logs

```bash
# Tail all replicas
kubectl logs -n production -l app=agis-bot -f --tail=100

# Search for errors
kubectl logs -n production -l app=agis-bot --since=1h | grep ERROR

# Export to file
kubectl logs -n production deploy/agis-bot --since=24h > \
  agis-logs-$(date +%Y%m%d).log

# Specific pod
kubectl logs -n production agis-bot-6f8b9c7d-xk2lm --tail=200
```

---

## Monitoring Queries

### Prometheus Metrics

```bash
# Port forward Prometheus
kubectl port-forward -n monitoring svc/prometheus 9090:9090 &

# Open browser
open http://localhost:9090
```

**Key Metrics**:
- `up{job="agis-bot"}` - Bot uptime
- `agis_bot_commands_total` - Commands executed
- `agis_bot_servers_total` - Active servers
- `agis_bot_premium_subscriptions` - Premium users
- `agis_bot_revenue_cents` - Monthly revenue

### Database Queries

```bash
# Active users (last 24h)
kubectl exec -it -n production deploy/agis-bot -- \
  psql $DATABASE_URL -c \
  "SELECT COUNT(DISTINCT discord_id) FROM credit_transactions 
   WHERE created_at > NOW() - INTERVAL '24 hours';"

# Premium subscribers
kubectl exec -it -n production deploy/agis-bot -- \
  psql $DATABASE_URL -c \
  "SELECT COUNT(*) FROM users 
   WHERE tier = 'premium' AND subscription_expires > NOW();"

# Active game servers
kubectl exec -it -n production deploy/agis-bot -- \
  psql $DATABASE_URL -c \
  "SELECT game_type, COUNT(*) FROM game_servers 
   WHERE status IN ('running', 'ready') GROUP BY game_type;"

# Today's revenue
kubectl exec -it -n production deploy/agis-bot -- \
  psql $DATABASE_URL -c \
  "SELECT SUM(amount)/1000.0 AS revenue_usd FROM credit_transactions 
   WHERE transaction_type = 'purchase' 
   AND created_at > CURRENT_DATE;"

# Table sizes
kubectl exec -it -n production deploy/agis-bot -- \
  psql $DATABASE_URL -c \
  "SELECT tablename, 
   pg_size_pretty(pg_total_relation_size('public.'||tablename)) AS size 
   FROM pg_tables WHERE schemaname = 'public' 
   ORDER BY pg_total_relation_size('public.'||tablename) DESC;"
```

---

## Configuration

### Environment Variables (Vault)

```bash
# View current secrets (base64 encoded)
kubectl get secret agis-bot-secrets -n production -o yaml

# Decode specific secret
kubectl get secret agis-bot-secrets -n production \
  -o jsonpath='{.data.DISCORD_TOKEN}' | base64 -d

# Update secret in Vault
vault kv put secret/agis-bot/production DISCORD_TOKEN=new_token

# Force ExternalSecret sync
kubectl annotate externalsecret agis-bot-secrets -n production \
  force-sync=$(date +%s) --overwrite
```

### Helm Values

```bash
# View current values
helm get values agis-bot -n production

# Update values
helm upgrade agis-bot charts/agis-bot -n production \
  --set replicaCount=5 \
  --set resources.limits.memory=2Gi

# Full values file
helm upgrade agis-bot charts/agis-bot -n production \
  -f custom-values.yaml
```

---

## Useful Aliases

Add to `~/.bashrc`:

```bash
# AGIS Bot shortcuts
alias agis-pods='kubectl get pods -n production -l app=agis-bot'
alias agis-logs='kubectl logs -n production -l app=agis-bot -f --tail=100'
alias agis-health='curl http://agis-bot.production.svc:9090/health'
alias agis-scale='kubectl scale deploy/agis-bot -n production --replicas='
alias agis-restart='kubectl rollout restart deploy/agis-bot -n production'
alias agis-db='kubectl exec -it -n production deploy/agis-bot -- psql $DATABASE_URL'
```

---

## Maintenance Schedule

### Daily
- [ ] `agis-health` - Check health
- [ ] `agis-logs | grep ERROR` - Review errors

### Weekly
- [ ] Database VACUUM
- [ ] Check backup success (S3 bucket)
- [ ] Review Prometheus alerts

### Monthly
- [ ] Database REINDEX
- [ ] Rotate DB password
- [ ] Verify backup restore
- [ ] Review pricing config

### Quarterly
- [ ] Rotate Discord token
- [ ] Load testing
- [ ] DR drill

---

## Troubleshooting Matrix

| Symptom | Check | Fix |
|---------|-------|-----|
| Bot offline | `agis-pods` | `agis-restart` |
| High latency | `kubectl top pods` | Scale up |
| DB errors | PostgreSQL logs | Check credentials |
| OOMKilled | Memory usage | Increase limits |
| Payment fail | Stripe dashboard | Verify webhook secret |
| Commands slow | `agis_bot_commands_total` | Scale horizontally |

---

## Support Escalation

| Severity | Response Time | Action |
|----------|---------------|--------|
| **P0** | Immediate | Page on-call |
| **P1** | 1 hour | Slack #agis-bot-ops |
| **P2** | 4 hours | Create ticket |
| **P3** | 24 hours | Add to backlog |

---

## Key Endpoints

| Endpoint | Port | Purpose |
|----------|------|---------|
| `/health` | 9090 | Liveness probe |
| `/ready` | 9090 | Readiness probe |
| `/metrics` | 9090 | Prometheus scrape |
| `/version` | 9090 | Build info |
| `/info` | 9090 | Bot statistics |

---

## Critical Files

| File | Purpose |
|------|---------|
| `charts/agis-bot/` | Helm deployment |
| `internal/database/migrations/` | DB schema |
| `internal/database/seeds/` | Seed data |
| `docs/OPS_MANUAL.md` | Full ops guide |
| `docs/USER_GUIDE.md` | User documentation |

---

**Keep this card accessible during on-call shifts!**  
**Last Updated:** 2025-01-09

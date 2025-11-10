# Integration Guide - v1.7.0 Features

This guide explains how to integrate the newly created REST API and Scheduler features into the main application.

## 1. Update go.mod Dependencies

Add the following dependencies:

```bash
cd /home/seb/wtg/agis-bot
go get github.com/gorilla/mux
go get github.com/robfig/cron/v3
go mod tidy
```

## 2. Run Database Migration

```bash
# Connect to your PostgreSQL database
psql -U agis_user -d agis_bot -f deployments/migrations/v1.7.0-rest-api-scheduling.sql

# Or via Docker if running in container
kubectl exec -it <postgres-pod> -- psql -U agis_user -d agis_bot < deployments/migrations/v1.7.0-rest-api-scheduling.sql
```

Verify migration:
```sql
SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename IN ('server_schedules', 'api_keys', 'user_stats');
```

## 3. Update main.go

### Add Imports
```go
import (
    "agis-bot/internal/api"
    "agis-bot/internal/services/scheduler"
    "github.com/gorilla/mux"
)
```

### Initialize Services in main()

**After database initialization:**
```go
// Initialize scheduler service
schedulerService := scheduler.NewSchedulerService(db, agonesClient)
if err := schedulerService.Start(); err != nil {
    log.Fatalf("Failed to start scheduler: %v", err)
}
defer schedulerService.Stop()

log.Println("✅ Scheduler service started")
```

**After enhanced server service initialization:**
```go
// Initialize REST API server
apiServer := api.NewAPIServer(":8080", db, agonesClient, enhancedServerService)
go func() {
    log.Println("🚀 Starting REST API server on :8080")
    if err := apiServer.Start(); err != nil {
        log.Fatalf("Failed to start API server: %v", err)
    }
}()
```

### Add Scheduler to CommandContext

Update the command context initialization:
```go
ctx := &bot.CommandContext{
    DB:                 db,
    Agones:            agonesClient,
    EnhancedServer:    enhancedServerService,
    Scheduler:         schedulerService,  // Add this line
    // ... other fields
}
```

### Register Schedule Command

In the command registration section:
```go
commands := map[string]bot.Command{
    "server":   &commands.ServerCommand{},
    "credits":  &commands.CreditsCommand{},
    "schedule": &commands.ScheduleCommand{},  // Add this line
    // ... other commands
}
```

## 4. Update CommandContext Interface

**File:** `internal/bot/context.go`

Add the Scheduler field:
```go
type CommandContext struct {
    DB                *services.DatabaseService
    Agones           *services.AgonesService
    EnhancedServer   *services.EnhancedServerService
    Scheduler        *scheduler.SchedulerService  // Add this line
    // ... other fields
}
```

## 5. Import Grafana Dashboards

### Option A: Via Grafana UI
1. Navigate to Grafana: `http://grafana.wethegamers.org`
2. Click **Dashboards** → **Import**
3. Upload `deployments/grafana/agis-bot-overview.json`
4. Select Prometheus datasource
5. Click **Import**
6. Repeat for `agis-bot-revenue.json`

### Option B: Via ConfigMap (Kubernetes)
```bash
kubectl create configmap grafana-dashboards \
  --from-file=agis-bot-overview.json=deployments/grafana/agis-bot-overview.json \
  --from-file=agis-bot-revenue.json=deployments/grafana/agis-bot-revenue.json \
  -n monitoring
```

Update Grafana deployment to mount the ConfigMap:
```yaml
volumeMounts:
  - name: dashboards
    mountPath: /var/lib/grafana/dashboards
volumes:
  - name: dashboards
    configMap:
      name: grafana-dashboards
```

## 6. Configure Prometheus Metrics

Ensure these metrics are exposed in your code:

```go
// Existing metrics (verify in your metrics.go)
agis_bot_active_users
agis_bot_servers_active{game_type="minecraft"}
agis_bot_commands_total{command="server"}
agis_bot_credits_transactions
agis_bot_database_operations_total{operation="insert"}

// New metrics needed for revenue dashboard
agis_ad_conversions_total{type="offerwall",status="completed"}
agis_ad_rewards_total{type="offerwall"}
agis_ad_callback_latency_seconds{provider="offertoro"}
agis_ad_fraud_detected_total
```

Add these to your metrics collection if missing:
```go
var (
    adConversions = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "agis_ad_conversions_total",
            Help: "Total ad conversions",
        },
        []string{"type", "status"},
    )
    
    adRewards = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "agis_ad_rewards_total",
            Help: "Total GC rewarded from ads",
        },
        []string{"type"},
    )
    
    // ... other metrics
)

func init() {
    prometheus.MustRegister(adConversions)
    prometheus.MustRegister(adRewards)
    // ...
}
```

## 7. Testing

### Test REST API
```bash
# Get your Discord ID
DISCORD_ID="your_discord_id_here"

# Test user endpoint
curl -H "Authorization: Bearer $DISCORD_ID" \
  http://localhost:8080/api/v1/users/me

# Test list servers
curl -H "Authorization: Bearer $DISCORD_ID" \
  http://localhost:8080/api/v1/servers

# Test create server
curl -X POST \
  -H "Authorization: Bearer $DISCORD_ID" \
  -H "Content-Type: application/json" \
  -d '{"game_type":"minecraft","server_name":"test-server"}' \
  http://localhost:8080/api/v1/servers
```

### Test Scheduler via Discord
```
!schedule start my-server 0 8 * * * daily startup at 8am
!schedule list my-server
!schedule disable <schedule-id>
!schedule enable <schedule-id>
!schedule delete <schedule-id>
```

### Test Grafana Dashboards
1. Open Grafana dashboards
2. Verify all panels load without errors
3. Check data is flowing from Prometheus
4. Test time range selector
5. Verify refresh works

## 8. Deployment Checklist

- [ ] Dependencies added to go.mod
- [ ] Database migration applied
- [ ] main.go updated with API server
- [ ] main.go updated with scheduler service
- [ ] CommandContext updated with Scheduler field
- [ ] Schedule command registered
- [ ] Grafana dashboards imported
- [ ] Prometheus metrics verified
- [ ] REST API tested locally
- [ ] Scheduler tested via Discord
- [ ] Grafana dashboards displaying data
- [ ] Production deployment planned
- [ ] Documentation updated
- [ ] Changelog updated

## 9. Rollback Plan

If issues occur:

### Rollback Database
```sql
DROP TABLE IF EXISTS server_schedules CASCADE;
DROP TABLE IF EXISTS api_keys CASCADE;
DROP TABLE IF EXISTS user_stats CASCADE;
```

### Rollback Code
```bash
git revert <commit-hash>
git push origin main
```

### Remove Grafana Dashboards
Delete via Grafana UI or:
```bash
kubectl delete configmap grafana-dashboards -n monitoring
```

## 10. Monitoring Post-Deployment

Watch these metrics:
- API response times: `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`
- Scheduler execution success rate: `rate(scheduler_executions_total{status="success"}[5m])`
- Database connection pool: `database_connections_active`
- Error rates: `rate(errors_total[5m])`

Check logs:
```bash
kubectl logs -f deployment/agis-bot -n default | grep -E "(API|Scheduler)"
```

## 11. Next Steps

After successful integration:

1. **API Keys Implementation** (v1.8.0)
   - Implement proper API key generation
   - Add key management endpoints
   - Update authentication middleware

2. **Rate Limiting** (v1.8.0)
   - Add Redis for distributed rate limiting
   - Implement tiered rate limits
   - Add rate limit headers

3. **Server Actions** (v1.7.1)
   - Complete start/stop/restart implementations
   - Add action status endpoints
   - Implement action queuing

4. **Enhanced Monitoring** (v1.7.1)
   - Add distributed tracing (Jaeger/Tempo)
   - Implement error tracking (Sentry)
   - Add performance profiling

## Support

For issues during integration:
- Check logs: `kubectl logs -f deployment/agis-bot`
- Review database: `psql -U agis_user -d agis_bot`
- Test locally first before production deployment
- Refer to REST_API_v1.7.0.md for API details

---

**Status:** Ready for integration  
**Version:** v1.7.0  
**Date:** 2025-11-10

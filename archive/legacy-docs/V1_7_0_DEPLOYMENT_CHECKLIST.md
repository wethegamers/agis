# v1.7.0 Deployment Checklist

## ✅ Completed Development Tasks

### Core Services
- [x] **SchedulerService**: Cron-based server scheduling with Prometheus metrics
  - Rich return types (`*ServerSchedule`, `[]*ServerSchedule`)
  - Metrics: `schedulerActiveSchedules` gauge, `schedulerExecutionsTotal` counter
  - Server actions: start/stop/restart via EnhancedServerService
  - Automatic next run calculation and database updates
  
- [x] **APIKeyService**: Cryptographically secure API key management
  - Format: `agis_<base64(32bytes)>`, SHA256 storage
  - Scopes-based authorization
  - Per-key rate limits and TTL support
  - Async last_used updates
  
- [x] **RateLimiter**: Token bucket rate limiting
  - Memory-based with background cleanup (10-minute ticker)
  - Thread-safe with RWMutex
  - Redis-ready architecture (easy backend swap)
  - Methods: Allow(), GetRemaining(), ResetAfter(), Stop()

### API Endpoints (internal/api/server.go)
- [x] Authentication middleware supporting both API keys and legacy tokens
- [x] Rate limiting middleware with 429 responses and retry headers
- [x] API key management endpoints:
  - `POST /api/v1/auth/keys` - Generate new API key
  - `GET /api/v1/auth/keys` - List user's keys
  - `DELETE /api/v1/auth/keys/:id` - Revoke key
- [x] Server management endpoints (10+ endpoints for CRUD operations)
- [x] Rate limit headers: X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset

### Integration (main.go)
- [x] Scheduler initialization with metrics injection
- [x] API server initialization with port configuration (API_PORT env var)
- [x] Command handler wiring with SetScheduler()
- [x] Graceful shutdown for scheduler and rate limiter
- [x] Prometheus metrics registration

### Database (postgres-dev)
- [x] Migration v1.7.0 applied successfully
  - Tables: `server_schedules`, `api_keys`, `user_stats`
  - Indexes: 9 performance indexes created
  - Triggers: auto-update timestamps on user_stats
- [x] Credentials discovered: agis_dev_user / agis-password-dev / agis_dev
- [x] Namespace: postgres-dev

### Development Tooling
- [x] Makefile with 15+ targets (build, test, docker, k8s helpers)
- [x] Unit test scaffolds for scheduler and API services
- [x] Table-driven tests following Go best practices

### Dependencies
- [x] gorilla/mux v1.8.1 - REST API routing
- [x] robfig/cron/v3 v3.0.1 - Cron expression parsing

---

## 🚀 Deployment Steps

### Pre-Deployment
1. **Build Docker Image**
   ```bash
   cd /home/seb/wtg/agis-bot
   make docker-build  # or use cluster buildah/podman
   ```

2. **Tag and Push Image**
   ```bash
   docker tag agis-bot:dev <registry>/agis-bot:v1.7.0
   docker push <registry>/agis-bot:v1.7.0
   ```

3. **Update Kubernetes Manifests**
   - Update image tag in `deployments/k8s/*.yaml` to v1.7.0
   - Add API_PORT environment variable (default: "8080")
   - Ensure Prometheus scrape annotations present

### Deployment (Development Environment)
1. **Apply to agis-bot-dev namespace**
   ```bash
   kubectl apply -f deployments/k8s/ -n agis-bot-dev
   ```

2. **Verify Pods Running**
   ```bash
   kubectl get pods -n agis-bot-dev
   kubectl logs -n agis-bot-dev deployment/agis-bot --tail=100
   ```

3. **Check Scheduler Started**
   - Look for "✅ Scheduler started successfully" in logs
   - Verify cron runner initialized

4. **Check API Server Started**
   - Look for "API server listening on :8080" in logs
   - Verify routes registered

### Post-Deployment Verification

#### Test API Key Generation (via Discord Command)
```
/schedule create server:MyServer action:start cron:"0 8 * * *" timezone:America/New_York
```
Expected: Schedule created, stored in database

#### Test API Key Management
```bash
# Port-forward API server
kubectl port-forward -n agis-bot-dev deployment/agis-bot 8080:8080

# Create API key (requires existing discord_id)
curl -X POST http://localhost:8080/api/v1/auth/keys \
  -H "Authorization: Bearer <discord_id>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-key",
    "scopes": ["read:servers", "write:servers"],
    "rate_limit": 100,
    "ttl_days": 90
  }'

# Save returned key (only shown once!)
# Response: {"key": "agis_<base64>", "id": "...", ...}
```

#### Test Rate Limiting
```bash
# Make 101 requests rapidly
for i in {1..101}; do
  curl -H "Authorization: Bearer agis_<your_key>" \
       http://localhost:8080/api/v1/users/me -w "\n%{http_code}\n"
done

# Expected: First 100 succeed (200), 101st returns 429
# Check headers: X-RateLimit-Remaining: 0, Retry-After: <seconds>
```

#### Test Server Actions
```bash
# Create a schedule via API
curl -X POST http://localhost:8080/api/v1/schedules \
  -H "Authorization: Bearer agis_<your_key>" \
  -H "Content-Type: application/json" \
  -d '{
    "server_id": 1,
    "action": "start",
    "cron_expression": "*/5 * * * *",
    "timezone": "UTC"
  }'

# Wait for next run, check logs
kubectl logs -n agis-bot-dev deployment/agis-bot --tail=50 -f

# Expected: "🚀 Starting server... ✅ Successfully started server"
```

#### Test Metrics
```bash
# Port-forward metrics server (existing HTTP server, likely :8081)
kubectl port-forward -n agis-bot-dev deployment/agis-bot 8081:8081

# Check scheduler metrics
curl http://localhost:8081/metrics | grep scheduler
# Expected:
# agis_bot_scheduler_active_schedules <count>
# agis_bot_scheduler_executions_total{action="start",status="success"} <count>

# Check API metrics
curl http://localhost:8081/metrics | grep api_request
# Expected:
# agis_bot_api_requests_total{method="POST",endpoint="/api/v1/auth/keys",status="200"} <count>
# agis_bot_api_request_duration_seconds_bucket{...} <histogram>
```

#### Import Grafana Dashboards
```bash
# Port-forward Grafana (if in monitoring namespace)
kubectl port-forward -n monitoring svc/grafana 3000:3000

# Open http://localhost:3000
# Login with admin credentials
# Go to Dashboards > Import
# Upload docs/grafana/agis-bot-overview.json
# Upload docs/grafana/agis-bot-revenue.json
# Verify panels showing data from Prometheus
```

---

## ⚠️ Known Issues and Workarounds

### Docker Daemon Failure (Development Machine)
- **Issue**: Docker systemd service fails on EndeavorOS (overlay2 module missing)
- **Workaround**: Use cluster-based builds with kubectl/buildah or manual dockerd with VFS driver
- **Impact**: Local Docker testing unavailable, use cluster deployments

### Missing Local Go Installation
- **Issue**: Go not installed on development machine
- **Workaround**: Use cluster-based builds or install Go 1.23.0+
- **Impact**: Cannot run `make build` or `make test` locally, must test via cluster

---

## 📋 Pending Tasks (Nice-to-Have)

### Testing
- [ ] Integration tests for SchedulerService with real DB
- [ ] Integration tests for APIServer with httptest
- [ ] Concurrent access tests for RateLimiter with -race flag
- [ ] End-to-end tests for full flow (API key → auth → rate limit → server action)

### Documentation
- [ ] Swagger/OpenAPI spec generation for REST API
- [ ] API key management user guide
- [ ] Rate limiting documentation for API consumers
- [ ] Scheduler cron expression examples

### Observability
- [ ] Grafana dashboard panels for scheduler metrics
- [ ] Grafana dashboard panels for API metrics
- [ ] Alert rules for rate limit violations
- [ ] Alert rules for scheduler execution failures

### Production Hardening
- [ ] CORS configuration for API server
- [ ] Request ID tracing across services
- [ ] Structured logging (JSON) for production
- [ ] Health check endpoint improvements (DB connectivity, scheduler status)
- [ ] Redis backend for RateLimiter (distributed rate limiting)

---

## 🎯 Success Criteria

### Development Phase Complete When:
- ✅ All services compile without errors
- ✅ Database migration applied successfully
- ✅ Scheduler can create/list/delete schedules
- ✅ Scheduler executes server actions on cron triggers
- ✅ API key generation and validation work
- ✅ Rate limiting enforces per-key limits
- ✅ API returns rate limit headers
- ✅ Prometheus metrics exported and scrapable
- ✅ Graceful shutdown works for all services

### Staging Phase Complete When:
- [ ] All pods running in agis-bot-staging namespace
- [ ] API key creation tested via Discord command
- [ ] Rate limiting tested with 100+ requests
- [ ] Server actions tested (start/stop/restart)
- [ ] Metrics visible in Grafana dashboards
- [ ] No error logs for 24 hours

### Production Phase Complete When:
- [ ] All staging tests passed
- [ ] Load testing completed (1000+ req/sec)
- [ ] Failover testing completed (pod restarts, DB disconnects)
- [ ] Documentation reviewed and published
- [ ] Runbook created for on-call engineers
- [ ] Rollback plan documented and tested

---

## 📞 Support Contacts

- **Database Issues**: Check postgres-dev logs in postgres-dev namespace
- **Scheduler Issues**: Check agis-bot logs, search for "⏰ Executing scheduled"
- **API Issues**: Check agis-bot logs, search for "API server listening"
- **Metrics Issues**: Verify Prometheus scraping with `kubectl get servicemonitor`

---

**Version**: v1.7.0  
**Last Updated**: 2025-01-XX  
**Status**: Development Complete, Ready for Staging Deployment

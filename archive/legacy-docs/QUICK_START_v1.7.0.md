# Quick Start Guide - v1.7.0 Development Phase

## Prerequisites Check

Run this script to verify your environment:

```bash
#!/bin/bash
echo "=== Checking Prerequisites ==="

# Check Go installation
if command -v go &> /dev/null; then
    echo "✅ Go installed: $(go version)"
else
    echo "❌ Go not found - Install Go 1.23+ from https://go.dev/dl/"
    exit 1
fi

# Check PostgreSQL access
if command -v psql &> /dev/null; then
    echo "✅ PostgreSQL client installed"
else
    echo "⚠️  psql not found - Install postgresql-client"
fi

# Check kubectl
if command -v kubectl &> /dev/null; then
    echo "✅ kubectl installed"
    kubectl cluster-info &> /dev/null && echo "   Connected to cluster" || echo "   ⚠️  Not connected to cluster"
else
    echo "⚠️  kubectl not found"
fi

# Check required environment variables
echo -e "\\n=== Environment Variables ==="
for var in DISCORD_TOKEN DB_HOST DB_USER DB_PASSWORD DB_NAME; do
    if [ -n "${!var}" ]; then
        echo "✅ $var is set"
    else
        echo "❌ $var is missing"
    fi
done

echo -e "\\n=== Optional Variables ==="
for var in API_PORT SENTRY_DSN STRIPE_SECRET_KEY; do
    if [ -n "${!var}" ]; then
        echo "✅ $var is set"
    else
        echo "⚠️  $var not set (optional)"
    fi
done
```

Save as `scripts/check-prerequisites.sh` and run:
```bash
chmod +x scripts/check-prerequisites.sh
./scripts/check-prerequisites.sh
```

---

## Step 1: Install Dependencies

```bash
cd /home/seb/wtg/agis-bot

# Add required Go modules
go get github.com/gorilla/mux@latest
go get github.com/robfig/cron/v3@latest
go mod tidy

# Verify dependencies
go mod verify
```

---

## Step 2: Configure Environment

Copy `.env.example` to `.env` and fill in:

```bash
# PostgreSQL Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=agis_user
DB_PASSWORD=your_secure_password
DB_NAME=agis_bot
DB_SSLMODE=disable  # Use 'require' in production

# REST API Server
API_PORT=8080
API_HOST=0.0.0.0

# Discord (already configured)
DISCORD_TOKEN=your_token_here
DISCORD_GUILD_ID=your_guild_id

# Optional: API Authentication
API_ENABLE_AUTH=true  # Set false for local testing
API_BEARER_AUTH=true  # Simple bearer token (v1.7.0)
```

---

## Step 3: Run Database Migration

### Option A: Direct psql
```bash
# Load environment
source .env

# Run migration
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -f deployments/migrations/v1.7.0-rest-api-scheduling.sql

# Verify tables created
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "\\dt" | grep -E "server_schedules|api_keys|user_stats"
```

### Option B: Via kubectl (if DB in cluster)
```bash
# Find postgres pod
kubectl get pods -n database  # Or your namespace

# Execute migration
kubectl exec -it <postgres-pod> -n database -- \
  psql -U $DB_USER -d $DB_NAME < deployments/migrations/v1.7.0-rest-api-scheduling.sql
```

### Option C: Automated via Go
```bash
# I can create a migration runner if you prefer
go run scripts/migrate.go up
```

---

## Step 4: Build and Test Locally

```bash
# Build the bot
go build -o agis-bot .

# Run with verbose logging
export LOG_LEVEL=debug
./agis-bot

# Should see:
# ✅ Database service initialized
# ✅ Scheduler service started
# 🚀 REST API server started on :8080
# ✅ Registered schedule command
```

---

## Step 5: Test REST API

```bash
# Get your Discord ID (from Discord, enable Developer Mode)
DISCORD_ID="123456789012345678"

# Test health check
curl http://localhost:8080/health

# Test authentication
curl -H "Authorization: Bearer $DISCORD_ID" \
  http://localhost:8080/api/v1/users/me

# Expected response:
# {
#   "success": true,
#   "data": {
#     "discord_id": "123...",
#     "credits": 1000,
#     "tier": "free",
#     ...
#   }
# }

# Test list servers
curl -H "Authorization: Bearer $DISCORD_ID" \
  http://localhost:8080/api/v1/servers

# Test shop endpoint
curl -H "Authorization: Bearer $DISCORD_ID" \
  http://localhost:8080/api/v1/shop
```

---

## Step 6: Test Scheduler via Discord

In your Discord server:

```
# Show help
!schedule help

# Create a schedule (start server at 8am daily)
!schedule start my-server 0 8 * * *

# List schedules
!schedule list my-server

# Disable schedule
!schedule disable 1

# Enable schedule
!schedule enable 1

# Delete schedule
!schedule delete 1
```

---

## Step 7: Import Grafana Dashboards

### Via Grafana UI
1. Open Grafana: `http://localhost:3000` or your Grafana URL
2. Login (default: admin/admin)
3. Navigate to **Dashboards** → **Import**
4. Click **Upload JSON file**
5. Select `deployments/grafana/agis-bot-overview.json`
6. Choose Prometheus datasource
7. Click **Import**
8. Repeat for `agis-bot-revenue.json`

### Via API (Automated)
```bash
# Set Grafana credentials
GRAFANA_URL="http://localhost:3000"
GRAFANA_API_KEY="your_api_key"  # Or user:pass

# Import overview dashboard
curl -X POST $GRAFANA_URL/api/dashboards/db \
  -H "Authorization: Bearer $GRAFANA_API_KEY" \
  -H "Content-Type: application/json" \
  -d @deployments/grafana/agis-bot-overview.json

# Import revenue dashboard
curl -X POST $GRAFANA_URL/api/dashboards/db \
  -H "Authorization: Bearer $GRAFANA_API_KEY" \
  -H "Content-Type: application/json" \
  -d @deployments/grafana/agis-bot-revenue.json
```

---

## Step 8: Verify Metrics

Check Prometheus is scraping metrics:

```bash
# Check if metrics endpoint is accessible
curl http://localhost:8081/metrics | grep agis_

# Should see:
# agis_bot_active_users
# agis_bot_servers_active
# agis_bot_commands_total
# agis_ad_conversions_total
# ...

# Check Prometheus UI
# Open http://localhost:9090 (or your Prometheus URL)
# Query: agis_bot_commands_total
# Should show data
```

---

## Troubleshooting

### Issue: "Failed to start scheduler"
**Cause:** Database migration not run or `server_schedules` table missing
**Fix:**
```bash
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "\\d server_schedules"
# If not found, run migration from Step 3
```

### Issue: "REST API 404 on all endpoints"
**Cause:** API routes not registered properly
**Fix:** Check logs for "REST API server started on :8080"
```bash
# Test base path
curl http://localhost:8080/api/v1/
```

### Issue: "401 Unauthorized" on API calls
**Cause:** Missing or invalid Authorization header
**Fix:**
```bash
# Ensure Bearer token format
curl -H "Authorization: Bearer <your_discord_id>" \
  http://localhost:8080/api/v1/users/me
```

### Issue: "Schedule command not found"
**Cause:** Command not registered in main.go
**Fix:** Verify integration steps completed (see INTEGRATION_GUIDE_v1.7.0.md)

### Issue: Grafana dashboards show "No data"
**Cause:** Prometheus not scraping or metrics not exposed
**Fix:**
1. Check Prometheus targets: `http://localhost:9090/targets`
2. Verify bot is exposing metrics: `curl http://localhost:8081/metrics`
3. Check datasource in Grafana dashboard settings

---

## Development Workflow

### Making Changes
```bash
# 1. Make code changes
vim internal/api/server.go

# 2. Run tests
go test ./internal/api/...

# 3. Build
go build -o agis-bot .

# 4. Run locally
./agis-bot

# 5. Test changes
curl http://localhost:8080/api/v1/...
```

### Docker Development
```bash
# Build Docker image
docker build -t agis-bot:v1.7.0 .

# Run with environment
docker run --rm \
  --env-file .env \
  -p 8080:8080 \
  -p 8081:8081 \
  agis-bot:v1.7.0
```

### Kubernetes Deployment
```bash
# Create ConfigMap with new env vars
kubectl create configmap agis-config \
  --from-env-file=.env \
  -n default

# Apply deployment
kubectl apply -f deployments/k8s/agis-bot-deployment.yaml

# Check logs
kubectl logs -f deployment/agis-bot -n default

# Test API from cluster
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://agis-bot-service:8080/api/v1/shop
```

---

## Next Steps After Development Phase

1. **Write Integration Tests**
   - API endpoint tests
   - Scheduler execution tests
   - Database transaction tests

2. **Performance Testing**
   - Load test API with k6 or Apache Bench
   - Test scheduler with 1000+ schedules
   - Profile memory usage

3. **Security Hardening**
   - Implement proper API keys (v1.8.0)
   - Add rate limiting with Redis
   - Enable HTTPS/TLS
   - Add CORS configuration

4. **Documentation**
   - Generate Swagger/OpenAPI spec
   - Create API client examples (Python, JS)
   - Write admin runbook

5. **Monitoring & Alerts**
   - Set up Grafana alerts for:
     - API error rate > 5%
     - Scheduler failures > 10/hour
     - Database connection pool exhaustion
   - Configure PagerDuty/Slack notifications

---

## Questions or Issues?

- **Documentation:** See `docs/REST_API_v1.7.0.md` and `docs/INTEGRATION_GUIDE_v1.7.0.md`
- **Discord:** #agis-bot-dev channel
- **GitHub Issues:** https://github.com/wethegamers/agis-bot/issues

---

**Status:** Ready for development testing  
**Version:** v1.7.0  
**Last Updated:** 2025-11-10

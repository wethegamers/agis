# Integration Tests - ayeT-Studios

End-to-end integration tests for ayeT-Studios ad conversion system.

## Overview

Integration tests validate the complete flow:
1. ayeT sandbox triggers conversion
2. S2S callback sent to AGIS Bot
3. Signature verification
4. Database persistence
5. Metrics export
6. Fraud detection

## Prerequisites

### 1. ayeT Sandbox Account
- Sign up at https://sandbox-dashboard.ayet-studios.com
- Create test app
- Get sandbox API key
- Configure S2S callback URL

### 2. Environment Variables

```bash
# ayeT Sandbox Credentials
export AYET_API_KEY_SANDBOX="your_sandbox_api_key"
export AYET_CALLBACK_TOKEN_SANDBOX="your_callback_token"

# AGIS Bot Endpoints (local or staging)
export AGIS_BOT_CALLBACK_URL="http://localhost:9090/ads/ayet/s2s"
export AGIS_BOT_METRICS_URL="http://localhost:9090/metrics"

# Database (optional for verification)
export DB_HOST="localhost"
export DB_NAME="agis_test"
export DB_USER="root"
export DB_PASSWORD="password"
```

### 3. Running AGIS Bot Locally

```bash
# Terminal 1: Start AGIS Bot
DISCORD_TOKEN={{DISCORD_TOKEN}} \
DB_HOST=localhost \
METRICS_PORT=9090 \
AYET_API_KEY=${AYET_API_KEY_SANDBOX} \
AYET_CALLBACK_TOKEN=${AYET_CALLBACK_TOKEN_SANDBOX} \
go run ./cmd

# Terminal 2: Run integration tests
go test -tags=integration -v ./internal/services
```

## Running Tests

### Run All Integration Tests

```bash
go test -tags=integration -v ./internal/services
```

### Run Specific Test

```bash
go test -tags=integration -run TestAyetOfferwallCallback -v ./internal/services
```

### Skip Integration Tests (Default)

```bash
# Integration tests are skipped by default
go test ./internal/services

# Or explicitly
go test -short ./internal/services
```

## Test Suite

### 1. TestAyetSandboxConnection
**Purpose**: Verify connectivity to ayeT sandbox API  
**Requirements**: `AYET_API_KEY_SANDBOX`  
**What it tests**:
- Sandbox API reachable
- Authentication working
- Health endpoint responds

### 2. TestAyetOfferwallCallback
**Purpose**: End-to-end offerwall conversion flow  
**Requirements**: `AYET_API_KEY_SANDBOX`, `AGIS_BOT_CALLBACK_URL`  
**What it tests**:
- ayeT sandbox simulates conversion
- S2S callback sent to AGIS Bot
- Signature verification passes
- Conversion recorded in database
- User credited with Game Credits

**Flow**:
```
User completes offer → ayeT sandbox → S2S callback → AGIS Bot → Database
```

### 3. TestAyetSurveywallCallback
**Purpose**: Surveywall conversion flow  
**Currency**: `points` (converted to GC)  
**What it tests**:
- Surveywall-specific flow
- Multi-currency support
- Type detection (`custom_1=surveywall`)

### 4. TestAyetRewardedVideoCallback
**Purpose**: Rewarded video conversion flow  
**Currency**: `coins` (lower payout: 50 coins)  
**What it tests**:
- Video ad completion
- Lower reward amounts
- Type detection (`custom_1=video`)

### 5. TestAyetInvalidSignature
**Purpose**: Signature verification failure  
**What it tests**:
- Invalid signature rejected (401/403)
- No credits awarded
- Fraud attempt logged

**Expected**: HTTP 401 or 403, error in Sentry

### 6. TestAyetDuplicateConversion
**Purpose**: Idempotency via `conversion_id`  
**What it tests**:
- First request succeeds
- Second request (same `conversion_id`) rejected
- No double-crediting
- Database constraint enforced

**Expected**: First request 200 OK, second request detects duplicate

### 7. TestAyetFraudDetection
**Purpose**: Velocity-based fraud detection  
**What it tests**:
- Send 11 conversions rapidly
- 11th request triggers fraud (threshold: 10/hour)
- Conversion marked as fraud
- No credits awarded

**Expected**: First 10 succeed, 11th rejected with fraud reason

### 8. TestAyetMetricsExport
**Purpose**: Prometheus metrics validation  
**Requirements**: `AGIS_BOT_METRICS_URL`  
**What it tests**:
- `/metrics` endpoint accessible
- All expected metrics present:
  - `agis_ad_conversions_total`
  - `agis_ad_rewards_total`
  - `agis_ad_fraud_attempts_total`
  - `agis_ad_callback_latency_seconds`
  - `agis_ad_conversions_by_tier_total`

## Running Against Staging

```bash
# Point to staging environment
export AGIS_BOT_CALLBACK_URL="https://staging.agis-bot.wtgservers.com/ads/ayet/s2s"
export AGIS_BOT_METRICS_URL="https://staging.agis-bot.wtgservers.com/metrics"

# Use staging database
export DB_HOST="staging-db.internal"
export DB_NAME="agis_staging"

go test -tags=integration -v ./internal/services
```

## Continuous Integration

### GitHub Actions

Add to `.github/workflows/integration-tests.yml`:

```yaml
name: Integration Tests

on:
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 */6 * * *'  # Every 6 hours

jobs:
  integration:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_DB: agis_test
          POSTGRES_USER: root
          POSTGRES_PASSWORD: password
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      
      - name: Start AGIS Bot
        env:
          DB_HOST: localhost
          DB_NAME: agis_test
          DB_USER: root
          DB_PASSWORD: password
          AYET_API_KEY: ${{ secrets.AYET_API_KEY_SANDBOX }}
          AYET_CALLBACK_TOKEN: ${{ secrets.AYET_CALLBACK_TOKEN_SANDBOX }}
          DISCORD_TOKEN: ${{ secrets.DISCORD_TOKEN_TEST }}
          METRICS_PORT: 9090
        run: |
          go build -o agis-bot ./cmd
          ./agis-bot &
          sleep 5
      
      - name: Run Integration Tests
        env:
          AYET_API_KEY_SANDBOX: ${{ secrets.AYET_API_KEY_SANDBOX }}
          AGIS_BOT_CALLBACK_URL: "http://localhost:9090/ads/ayet/s2s"
          AGIS_BOT_METRICS_URL: "http://localhost:9090/metrics"
          DB_HOST: localhost
        run: |
          go test -tags=integration -v ./internal/services
```

## Manual Testing

### 1. Trigger Test Conversion via Sandbox Dashboard

1. Log into https://sandbox-dashboard.ayet-studios.com
2. Navigate to **Test Tools** → **Simulate Conversion**
3. Fill in:
   - User ID: `999999999999999999` (test Discord ID)
   - Offer ID: `test-offer-123`
   - Payout: `500 coins`
4. Click **Simulate**
5. Check AGIS Bot logs for callback

### 2. Verify in Database

```sql
-- Check conversion recorded
SELECT * FROM ad_conversions 
WHERE discord_id = '999999999999999999' 
ORDER BY created_at DESC LIMIT 10;

-- Check user credited
SELECT game_credits FROM users 
WHERE discord_id = '999999999999999999';

-- Check fraud attempts
SELECT * FROM ad_conversions 
WHERE status = 'fraud' 
ORDER BY created_at DESC LIMIT 10;
```

### 3. Check Metrics

```bash
curl http://localhost:9090/metrics | grep agis_ad
```

Expected output:
```
agis_ad_conversions_total{provider="ayet",type="offerwall",status="completed"} 1
agis_ad_rewards_total{provider="ayet",type="offerwall"} 500
agis_ad_callback_latency_seconds_bucket{provider="ayet",status="completed",le="0.5"} 1
```

### 4. Check Sentry

- Navigate to Sentry project
- Filter by `event.tags.provider:ayet`
- Verify no errors for valid conversions
- Verify errors logged for invalid signatures

## Troubleshooting

### Test Fails: "signature verification failed"

**Cause**: API key mismatch between test and server  
**Fix**:
```bash
# Ensure both use same key
echo $AYET_API_KEY_SANDBOX
echo $AYET_API_KEY
```

### Test Fails: "connection refused"

**Cause**: AGIS Bot not running  
**Fix**:
```bash
# Start AGIS Bot first
go run ./cmd &
# Wait for startup
sleep 5
# Run tests
go test -tags=integration ./internal/services
```

### Test Fails: "sandbox API unreachable"

**Cause**: Network issue or sandbox down  
**Fix**:
```bash
# Check sandbox status
curl https://sandbox-api.ayet-studios.com/health
# Skip sandbox tests if down
go test -tags=integration -run "^((?!Sandbox).)*$" ./internal/services
```

### Duplicate Detection Not Working

**Cause**: Database state or wrong table  
**Fix**:
```sql
-- Check for existing conversion
SELECT * FROM ad_conversions WHERE conversion_id = 'test-conv-123';
-- Clean up test data
DELETE FROM ad_conversions WHERE discord_id = '999999999999999999';
```

### Fraud Detection Not Triggering

**Cause**: Threshold not reached or database in local mode  
**Fix**:
```bash
# Ensure DB_HOST is set (not local mode)
export DB_HOST=localhost
# Run fraud test specifically
go test -tags=integration -run TestAyetFraudDetection -v ./internal/services
```

## Test Coverage

```bash
# Run with coverage
go test -tags=integration -coverprofile=coverage.out ./internal/services

# View coverage report
go tool cover -html=coverage.out
```

Target: >80% coverage for ad conversion paths

## Next Steps

1. Add database verification to tests (check credit amounts)
2. Implement cleanup routine (delete test conversions after tests)
3. Add load testing (simulate 100+ concurrent callbacks)
4. Test multi-provider scenarios (multiple ad networks)
5. Add E2E Discord bot command tests (`/earn` → open dashboard → callback)

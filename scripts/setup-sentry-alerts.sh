#!/bin/bash
set -e

# AGIS Bot v2.0 - Sentry Alert Setup Script
# This script creates alert rules in Sentry and configures Discord webhook routing

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SENTRY_ORG="${SENTRY_ORG:-}"
SENTRY_PROJECT="${SENTRY_PROJECT:-agis-bot}"
SENTRY_AUTH_TOKEN="${SENTRY_AUTH_TOKEN:-}"
SENTRY_API="${SENTRY_API:-https://sentry.io/api/0}"
ENVIRONMENT="${ENVIRONMENT:-development}"

# Discord webhooks (from Vault)
DISCORD_WEBHOOK_PAYMENTS="${DISCORD_WEBHOOK_PAYMENTS:-}"
DISCORD_WEBHOOK_ADS="${DISCORD_WEBHOOK_ADS:-}"
DISCORD_WEBHOOK_INFRA="${DISCORD_WEBHOOK_INFRA:-}"
DISCORD_WEBHOOK_SECURITY="${DISCORD_WEBHOOK_SECURITY:-}"
DISCORD_WEBHOOK_PERFORMANCE="${DISCORD_WEBHOOK_PERFORMANCE:-}"
DISCORD_WEBHOOK_REVENUE="${DISCORD_WEBHOOK_REVENUE:-}"
DISCORD_WEBHOOK_CRITICAL="${DISCORD_WEBHOOK_CRITICAL:-}"
DISCORD_WEBHOOK_COMPLIANCE="${DISCORD_WEBHOOK_COMPLIANCE:-}"

echo -e "${BLUE}🔔 AGIS Bot v2.0 - Sentry Alert Setup${NC}"
echo "Environment: $ENVIRONMENT"
echo ""

# Validate inputs
if [ -z "$SENTRY_ORG" ]; then
  echo -e "${RED}❌ SENTRY_ORG not set${NC}"
  echo "Usage: SENTRY_ORG=your-org SENTRY_AUTH_TOKEN=token ./setup-sentry-alerts.sh"
  exit 1
fi

if [ -z "$SENTRY_AUTH_TOKEN" ]; then
  echo -e "${RED}❌ SENTRY_AUTH_TOKEN not set${NC}"
  echo "Get token from: https://sentry.io/settings/account/api/auth-tokens/"
  exit 1
fi

# Function to create alert rule
create_alert_rule() {
  local name=$1
  local query=$2
  local webhook=$3
  local threshold=${4:-1}
  local time_window=${5:-5}
  
  if [ -z "$webhook" ]; then
    echo -e "${YELLOW}⚠️  Skipping '$name' - webhook not configured${NC}"
    return 0
  fi
  
  echo -e "${BLUE}📝 Creating alert rule: $name${NC}"
  
  local response=$(curl -s -X POST \
    "${SENTRY_API}/projects/${SENTRY_ORG}/${SENTRY_PROJECT}/alert-rules/" \
    -H "Authorization: Bearer ${SENTRY_AUTH_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"${name}\",
      \"environment\": \"${ENVIRONMENT}\",
      \"dataset\": \"events\",
      \"query\": \"${query}\",
      \"aggregate\": \"count()\",
      \"timeWindow\": ${time_window},
      \"triggers\": [
        {
          \"label\": \"critical\",
          \"alertThreshold\": ${threshold},
          \"actions\": [
            {
              \"type\": \"discord\",
              \"targetIdentifier\": \"${webhook}\"
            }
          ]
        }
      ]
    }")
  
  # Check if successful
  if echo "$response" | grep -q '"id"'; then
    echo -e "${GREEN}✅ Created: $name${NC}"
  else
    echo -e "${RED}❌ Failed to create: $name${NC}"
    echo "Response: $response"
    return 1
  fi
}

echo -e "${YELLOW}📋 Creating alert rules...${NC}"
echo ""

# Alert Rule 1: Payment Processing Failures
create_alert_rule \
  "Payment Processing Failure - Critical" \
  "event.type:error tags.category:payment" \
  "$DISCORD_WEBHOOK_PAYMENTS" \
  1 \
  5

# Alert Rule 2: Ad Conversion Errors
create_alert_rule \
  "Ad Conversion Error - High" \
  "event.type:error tags.category:ad_conversion" \
  "$DISCORD_WEBHOOK_ADS" \
  5 \
  10

# Alert Rule 3: Database Connection Errors
create_alert_rule \
  "Database Connection Error - Critical" \
  "event.type:error tags.category:database" \
  "$DISCORD_WEBHOOK_INFRA" \
  1 \
  5

# Alert Rule 4: Authentication Failures
create_alert_rule \
  "Authentication Failure - High" \
  "event.type:error tags.category:auth" \
  "$DISCORD_WEBHOOK_SECURITY" \
  3 \
  10

# Alert Rule 5: Performance Degradation
create_alert_rule \
  "Performance Degradation - Medium" \
  "event.type:transaction transaction.duration:>5000" \
  "$DISCORD_WEBHOOK_PERFORMANCE" \
  10 \
  15

# Alert Rule 6: Revenue Processing Errors
create_alert_rule \
  "Revenue Processing Error - Critical" \
  "event.type:error tags.category:revenue" \
  "$DISCORD_WEBHOOK_REVENUE" \
  1 \
  5

# Alert Rule 7: Critical Errors (Panics)
create_alert_rule \
  "Critical Error - Panic" \
  "event.type:error level:fatal" \
  "$DISCORD_WEBHOOK_CRITICAL" \
  1 \
  5

# Alert Rule 8: Compliance Issues
create_alert_rule \
  "Compliance Issue - Critical" \
  "event.type:error tags.category:compliance" \
  "$DISCORD_WEBHOOK_COMPLIANCE" \
  1 \
  5

echo ""
echo -e "${GREEN}✅ Alert rules setup complete!${NC}"
echo ""
echo -e "${BLUE}📊 Next steps:${NC}"
echo "1. Verify alert rules in Sentry: https://sentry.io/settings/${SENTRY_ORG}/${SENTRY_PROJECT}/alerts/"
echo "2. Test error capture: kubectl exec -n development agis-bot-xxx -- curl http://localhost:9090/api/test-error"
echo "3. Monitor Discord channels for alerts"
echo "4. Adjust thresholds if needed"
echo ""

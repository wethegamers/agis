# Week 2, Step 7: Guild Provisioning Testing

**Status**: Implementation Phase  
**Date**: 2025-11-10  
**Objective**: Verify guild provisioning and server management system  
**Timeline**: 2-3 hours

## Overview

The guild provisioning system allows guilds to provision game servers using their treasury balance. This step verifies all components work correctly.

## Current Status

### ✅ Database Schema Ready
- `server_provision_requests` - Provisioning requests
- `server_templates` - Pre-configured templates (5 templates)
- `guild_treasury` - Guild balance tracking
- `treasury_transactions` - Transaction audit log

### ✅ Command Handlers Ready
- `/guild-server templates` - List available templates
- `/guild-server create` - Create provisioning request
- `/guild-server list` - List active servers
- `/guild-server terminate` - Terminate server
- `/guild-server treasury` - View treasury balance
- `/guild-server info` - Get server info

### ✅ Features Implemented
- Server template selection
- Treasury balance tracking
- Automatic cost deduction
- Server lifecycle management
- Audit logging

## Step 7 Tasks

### Task 1: List Available Templates

**Objective**: Verify server templates are available

**Steps**:
1. Run command: `/guild-server templates`
2. Verify all 5 templates are listed:
   - Minecraft (Small, Medium, Large)
   - Valheim (Small)
   - Palworld (Small)

**Expected Output**:
```
📋 Available Server Templates:

1. Minecraft (Small)
   - Game: minecraft
   - Max Players: 10
   - Cost: 100 GC/hour
   - Setup: 500 GC

2. Minecraft (Medium)
   - Game: minecraft
   - Max Players: 25
   - Cost: 200 GC/hour
   - Setup: 1000 GC

3. Minecraft (Large)
   - Game: minecraft
   - Max Players: 50
   - Cost: 400 GC/hour
   - Setup: 2000 GC

4. Valheim (Small)
   - Game: valheim
   - Max Players: 10
   - Cost: 150 GC/hour
   - Setup: 750 GC

5. Palworld (Small)
   - Game: palworld
   - Max Players: 10
   - Cost: 200 GC/hour
   - Setup: 1000 GC
```

**Verification**:
```bash
# Check templates in database
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT id, name, game_type, cost_per_hour, setup_cost FROM server_templates ORDER BY id;"
```

### Task 2: Check Guild Treasury

**Objective**: Verify guild treasury balance

**Steps**:
1. Run command: `/guild-server treasury`
2. Parameters:
   - Guild ID: (your test guild)
3. Verify balance is displayed

**Expected Output**:
```
💰 Guild Treasury: test-guild-001

Balance: 10,000 GC
Last Updated: 2025-11-10 23:00:00 UTC

Recent Transactions:
- 2025-11-10 22:50:00: +500 GC (Ad conversion)
- 2025-11-10 22:45:00: -1000 GC (Server provisioning)
- 2025-11-10 22:40:00: +1000 GC (Reward)
```

**Verification**:
```bash
# Check treasury balance
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT guild_id, balance, last_updated FROM guild_treasury WHERE guild_id = 'test-guild-001';"

# Check transactions
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT * FROM treasury_transactions WHERE guild_id = 'test-guild-001' ORDER BY created_at DESC LIMIT 10;"
```

### Task 3: Create Provisioning Request

**Objective**: Create a server provisioning request

**Steps**:
1. Run command: `/guild-server create`
2. Parameters:
   - Guild ID: test-guild-001
   - Template: minecraft-small
   - Server Name: "Test Server"
   - Duration: 1 hour
   - Auto-renew: false
3. Verify request created successfully

**Expected Output**:
```
✅ Provisioning request created: prov-req-001

Details:
- Guild: test-guild-001
- Template: Minecraft (Small)
- Server Name: Test Server
- Duration: 1 hour
- Cost: 100 GC/hour + 500 GC setup = 600 GC total
- Status: pending
- Created: 2025-11-10 23:05:00 UTC

⏳ Waiting for approval...
```

**Verification**:
```bash
# Check provisioning request
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT id, guild_id, template_id, status, created_at FROM server_provision_requests WHERE guild_id = 'test-guild-001' ORDER BY created_at DESC LIMIT 1;"

# Check treasury was debited
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT balance FROM guild_treasury WHERE guild_id = 'test-guild-001';"
```

### Task 4: List Active Servers

**Objective**: Verify server listing works

**Steps**:
1. Run command: `/guild-server list`
2. Parameters:
   - Guild ID: test-guild-001
3. Verify active servers are listed

**Expected Output**:
```
🖥️  Active Servers: test-guild-001

1. Test Server (prov-req-001)
   - Template: Minecraft (Small)
   - Status: active
   - Players: 0/10
   - Uptime: 5 minutes
   - Cost: 100 GC/hour
   - Expires: 2025-11-10 23:05:00 UTC
```

**Verification**:
```bash
# Check active servers
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT id, server_name, status, created_at FROM server_provision_requests WHERE guild_id = 'test-guild-001' AND status = 'active';"
```

### Task 5: Get Server Info

**Objective**: Verify server information retrieval

**Steps**:
1. Run command: `/guild-server info`
2. Parameters:
   - Server ID: prov-req-001
3. Verify detailed server info is displayed

**Expected Output**:
```
ℹ️  Server Information: prov-req-001

Guild: test-guild-001
Template: Minecraft (Small)
Server Name: Test Server
Status: active
Created: 2025-11-10 23:05:00 UTC
Expires: 2025-11-10 23:05:00 UTC

Resources:
- CPU: 1000m
- Memory: 2Gi
- Max Players: 10

Costs:
- Hourly: 100 GC
- Setup: 500 GC
- Total Spent: 600 GC

Auto-Renew: disabled
```

**Verification**:
```bash
# Check server details
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT * FROM server_provision_requests WHERE id = 'prov-req-001';"
```

### Task 6: Test Subscription Tier Validation

**Objective**: Verify subscription tier limits are enforced

**Steps**:
1. Check guild subscription tier
2. Attempt to provision server beyond tier limit
3. Verify error message

**Expected Output** (if tier limit exceeded):
```
❌ Cannot provision server

Reason: Subscription tier limit exceeded
- Current Tier: Free
- Max Servers: 1
- Active Servers: 1

Upgrade to Premium for unlimited servers.
```

**Verification**:
```bash
# Check subscription tier
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT guild_id, tier, max_servers FROM subscriptions WHERE guild_id = 'test-guild-001';"
```

### Task 7: Terminate Server

**Objective**: Verify server termination

**Steps**:
1. Run command: `/guild-server terminate`
2. Parameters:
   - Server ID: prov-req-001
3. Verify server status changed to "terminated"

**Expected Output**:
```
✅ Server terminated: prov-req-001

Details:
- Server: Test Server
- Status: terminated
- Uptime: 10 minutes
- Cost: 100 GC (10 minutes usage)
- Refund: 0 GC (no refund for early termination)

Treasury Updated: 9,900 GC → 9,900 GC
```

**Verification**:
```bash
# Check server status
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT id, status, terminated_at FROM server_provision_requests WHERE id = 'prov-req-001';"

# Check final treasury balance
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT balance FROM guild_treasury WHERE guild_id = 'test-guild-001';"
```

### Task 8: Verify Audit Logging

**Objective**: Verify all actions are logged

**Steps**:
1. Review transaction history
2. Verify all provisioning actions are logged
3. Check timestamps and amounts

**Expected Output**:
```
📋 Audit Log: test-guild-001

2025-11-10 23:15:00: Server terminated (prov-req-001)
2025-11-10 23:05:00: Server provisioned (prov-req-001)
2025-11-10 23:00:00: Treasury balance checked
```

**Verification**:
```bash
# Check audit log
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT * FROM treasury_transactions WHERE guild_id = 'test-guild-001' ORDER BY created_at DESC LIMIT 20;"
```

## Success Criteria

- [x] Database schema verified
- [ ] Templates listed successfully
- [ ] Treasury balance displayed correctly
- [ ] Provisioning request created successfully
- [ ] Active servers listed correctly
- [ ] Server info retrieved successfully
- [ ] Subscription tier limits enforced
- [ ] Server terminated successfully
- [ ] Audit logging working correctly

## Testing Checklist

### Database Checks
- [ ] `server_templates` table has 5 templates
- [ ] `guild_treasury` table has balance
- [ ] `treasury_transactions` table has entries
- [ ] `server_provision_requests` table has requests

### Command Checks
- [ ] `/guild-server templates` working
- [ ] `/guild-server treasury` working
- [ ] `/guild-server create` working
- [ ] `/guild-server list` working
- [ ] `/guild-server info` working
- [ ] `/guild-server terminate` working

### Data Integrity Checks
- [ ] Treasury balance accurate
- [ ] Transactions recorded correctly
- [ ] Costs calculated correctly
- [ ] Subscription limits enforced
- [ ] Audit log complete

## Troubleshooting

### Templates Not Listed
- Check database: `SELECT * FROM server_templates;`
- Verify bot has database access
- Check bot logs

### Treasury Balance Wrong
- Check transactions: `SELECT * FROM treasury_transactions;`
- Verify calculation logic
- Check for duplicate entries

### Provisioning Failed
- Check bot logs
- Verify treasury has sufficient balance
- Check subscription tier limits

### Agones Integration Issues
- Verify Agones is available
- Check Agones namespace
- Review Agones logs

## Performance Metrics

After testing, verify:
- Template listing < 100ms
- Treasury query < 50ms
- Provisioning request < 500ms
- Server termination < 1s

## Next Steps

After Step 7 completion:
1. Proceed to Step 8: Webhook Verification
2. Test Discord webhooks
3. Verify Sentry alerts
4. Test payment notifications

## Files Involved

### Database
- `deployments/migrations/v2.0-production-enhancements.sql`

### Code
- `internal/services/guild_provisioning.go`
- `internal/bot/commands/guild_server_command.go`

### Configuration
- Helm values for environment variables
- Vault secrets for API keys

## Resources

- [Guild Provisioning Guide](docs/GUILD_PROVISIONING_GUIDE.md)
- [Database Schema](deployments/migrations/v2.0-production-enhancements.sql)
- [Integration Tests](docs/INTEGRATION_TESTS.md)

---

**Step 7 Ready to Execute!** 🚀

# Week 2, Step 6: A/B Testing Verification

**Status**: Implementation Phase  
**Date**: 2025-11-10  
**Objective**: Verify A/B testing framework is working correctly  
**Timeline**: 2-3 hours

## Overview

The A/B testing framework has been fully implemented in the database and bot code. This step verifies all components work together correctly.

## Current Status

### ✅ Database Schema Ready
- `ab_experiments` - Experiment configurations
- `ab_variants` - Experiment variants
- `ab_assignments` - User-to-variant assignments
- `ab_events` - Event tracking
- `ab_experiment_results` - Analytics view

### ✅ Command Handlers Ready
- `/experiment create` - Create new experiment
- `/experiment start` - Start experiment
- `/experiment stop` - Stop experiment
- `/experiment results` - View results
- `/experiment list` - List experiments

### ✅ Features Implemented
- Sticky assignments (users stay in same variant)
- Event tracking
- Results analytics
- Experiment lifecycle management

## Step 6 Tasks

### Task 1: Create Test A/B Experiment

**Objective**: Create a test experiment via Discord command

**Steps**:
1. Connect to Discord server where bot is running
2. Run command: `/experiment create`
3. Fill in parameters:
   - Name: "Test Experiment"
   - Description: "Testing A/B framework"
   - Hypothesis: "Testing sticky assignments"
4. Verify experiment created successfully

**Expected Output**:
```
✅ Experiment created: test-experiment-001
ID: abc123def456
Status: draft
Created: 2025-11-10 22:45:00 UTC
```

**Verification**:
```bash
# Check database
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT id, name, status FROM ab_experiments ORDER BY created_at DESC LIMIT 1;"
```

### Task 2: Create Test Variants

**Objective**: Create variants for the experiment

**Steps**:
1. Run command: `/experiment create-variant`
2. Parameters:
   - Experiment ID: (from Task 1)
   - Variant Name: "Control"
   - Description: "Control group"
3. Repeat for "Treatment" variant

**Expected Output**:
```
✅ Variant created: control-v1
Experiment: test-experiment-001
Name: Control
Weight: 50%
```

**Verification**:
```bash
# Check variants
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT id, name, weight FROM ab_variants WHERE experiment_id = 'test-experiment-001';"
```

### Task 3: Start Experiment

**Objective**: Start the experiment

**Steps**:
1. Run command: `/experiment start`
2. Parameters:
   - Experiment ID: (from Task 1)
3. Verify experiment status changed to "active"

**Expected Output**:
```
✅ Experiment started: test-experiment-001
Status: active
Started: 2025-11-10 22:50:00 UTC
```

**Verification**:
```bash
# Check experiment status
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT id, status, started_at FROM ab_experiments WHERE id = 'test-experiment-001';"
```

### Task 4: Test Sticky Assignments

**Objective**: Verify users are assigned to variants and stay in same variant

**Steps**:
1. Simulate user assignment (via API or direct DB insert)
2. Assign multiple users to experiment
3. Verify assignments are sticky (same variant on repeat)

**Test Scenario**:
```bash
# Simulate user assignment
curl -X POST http://localhost:9090/api/ab/assign \
  -H "Content-Type: application/json" \
  -d '{
    "experiment_id": "test-experiment-001",
    "user_id": "user-123",
    "user_properties": {"country": "US"}
  }'

# Expected response
{
  "experiment_id": "test-experiment-001",
  "user_id": "user-123",
  "variant_id": "control-v1",
  "assigned_at": "2025-11-10T22:55:00Z"
}
```

**Verification**:
```bash
# Check assignments
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT user_id, variant_id, assigned_at FROM ab_assignments WHERE experiment_id = 'test-experiment-001';"

# Verify sticky (call again, should get same variant)
curl -X POST http://localhost:9090/api/ab/assign \
  -H "Content-Type: application/json" \
  -d '{
    "experiment_id": "test-experiment-001",
    "user_id": "user-123"
  }'
# Should return same variant_id as before
```

### Task 5: Track Events

**Objective**: Verify event tracking is working

**Steps**:
1. Simulate user events (conversions, clicks, etc.)
2. Track events for assigned users
3. Verify events are recorded

**Test Scenario**:
```bash
# Track event
curl -X POST http://localhost:9090/api/ab/event \
  -H "Content-Type: application/json" \
  -d '{
    "experiment_id": "test-experiment-001",
    "user_id": "user-123",
    "event_type": "conversion",
    "event_value": 100
  }'

# Expected response
{
  "event_id": "evt-abc123",
  "recorded_at": "2025-11-10T22:58:00Z"
}
```

**Verification**:
```bash
# Check events
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT event_type, event_value, COUNT(*) FROM ab_events WHERE experiment_id = 'test-experiment-001' GROUP BY event_type, event_value;"
```

### Task 6: View Results

**Objective**: Verify results view shows correct analytics

**Steps**:
1. Run command: `/experiment results`
2. Parameters:
   - Experiment ID: (from Task 1)
3. Verify results show:
   - Variant breakdown
   - Event counts
   - Conversion rates
   - Statistical significance (if applicable)

**Expected Output**:
```
📊 Experiment Results: test-experiment-001

Control Group:
  Users: 50
  Conversions: 25
  Conversion Rate: 50%
  
Treatment Group:
  Users: 50
  Conversions: 28
  Conversion Rate: 56%

Statistical Significance: Not significant (p > 0.05)
```

**Verification**:
```bash
# Check results view
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT * FROM ab_experiment_results WHERE experiment_id = 'test-experiment-001';"
```

### Task 7: Stop Experiment

**Objective**: Stop the experiment

**Steps**:
1. Run command: `/experiment stop`
2. Parameters:
   - Experiment ID: (from Task 1)
3. Verify experiment status changed to "stopped"

**Expected Output**:
```
✅ Experiment stopped: test-experiment-001
Status: stopped
Stopped: 2025-11-10 23:05:00 UTC
Final Results: Control 50%, Treatment 56%
```

**Verification**:
```bash
# Check experiment status
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT id, status, stopped_at FROM ab_experiments WHERE id = 'test-experiment-001';"
```

### Task 8: List Experiments

**Objective**: Verify experiment listing works

**Steps**:
1. Run command: `/experiment list`
2. Verify test experiment appears in list
3. Check status and other details

**Expected Output**:
```
📋 Active Experiments:
1. test-experiment-001 (stopped)
   - Variants: 2
   - Users: 100
   - Duration: 15 minutes
```

## Success Criteria

- [x] Database schema verified
- [ ] Test experiment created successfully
- [ ] Variants created successfully
- [ ] Experiment started successfully
- [ ] Sticky assignments working
- [ ] Events tracked correctly
- [ ] Results view showing correct data
- [ ] Experiment stopped successfully
- [ ] Experiment listing working

## Testing Checklist

### Database Checks
- [ ] `ab_experiments` table has test data
- [ ] `ab_variants` table has 2 variants
- [ ] `ab_assignments` table has assignments
- [ ] `ab_events` table has events
- [ ] `ab_experiment_results` view returns data

### API Checks
- [ ] `/api/ab/assign` endpoint working
- [ ] `/api/ab/event` endpoint working
- [ ] `/api/ab/results` endpoint working

### Discord Command Checks
- [ ] `/experiment create` working
- [ ] `/experiment start` working
- [ ] `/experiment stop` working
- [ ] `/experiment results` working
- [ ] `/experiment list` working

### Data Integrity Checks
- [ ] Sticky assignments verified (same user gets same variant)
- [ ] Event tracking accurate
- [ ] Results calculations correct
- [ ] No data loss or corruption

## Troubleshooting

### Experiment Not Created
- Check bot logs: `kubectl logs -n development agis-bot-xxx`
- Verify database connection
- Check Discord permissions

### Assignments Not Sticky
- Verify assignment logic in code
- Check database for duplicate assignments
- Review assignment algorithm

### Events Not Tracked
- Check event endpoint logs
- Verify event format
- Check database for events

### Results Not Showing
- Verify view exists: `\dv ab_experiment_results`
- Check view query
- Verify data in underlying tables

## Performance Metrics

After testing, verify:
- Assignment latency < 100ms
- Event tracking latency < 50ms
- Results query latency < 500ms
- No database locks or deadlocks

## Next Steps

After Step 6 completion:
1. Proceed to Step 7: Guild Provisioning Testing
2. Test server provisioning workflow
3. Verify treasury system
4. Test subscription tiers

## Files Involved

### Database
- `deployments/migrations/v2.0-production-enhancements.sql`

### Code
- `internal/services/ab_testing.go` (if exists)
- `internal/bot/commands/experiment_command.go`

### Configuration
- Helm values for environment variables
- Vault secrets for API keys

## Resources

- [A/B Testing Guide](docs/AB_TESTING_GUIDE.md)
- [Database Schema](deployments/migrations/v2.0-production-enhancements.sql)
- [Integration Tests](docs/INTEGRATION_TESTS.md)

---

**Step 6 Ready to Execute!** 🚀

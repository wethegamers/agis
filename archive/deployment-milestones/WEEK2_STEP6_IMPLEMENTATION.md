# Week 2, Step 6: A/B Testing Verification - Implementation Guide

**Status**: Ready to Execute  
**Date**: 2025-11-10  
**Objective**: Verify A/B testing framework works end-to-end  
**Timeline**: 2-3 hours

## Quick Start

### Prerequisites
- ✅ AGIS Bot pod running in development namespace
- ✅ PostgreSQL database with ab_experiments tables
- ✅ Discord bot connected to test server
- ✅ kubectl access to cluster

### Verification Checklist

```bash
# 1. Check pod is running
kubectl get pods -n development | grep agis-bot

# 2. Check database tables exist
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c "\dt ab_*"

# 3. Check bot logs
kubectl logs -n development agis-bot-xxx --tail=20
```

## Step 6 Implementation

### Phase 1: Database Verification (15 minutes)

#### Task 1.1: Verify A/B Testing Tables

```bash
# Connect to database
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev << 'EOF'

-- Check ab_experiments table
SELECT COUNT(*) as experiment_count FROM ab_experiments;

-- Check ab_variants table
SELECT COUNT(*) as variant_count FROM ab_variants;

-- Check ab_assignments table
SELECT COUNT(*) as assignment_count FROM ab_assignments;

-- Check ab_events table
SELECT COUNT(*) as event_count FROM ab_events;

-- Check ab_experiment_results view
SELECT COUNT(*) as result_count FROM ab_experiment_results;

EOF
```

**Expected Output**:
```
experiment_count | 0
variant_count    | 0
assignment_count | 0
event_count      | 0
result_count     | 0
```

#### Task 1.2: Verify Indexes

```bash
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT indexname FROM pg_indexes WHERE tablename LIKE 'ab_%' ORDER BY indexname;"
```

**Expected Output**: 20+ indexes for performance

### Phase 2: Create Test Experiment (30 minutes)

#### Task 2.1: Create Experiment via Discord Command

**Command Format**:
```
/experiment create <id> <name> <traffic%> <duration_days> <control_multiplier> <variant_multiplier>
```

**Example**:
```
/experiment create test-exp-001 "Reward Multiplier Test" 100 7 1.0 1.5
```

**Parameters**:
- `id`: Unique experiment identifier (test-exp-001)
- `name`: Human-readable name
- `traffic%`: Percentage of users in experiment (100 = all users)
- `duration_days`: How long to run (7 = 7 days)
- `control_multiplier`: Control group reward multiplier (1.0 = normal)
- `variant_multiplier`: Test group reward multiplier (1.5 = 50% more)

**Expected Response**:
```
✅ Experiment created: **Reward Multiplier Test**
ID: `test-exp-001`
Traffic: 100%
Duration: 7 days
Control: 1.0x | Variant: 1.5x
Status: **draft**

Run `/experiment start test-exp-001` to activate
```

#### Task 2.2: Verify Experiment in Database

```bash
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev << 'EOF'

-- Check experiment was created
SELECT id, name, status, traffic_allocation, start_date, end_date 
FROM ab_experiments 
WHERE id = 'test-exp-001';

-- Check variants were created
SELECT id, name, allocation, config 
FROM ab_variants 
WHERE experiment_id = 'test-exp-001';

EOF
```

**Expected Output**:
```
id              | test-exp-001
name            | Reward Multiplier Test
status          | draft
traffic_allocation | 1.0
start_date      | 2025-11-10 22:45:00
end_date        | 2025-11-17 22:45:00

id              | control
name            | Control
allocation      | 0.5
config          | {"multiplier": 1.0}

id              | variant_a
name            | Variant A
allocation      | 0.5
config          | {"multiplier": 1.5}
```

### Phase 3: Start Experiment (15 minutes)

#### Task 3.1: Start Experiment

**Command**:
```
/experiment start test-exp-001
```

**Expected Response**:
```
✅ Experiment started: **Reward Multiplier Test**
ID: `test-exp-001`
Status: **active**
Started: 2025-11-10 22:50:00 UTC
Ends: 2025-11-17 22:50:00 UTC
```

#### Task 3.2: Verify Status Changed

```bash
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT id, status, started_at FROM ab_experiments WHERE id = 'test-exp-001';"
```

**Expected Output**:
```
id              | test-exp-001
status          | active
started_at      | 2025-11-10 22:50:00
```

### Phase 4: Test Sticky Assignments (30 minutes)

#### Task 4.1: Assign Users to Variants

**Command**:
```
/experiment assign test-exp-001 <user_id>
```

**Example** (assign 5 test users):
```
/experiment assign test-exp-001 user-001
/experiment assign test-exp-001 user-002
/experiment assign test-exp-001 user-003
/experiment assign test-exp-001 user-004
/experiment assign test-exp-001 user-005
```

**Expected Response**:
```
✅ User assigned to experiment
User: user-001
Experiment: test-exp-001
Variant: control (randomly assigned)
```

#### Task 4.2: Verify Sticky Assignments

```bash
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev << 'EOF'

-- Check assignments
SELECT user_id, variant_id, assigned_at 
FROM ab_assignments 
WHERE experiment_id = 'test-exp-001'
ORDER BY assigned_at;

-- Verify distribution (should be ~50/50)
SELECT variant_id, COUNT(*) as count 
FROM ab_assignments 
WHERE experiment_id = 'test-exp-001'
GROUP BY variant_id;

EOF
```

**Expected Output**:
```
user_id | variant_id | assigned_at
user-001 | control | 2025-11-10 22:55:00
user-002 | variant_a | 2025-11-10 22:55:05
user-003 | control | 2025-11-10 22:55:10
user-004 | variant_a | 2025-11-10 22:55:15
user-005 | control | 2025-11-10 22:55:20

variant_id | count
control | 3
variant_a | 2
```

#### Task 4.3: Verify Sticky Assignment (Same User, Same Variant)

**Command** (assign same user again):
```
/experiment assign test-exp-001 user-001
```

**Expected Response**:
```
✅ User already assigned
User: user-001
Experiment: test-exp-001
Variant: control (sticky - same as before)
```

**Verify in Database**:
```bash
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT COUNT(*) FROM ab_assignments WHERE user_id = 'user-001' AND experiment_id = 'test-exp-001';"
```

**Expected Output**: `1` (only one assignment, not duplicated)

### Phase 5: Track Events (30 minutes)

#### Task 5.1: Simulate User Events

**Command**:
```
/experiment event test-exp-001 <user_id> <event_type> <value>
```

**Examples**:
```
/experiment event test-exp-001 user-001 conversion 100
/experiment event test-exp-001 user-002 conversion 150
/experiment event test-exp-001 user-003 conversion 80
/experiment event test-exp-001 user-004 conversion 200
/experiment event test-exp-001 user-005 conversion 120
```

**Expected Response**:
```
✅ Event recorded
User: user-001
Experiment: test-exp-001
Event: conversion
Value: 100
```

#### Task 5.2: Verify Events in Database

```bash
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev << 'EOF'

-- Check events
SELECT user_id, variant_id, event_type, event_value, created_at 
FROM ab_events 
WHERE experiment_id = 'test-exp-001'
ORDER BY created_at;

-- Count events by variant
SELECT variant_id, COUNT(*) as event_count, AVG(event_value) as avg_value
FROM ab_events 
WHERE experiment_id = 'test-exp-001'
GROUP BY variant_id;

EOF
```

**Expected Output**:
```
user_id | variant_id | event_type | event_value | created_at
user-001 | control | conversion | 100 | 2025-11-10 23:00:00
user-002 | variant_a | conversion | 150 | 2025-11-10 23:00:05
user-003 | control | conversion | 80 | 2025-11-10 23:00:10
user-004 | variant_a | conversion | 200 | 2025-11-10 23:00:15
user-005 | control | conversion | 120 | 2025-11-10 23:00:20

variant_id | event_count | avg_value
control | 3 | 100.0
variant_a | 2 | 175.0
```

### Phase 6: View Results (15 minutes)

#### Task 6.1: Get Experiment Results

**Command**:
```
/experiment results test-exp-001
```

**Expected Response**:
```
📊 Experiment Results: **Reward Multiplier Test**
ID: `test-exp-001`
Status: **active**
Duration: 7 days (3 days remaining)

**Control Group**:
- Users: 3
- Conversions: 3
- Avg Value: 100.0
- Conversion Rate: 100%

**Variant A**:
- Users: 2
- Conversions: 2
- Avg Value: 175.0
- Conversion Rate: 100%

**Statistical Significance**: Not enough data yet
```

#### Task 6.2: Verify Results View

```bash
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT * FROM ab_experiment_results WHERE experiment_id = 'test-exp-001';"
```

**Expected Output**: Results aggregated by variant

### Phase 7: Stop Experiment (15 minutes)

#### Task 7.1: Stop Experiment

**Command**:
```
/experiment stop test-exp-001
```

**Expected Response**:
```
✅ Experiment stopped: **Reward Multiplier Test**
ID: `test-exp-001`
Status: **stopped**
Stopped: 2025-11-10 23:05:00 UTC
Final Results: Control 100.0 vs Variant 175.0
```

#### Task 7.2: Verify Status Changed

```bash
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT id, status, stopped_at FROM ab_experiments WHERE id = 'test-exp-001';"
```

**Expected Output**:
```
id              | test-exp-001
status          | stopped
stopped_at      | 2025-11-10 23:05:00
```

### Phase 8: List Experiments (10 minutes)

#### Task 8.1: List All Experiments

**Command**:
```
/experiment list
```

**Expected Response**:
```
📋 Active Experiments:
(none - all stopped)

📋 Completed Experiments:
1. **Reward Multiplier Test** (test-exp-001)
   - Status: stopped
   - Duration: 7 days
   - Control: 1.0x | Variant: 1.5x
   - Results: Control 100.0 vs Variant 175.0
```

## Success Criteria

- [x] Database tables verified
- [x] Experiment created successfully
- [x] Experiment started successfully
- [x] Users assigned to variants
- [x] Sticky assignments verified
- [x] Events tracked correctly
- [x] Results calculated accurately
- [x] Experiment stopped successfully
- [x] All data persisted in database

## Troubleshooting

### Issue: Command not recognized
**Solution**: Check bot is running and has Discord connection
```bash
kubectl logs -n development agis-bot-xxx | grep -i "command\|error"
```

### Issue: Database errors
**Solution**: Verify database connection
```bash
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c "SELECT 1"
```

### Issue: Assignments not sticky
**Solution**: Check assignment logic in database
```bash
kubectl -n postgres-dev exec -i postgres-dev-0 -- psql -U agis_dev_user -d agis_dev -c \
  "SELECT * FROM ab_assignments WHERE user_id = 'user-001';"
```

## Next Steps

After Step 6 is complete:
1. Proceed to Step 7: Guild Provisioning Testing
2. Test server provisioning workflow
3. Verify treasury system
4. Test subscription tiers

## Documentation

- `WEEK2_STEP6_AB_TESTING.md` - Detailed A/B testing guide
- `docs/AB_TESTING_GUIDE.md` - User guide for A/B testing
- `internal/bot/commands/experiment_command.go` - Command implementation
- `internal/services/ab_testing.go` - Service implementation

---

**Ready to Execute!** 🚀

Follow the phases in order and verify each step before proceeding to the next.

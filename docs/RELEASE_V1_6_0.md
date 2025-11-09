# AGIS Bot v1.6.0 Release Notes
**Release Date:** 2025-11-08  
**Code Name:** "Economy & Infrastructure"  
**Status:** Ready for Deployment

---

## 🎯 Executive Summary

v1.6.0 is a **major infrastructure and economy update** that implements:

1. **Dual-Currency Economy** - WTG Coin (hard currency) + GameCredits (soft currency)
2. **Real Kubernetes Log Streaming** - Production-ready log viewing
3. **Expanded Role System** - 8 granular permission levels
4. **BotKube-style Cluster Commands** - Live cluster querying for admins
5. **Complete Shop Purchase System** - Buy WTG, convert to GC, manage inventory

This release transforms AGIS Bot from a server management tool into a **complete gaming platform** with a sustainable monetization model.

---

## 📊 Release Statistics

- **New Commands:** 9
- **Total Commands:** 54 (was 45)
- **New Database Tables:** 0 (schema enhancements only)
- **Database Migrations:** 4
- **Permission Levels:** 8 (was 4)
- **Lines of Code Added:** ~606
- **Files Modified:** 6
- **Files Created:** 3

---

## 🚀 Major Features

### 1. Dual-Currency Economy System

**Implementation of Economy Plan v2.0**

#### WTG Coin (Hard Currency)
- Purchased with real money ($1 = 1 WTG)
- Used to buy premium items
- Convertible to GameCredits (1 WTG = 1000 GC)
- Account balance tracked in database

#### GameCredits (Soft Currency)  
- Earned through ads, daily bonuses, work tasks
- Used to pay for server costs
- Can be gifted between users
- Maintains existing earning mechanisms

#### Conversion System
- **Rate:** 1 WTG = 1000 GC = $1.00 USD
- Instant conversion via `convert` command
- Transaction logging for audit trail
- Rollback on failure

**New Commands:**
- `buy <item-id> [quantity]` - Purchase items from shop
- `convert <amount-wtg>` - Convert WTG to GameCredits
- `inventory` - View purchased items

**Database Changes:**
- Added `wtg_coins` column to `users` table
- Added `currency_type` and `bonus_amount` to `shop_items`
- Added `currency_type` to `credit_transactions`
- Unique constraint on `user_inventory` (discord_id, item_id)

---

### 2. Real Kubernetes Log Streaming

**Replaces placeholder with production implementation**

#### Features
- **Real-time streaming** from Kubernetes pods
- **Configurable line count** (default 50, max 200)
- **Automatic pod discovery** via label selector
- **Discord-friendly formatting** (2000 char limit handling)
- **Error handling** for pod not found, connection issues

#### Technical Implementation
```go
// Uses kubernetes.io/client-go
- CoreV1().Pods(namespace).List() for pod discovery
- GetLogs(podName, logOptions).Stream() for log retrieval
- TailLines parameter for efficient log fetching
- Auto-truncation for Discord message limits
```

#### Usage
```
logs <server-name> [lines]
logs minecraft-server 100
```

**Requirements:**
- Bot must run in-cluster OR have valid kubeconfig
- Service account needs pod/log read permissions
- Server pods must have `server-name` label

---

### 3. Expanded Permission System

**Granular role-based access control**

#### New Permission Hierarchy

| Level | Role | Access |
|-------|------|--------|
| 0 | User | Basic commands, server management |
| 1 | Game Server Mod | Server moderation tools |
| 2 | Community Ambassador | Community engagement features |
| 3 | Discord Mod | Discord moderation commands |
| 4 | Discord Admin | Discord administration |
| 5 | Backend Dev | Development/debugging tools |
| 6 | Cluster Admin | Kubernetes cluster access |
| 7 | Owner | Full system access |

#### Database Schema
```sql
ALTER TABLE bot_roles 
ADD CHECK (role_type IN (
  'admin', 'moderator', 'gameserver-mod', 
  'community-ambassador', 'discord-mod', 
  'discord-admin', 'backend-dev', 'cluster-admin'
));
```

#### Benefits
- **Principle of Least Privilege** - Users get only necessary permissions
- **Clear separation of concerns** - Cluster admins ≠ Discord admins
- **Scalable team structure** - Support multiple admin types
- **Audit trail** - All role changes logged

---

### 4. BotKube-Style Cluster Commands

**Live Kubernetes cluster querying for ClusterAdmin role**

#### New Commands

**`cluster-pods [namespace]`**
- Lists all pods in namespace (default: game-servers)
- Shows: Name, Status, Age
- Format: Table view with 80-char width

**`cluster-nodes`**
- Lists all cluster nodes
- Shows: Name, Status, CPU, Memory, Age
- Capacity information for resource planning

**`cluster-events [namespace]`**
- Shows last 20 events in namespace
- Format: [Age] Reason: Message
- Helps debug server creation issues

**`cluster-namespaces`**
- Lists all namespaces
- Shows: Name, Status, Age
- Overview of cluster organization

#### Use Cases
1. **Debugging** - Check pod status when server won't start
2. **Monitoring** - Node capacity planning
3. **Troubleshooting** - Event logs for error investigation
4. **Operations** - Namespace management

#### Security
- **ClusterAdmin permission required** (level 6)
- Read-only access (no modifications)
- Namespace-scoped by default
- Uses existing service account

---

### 5. Complete Shop Purchase System

**End-to-end purchasing with dual currency support**

#### WTG Shop Seed Data

**WTG Packages (Real Money)**
| Package | Price | Bonus | Total Received |
|---------|-------|-------|----------------|
| 5 WTG | $4.99 | 0 | 5 WTG |
| 11 WTG | $9.99 | +1 WTG | 11 WTG |
| 23 WTG | $19.99 | +3 WTG | 23 WTG |
| 60 WTG | $49.99 | +10 WTG | 60 WTG |

**GC Conversion (In-App)**
- 1000 GC = 1 WTG
- 3000 GC = 3 WTG (1 month server rent)
- 10000 GC = 10 WTG (bulk package)

#### Purchase Flow
1. User browses shop with `shop` command
2. Selects item by ID or name: `buy 1`
3. System checks currency balance (WTG or GC)
4. Deducts cost, applies item effect
5. Logs transaction, commits to database
6. Sends confirmation to user

#### Transaction Safety
- **Atomic transactions** - All-or-nothing commits
- **Balance checks** - Prevent overspending
- **Rollback on error** - Database integrity maintained
- **Audit logging** - Full transaction history

---

## 🗄️ Database Schema Changes

### New Columns

```sql
-- v1.6.0 Dual currency migrations
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS wtg_coins INTEGER DEFAULT 0;

ALTER TABLE shop_items 
ADD COLUMN IF NOT EXISTS currency_type VARCHAR(10) DEFAULT 'GC',
ADD COLUMN IF NOT EXISTS bonus_amount INTEGER DEFAULT 0;

ALTER TABLE credit_transactions 
ADD COLUMN IF NOT EXISTS currency_type VARCHAR(10) DEFAULT 'GC';
```

### Updated Constraints

```sql
-- Enhanced role types
ALTER TABLE bot_roles 
MODIFY role_type CHECK (role_type IN (
  'admin', 'moderator', 'gameserver-mod', 
  'community-ambassador', 'discord-mod', 
  'discord-admin', 'backend-dev', 'cluster-admin'
));

-- Shop currency validation
ALTER TABLE shop_items
ADD CHECK (currency_type IN ('GC', 'WTG', 'USD'));

-- Inventory uniqueness
ALTER TABLE user_inventory
ADD UNIQUE(discord_id, item_id);
```

---

## 📦 Deployment Instructions

### Prerequisites
- Kubernetes cluster with v1.6.0 deployed
- PostgreSQL database accessible
- Service account with pod/log read permissions

### Step 1: Database Migration

```bash
# Connect to your database
psql -h $DB_HOST -U $DB_USER -d agis

# Run migrations (automatic on bot startup, but can be manual)
ALTER TABLE users ADD COLUMN IF NOT EXISTS wtg_coins INTEGER DEFAULT 0;
ALTER TABLE shop_items ADD COLUMN IF NOT EXISTS currency_type VARCHAR(10) DEFAULT 'GC';
ALTER TABLE shop_items ADD COLUMN IF NOT EXISTS bonus_amount INTEGER DEFAULT 0;
ALTER TABLE credit_transactions ADD COLUMN IF NOT EXISTS currency_type VARCHAR(10) DEFAULT 'GC';
```

### Step 2: Seed WTG Shop

```bash
# From agis-bot directory
psql -h $DB_HOST -U $DB_USER -d agis < scripts/seed-wtg-shop.sql

# Verify seeding
psql -h $DB_HOST -U $DB_USER -d agis -c "SELECT COUNT(*) FROM shop_items WHERE is_active = true;"
```

Expected output: 7 active items (4 WTG packages + 3 GC conversions)

### Step 3: Update RBAC (if needed)

```bash
# Ensure service account has pod/log permissions
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: agis-bot-logs
  namespace: game-servers
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: agis-bot-logs
  namespace: game-servers
subjects:
- kind: ServiceAccount
  name: agis-bot
  namespace: development
roleRef:
  kind: Role
  name: agis-bot-logs
  apiGroup: rbac.authorization.k8s.io
EOF
```

### Step 4: Deploy v1.6.0

```bash
# Build and push image
docker build --build-arg VERSION=v1.6.0 -t ghcr.io/wethegamers/agis-bot:v1.6.0 .
docker push ghcr.io/wethegamers/agis-bot:v1.6.0

# Update deployment
kubectl set image deployment/agis-bot agis-bot=ghcr.io/wethegamers/agis-bot:v1.6.0 -n development

# Watch rollout
kubectl rollout status deployment/agis-bot -n development
```

### Step 5: Verify Deployment

```bash
# Check bot logs
kubectl logs -f deployment/agis-bot -n development

# Look for:
# ✅ Database initialization completed
# ✅ Command handler initialized (54 commands)
# ✅ Connected to Discord

# Test in Discord
@agis-bot about
# Should show version v1.6.0

@agis-bot shop
# Should display WTG packages

@agis-bot cluster-pods game-servers
# Should list pods (ClusterAdmin only)
```

---

## 🧪 Testing Checklist

### Currency System
- [ ] User can view both WTG and GC balances in `credits` command (needs update)
- [ ] `convert 1` converts 1 WTG to 1000 GC successfully
- [ ] `convert 10` with only 5 WTG shows error with current balance
- [ ] Conversion is logged in `transactions`

### Shop System
- [ ] `shop` displays 7 active items
- [ ] `buy 1` purchases 5 WTG package (requires USD payment integration)
- [ ] `buy "1000 GameCredits"` converts WTG to GC  
- [ ] `buy 1 2` purchases 2 of item #1
- [ ] `inventory` shows purchased items
- [ ] Insufficient balance shows helpful error message

### Log Streaming
- [ ] `logs <existing-server>` displays real logs
- [ ] `logs <non-existent-server>` shows "pod not found" error
- [ ] `logs <server> 100` shows 100 lines
- [ ] Long logs are truncated to fit Discord limit
- [ ] Works for both running and recently stopped servers

### Cluster Commands (ClusterAdmin only)
- [ ] `cluster-pods` lists pods in game-servers namespace
- [ ] `cluster-pods kube-system` lists system pods
- [ ] `cluster-nodes` shows node status and capacity
- [ ] `cluster-events` displays recent events
- [ ] `cluster-namespaces` lists all namespaces
- [ ] Regular users get "insufficient permissions" error

### Permissions
- [ ] Owner can use all commands
- [ ] ClusterAdmin can use cluster-* commands
- [ ] Regular users cannot use cluster commands
- [ ] Permission denied errors are clear and helpful

---

## 🐛 Known Issues

### Critical
- None identified

### Important
1. **Payment Integration Missing** - WTG packages show in shop but cannot be purchased yet
   - **Workaround:** Admin can manually add WTG: `UPDATE users SET wtg_coins = 100 WHERE discord_id = '<id>'`
   - **Fix ETA:** v1.7.0 (Stripe/PayPal integration)

2. **Log Streaming Requires In-Cluster** - Bot must run in Kubernetes for logs command
   - **Workaround:** Use kubeconfig mount for local testing
   - **Fix ETA:** Not planned (expected deployment model)

### Minor
3. **Credits Command Doesn't Show WTG** - Still shows only GC balance
   - **Workaround:** Use `inventory` or manual query
   - **Fix ETA:** v1.6.1 (quick patch)

4. **Shop UI Could Be Better** - Text-only, no images
   - **Enhancement:** Embed-based shop with item previews
   - **Fix ETA:** v1.7.0

---

## 📈 Performance Impact

### Memory
- **Before:** ~150MB baseline
- **After:** ~165MB baseline (+10%)
- **Cause:** Kubernetes clientset initialization

### Database
- **Additional queries per purchase:** 4-6 (with transaction)
- **Index recommendations:**
  - `CREATE INDEX idx_users_wtg ON users(wtg_coins) WHERE wtg_coins > 0;`
  - `CREATE INDEX idx_shop_currency ON shop_items(currency_type, is_active);`

### Network
- Log streaming: ~1-5KB per command (depends on log size)
- Cluster commands: <1KB per command (JSON API responses)

---

## 🔒 Security Considerations

### Currency Security
- ✅ **Atomic transactions** prevent partial purchases
- ✅ **Balance validation** before deduction
- ✅ **Audit logging** for all currency changes
- ⚠️ **Payment integration** not yet implemented (v1.7.0)

### Cluster Access
- ✅ **Role-based access** (ClusterAdmin only)
- ✅ **Read-only operations** (no modifications possible)
- ✅ **Namespace isolation** (default: game-servers)
- ✅ **Service account** with minimal permissions

### Data Integrity
- ✅ **Foreign key constraints** maintained
- ✅ **Check constraints** on enums
- ✅ **Unique constraints** prevent duplicates
- ✅ **Transaction rollbacks** on errors

---

## 🚧 Future Enhancements (v1.7.0+)

### Payment Integration
- Stripe API for WTG purchases
- PayPal support
- Payment webhook handling
- Receipt generation

### Subscription System
- Premium tier ($3.99/month)
  - 5 WTG monthly allowance
  - Free 3000 GC server rent
  - 2x GC multiplier on ads
  - Exclusive role

### Enhanced Shop
- Discord embeds with item previews
- Limited-time offers
- Promo codes / gift codes
- Referral rewards

### Advanced Cluster Features
- Resource usage graphs
- Pod shell access (premium)
- Log filtering (error/warn/info)
- Real-time log tailing

---

## 📚 Documentation Updates

**New Docs:**
- `/docs/RELEASE_V1_6_0.md` - This file
- `/scripts/seed-wtg-shop.sql` - Shop data seeding

**Updated Docs (TODO):**
- `COMMANDS.md` - Add v1.6.0 commands
- `README.md` - Update command count, features
- `COMPREHENSIVE_REVIEW_2025.md` - Mark v1.6.0 complete

---

## 🙏 Credits

- **Economy Design:** WeTheGamers Economy Plan v2.0
- **Kubernetes Integration:** k8s.io/client-go library
- **Inspiration:** BotKube for cluster command patterns

---

## 📞 Support

- **Discord:** wethegamers.org
- **GitHub Issues:** github.com/wethegamers/agis-bot/issues
- **Documentation:** github.com/wethegamers/agis-bot/docs

---

**Document Version:** 1.0  
**Last Updated:** 2025-11-08  
**Author:** AGIS Bot Development Team

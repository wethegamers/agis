# AGIS Bot - Command Analysis & Next-Gen Design

## 📊 Current Command Inventory

### User Commands (13)
| Command | Purpose | Status |
|---------|---------|--------|
| `servers` | List user's servers | ✅ Core |
| `create <game> [name]` | Deploy server | ✅ Core |
| `stop <server>` | Stop server | ✅ Core |
| `delete <server>` | Delete server | ✅ Core |
| `export <server>` | Export save files | ✅ Core |
| `diagnostics <server>` | Health check | ✅ Core |
| `ping [server]` | Connectivity test | ✅ Core |
| `credits` | Check balance | ✅ Core |
| `credits earn` | Ad dashboard | ✅ Monetization |
| `work` | Earn credits (task) | ✅ Monetization |
| `daily` | Daily bonus | ✅ Monetization |
| `lobby list/add/remove/my` | Public lobby | ✅ Social |
| `help` | Help menu | ✅ Core |

### Moderator Commands (3)
| Command | Purpose | Status |
|---------|---------|--------|
| `mod-servers` | View all servers | ✅ Oversight |
| `mod-control <user> <server> <action>` | Control user servers | ✅ Oversight |
| `mod-delete <server-id>` | Delete any server | ✅ Oversight |

### Admin Commands (7)
| Command | Purpose | Status |
|---------|---------|--------|
| `admin status` | Cluster health | ✅ Infrastructure |
| `admin pods` | List pods | ✅ Infrastructure |
| `admin nodes` | List nodes | ✅ Infrastructure |
| `admin credits add/remove/check @user <amount>` | Credit management | ✅ Economy |
| `admin-restart` | Restart bot | ✅ Maintenance |
| `log-channel` | Configure logging | ✅ Configuration |
| `adopt <server> <user>` | Transfer ownership | ✅ Special |

### Owner Commands (5)
| Command | Purpose | Status |
|---------|---------|--------|
| `owner set-admin <@role>` | Add admin role | ✅ Permissions |
| `owner set-mod <@role>` | Add mod role | ✅ Permissions |
| `owner list-roles` | Show roles | ✅ Permissions |
| `owner remove-admin <@role>` | Remove admin | ✅ Permissions |
| `owner remove-mod <@role>` | Remove mod | ✅ Permissions |

**Total: 28 unique commands**

---

## 🔍 Common Bot Patterns (Industry Analysis)

### Popular Gaming Bot Features
Based on analysis of similar bots (game server management, economy, community):

#### User Profile & Stats
- ✅ `profile [@user]` - View user stats, servers, credits, join date
- ❌ `leaderboard [type]` - Top users by credits, servers, playtime
- ❌ `stats` - Personal statistics dashboard
- ❌ `history` - Command/server history

#### Server Management Extended
- ✅ `create` - Current
- ❌ `restart <server>` - Missing (only stop/start cycle)
- ❌ `start <server>` - Missing (auto-starts but no manual control)
- ❌ `rename <server> <new-name>` - Missing
- ❌ `clone <server> [new-name]` - Missing
- ❌ `backup <server>` - Missing (export exists but not backup)
- ❌ `restore <server> <backup-id>` - Missing
- ❌ `config <server> [setting] [value]` - Missing
- ❌ `logs <server> [lines]` - Missing (only via mod-control)
- ❌ `console <server>` - Missing (direct console access)
- ❌ `schedule <server> <action> <time>` - Missing

#### Social & Community
- ✅ `lobby` - Current
- ❌ `invite <@user> <server>` - Share server invite
- ❌ `favorite <server>` - Bookmark servers
- ❌ `favorites` - List bookmarked servers
- ❌ `review <server> <rating> [comment]` - Rate servers
- ❌ `report <server/user> <reason>` - Report abuse
- ❌ `block <@user>` - Block user from your servers

#### Economy Extended
- ✅ `credits`, `work`, `daily`, `credits earn` - Current
- ❌ `shop` - Purchase items/upgrades
- ❌ `inventory` - View purchased items
- ❌ `gift <@user> <amount>` - Transfer credits
- ❌ `transactions [limit]` - View credit history
- ❌ `subscription` - Manage premium
- ❌ `redeem <code>` - Promo codes

#### Bot Information
- ✅ `help` - Current
- ❌ `about` - Bot info, version, uptime
- ❌ `status` - Bot status & latency
- ❌ `invite` - Bot invite link
- ❌ `support` - Support server link
- ❌ `changelog` - Recent updates
- ❌ `roadmap` - Planned features

#### Notifications & Alerts
- ❌ `notify <on|off> <event>` - Server event notifications
- ❌ `alerts` - View active alerts
- ❌ `watch <server>` - Get notifications for server

---

## 🚀 Next-Gen Command Structure

### Design Principles
1. **Logical Grouping** - Commands organized by feature domain
2. **Consistent Naming** - Verb-first pattern (action-oriented)
3. **Slash Command Native** - All commands as Discord slash commands
4. **Subcommand Support** - Use Discord's subcommand structure
5. **Autocomplete** - Server names, games, etc.
6. **Ephemeral Responses** - Private replies for sensitive data
7. **Rich Embeds** - Visual consistency across responses

### Proposed Command Tree

```
/server
  ├─ list              # Your servers
  ├─ create            # Deploy new server
  ├─ start             # Start stopped server
  ├─ stop              # Stop running server
  ├─ restart           # Restart server
  ├─ delete            # Delete server
  ├─ rename            # Rename server
  ├─ clone             # Clone server config
  ├─ info              # Server details
  ├─ diagnostics       # Health check
  ├─ logs              # View logs
  ├─ console           # Direct console (premium)
  ├─ config            # Configure settings
  ├─ backup            # Create backup
  ├─ restore           # Restore from backup
  ├─ export            # Export save files
  ├─ schedule          # Schedule actions
  └─ transfer          # Transfer ownership

/lobby
  ├─ browse            # Browse all public servers
  ├─ search            # Search servers
  ├─ publish           # Make server public
  ├─ unpublish         # Make server private
  ├─ my-listings       # Your public servers
  ├─ invite            # Share invite link
  ├─ favorite          # Bookmark server
  ├─ favorites         # List bookmarks
  ├─ review            # Rate server
  └─ reviews           # View reviews

/credits
  ├─ balance           # Check balance
  ├─ earn              # Earning options (ads/work)
  ├─ gift              # Transfer to user
  ├─ history           # Transaction history
  ├─ shop              # Browse store
  ├─ inventory         # View purchases
  ├─ redeem            # Redeem code
  └─ daily             # Daily bonus

/profile
  ├─ view              # View profile
  ├─ stats             # Detailed statistics
  ├─ history           # Server/command history
  ├─ achievements      # Unlocked achievements
  ├─ settings          # User preferences
  └─ notifications     # Notification settings

/leaderboard
  ├─ credits           # Top credit holders
  ├─ servers           # Most servers
  ├─ playtime          # Most playtime
  └─ contributions     # Top contributors

/mod
  ├─ servers           # View all servers
  ├─ control           # Control user server
  ├─ delete            # Delete server
  ├─ ban               # Ban user
  ├─ warn              # Warn user
  ├─ reports           # View reports
  └─ logs              # Moderation logs

/admin
  ├─ cluster
  │   ├─ status        # Cluster health
  │   ├─ pods          # List pods
  │   ├─ nodes         # List nodes
  │   └─ resources     # Resource usage
  ├─ credits
  │   ├─ add           # Add credits
  │   ├─ remove        # Remove credits
  │   ├─ set           # Set balance
  │   └─ check         # Check balance
  ├─ bot
  │   ├─ restart       # Restart bot
  │   ├─ status        # Bot status
  │   ├─ logs          # Bot logs
  │   └─ config        # Bot configuration
  ├─ users
  │   ├─ list          # List all users
  │   ├─ lookup        # User details
  │   ├─ ban           # Ban user
  │   └─ unban         # Unban user
  └─ servers
      ├─ list          # All servers
      ├─ cleanup       # Force cleanup
      └─ adopt         # Transfer ownership

/owner
  ├─ roles
  │   ├─ set-admin     # Set admin role
  │   ├─ set-mod       # Set mod role
  │   ├─ list          # List roles
  │   ├─ remove-admin  # Remove admin
  │   └─ remove-mod    # Remove mod
  ├─ channels
  │   ├─ set-log       # Set log channel
  │   ├─ set-alerts    # Set alerts channel
  │   └─ list          # List channels
  ├─ config
  │   ├─ set           # Set config value
  │   ├─ get           # Get config value
  │   └─ reset         # Reset to default
  └─ maintenance
      ├─ enable        # Enable maintenance mode
      ├─ disable       # Disable maintenance mode
      └─ announce      # Send announcement

/info
  ├─ help              # Command help
  ├─ about             # Bot information
  ├─ status            # Bot & cluster status
  ├─ games             # Supported games
  ├─ pricing           # Credit costs
  ├─ premium           # Premium features
  ├─ changelog         # Recent updates
  ├─ roadmap           # Planned features
  ├─ support           # Support server
  └─ invite            # Bot invite link

/ping                  # Connectivity test (standalone)
```

---

## 📋 Priority Implementation Roadmap

### Phase 1: Core Improvements (v1.3)
**Goal: Fill critical gaps, improve UX**

1. **Server Management**
   - `/server start` - Manual start control
   - `/server restart` - Quick restart
   - `/server logs` - View logs (pagination)
   - `/server info` - Detailed server info embed

2. **User Profile**
   - `/profile view [@user]` - User profile card
   - `/profile stats` - Statistics dashboard

3. **Bot Info**
   - `/info about` - Bot information
   - `/info games` - Supported games list
   - `/info pricing` - Cost breakdown

4. **Slash Command Migration**
   - Convert all commands to proper subcommand structure
   - Add autocomplete for server names
   - Implement ephemeral responses for sensitive data

### Phase 2: Social Features (v1.4)
**Goal: Community engagement**

1. **Enhanced Lobby**
   - `/lobby browse` (with pagination)
   - `/lobby search <query>`
   - `/lobby favorite <server>`
   - `/lobby reviews <server>`

2. **Social Interactions**
   - `/server invite <@user>`
   - `/profile achievements`
   - `/leaderboard credits/servers`

### Phase 3: Advanced Features (v1.5)
**Goal: Power user & premium features**

1. **Backups & Scheduling**
   - `/server backup`
   - `/server restore <backup-id>`
   - `/server schedule <action> <time>`

2. **Economy Extended**
   - `/credits shop`
   - `/credits inventory`
   - `/credits gift <@user> <amount>`
   - `/credits redeem <code>`

3. **Notifications**
   - `/profile notifications` - Configure alerts
   - Server event webhooks

### Phase 4: Admin & Moderation (v1.6)
**Goal: Better management tools**

1. **Enhanced Moderation**
   - `/mod ban/warn`
   - `/mod reports`
   - `/mod logs`

2. **Admin Dashboard**
   - `/admin cluster resources`
   - `/admin users list/lookup`
   - `/admin servers cleanup`

---

## 🎯 Missing Features Summary

### High Priority
- ✅ Server start/restart commands
- ✅ User profile & stats
- ✅ Leaderboards
- ✅ Server logs viewing
- ✅ Bot about/status info
- ✅ Slash command structure refactor

### Medium Priority
- ⚠️ Backup & restore system
- ⚠️ Server scheduling
- ⚠️ Credit gifting
- ⚠️ Shop & inventory
- ⚠️ Favorites & bookmarks
- ⚠️ Server search

### Low Priority
- 🔹 Achievements system
- 🔹 Review/rating system
- 🔹 Server cloning
- 🔹 Direct console access (premium)
- 🔹 Promo code system

---

## 💡 Implementation Notes

### Technical Considerations

1. **Database Schema Updates**
   - Add `user_stats` table (playtime, commands_used, etc.)
   - Add `server_backups` table
   - Add `favorites` table
   - Add `transactions` table (credit history)
   - Add `achievements` table

2. **Caching Strategy**
   - Cache leaderboards (5min TTL)
   - Cache server lists (1min TTL)
   - Cache user profiles (5min TTL)

3. **Rate Limiting**
   - Credit operations: 5/min
   - Server actions: 10/min
   - Lobby browsing: 20/min

4. **Permission System**
   - Verified role enforcement (already implemented)
   - Premium tier detection
   - Server ownership validation

5. **Slash Command Migration**
   - Maintain backward compatibility
   - Deprecation warnings for text commands
   - Full migration by v2.0

---

## 📊 Competitive Analysis

### Similar Bots Analyzed
- **Pterodactyl Discord Bot** - Server management
- **GameServerManager** - Multi-game hosting
- **UnbelievaBoat** - Economy system
- **Dyno** - Moderation & management
- **MEE6** - Leveling & economy

### Key Takeaways
1. Slash commands are now standard (Discord's recommendation)
2. Subcommand grouping improves discoverability
3. Ephemeral responses for private data (credits, profiles)
4. Rich embeds with thumbnails/images improve engagement
5. Autocomplete for common inputs (server names, games)
6. Pagination for lists (servers, lobbies, logs)
7. Confirmation dialogs for destructive actions
8. Activity feeds/notifications for important events

---

## 🚀 Recommended Next Steps

1. **Immediate (v1.3)**
   - Implement `/server start`, `/server restart`, `/server logs`
   - Create `/profile view` with stats
   - Refactor commands to Discord's subcommand structure
   - Add `/info about` and `/info games`

2. **Short-term (v1.4)**
   - Implement leaderboards
   - Enhanced lobby with search/favorites
   - Credit history & gifting

3. **Long-term (v1.5+)**
   - Backup & restore system
   - Scheduling system
   - Shop & inventory
   - Achievements

---

**Document Version:** 1.0  
**Date:** 2025-11-08  
**Author:** WARP AI Analysis

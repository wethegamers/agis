# AGIS Bot v1.3.0 Release Notes

**Release Date:** 2025-11-08  
**Release Type:** Minor version - New Features  
**Status:** ✅ Deployed

---

## 🎯 Overview

Version 1.3.0 focuses on filling critical feature gaps identified through competitive analysis. This release adds essential user-facing commands for profiles, leaderboards, and enhanced server management.

---

## ✨ New Features

### User Profile System
- **`profile [@user]`** - View comprehensive user statistics
  - Credits balance and tier
  - Server statistics (total created, active, owned)
  - Activity metrics (commands used, last daily/work)
  - Join date and account age
  - Support for viewing other users' profiles

### Leaderboard System  
- **`leaderboard credits`** - Top 10 users by credit balance
- **`leaderboard servers`** - Top 10 users by server count
  - Visual medals (🥇🥈🥉) for top 3 positions
  - Real-time rankings from database

### Bot Information Commands
- **`about`** - Bot statistics and system information
  - Version, build info, and uptime
  - Platform statistics (users, servers, active servers)
  - System metrics (memory usage, goroutines)
  - Kubernetes/Agones integration status

- **`games`** - Supported games list with pricing
  - Detailed game information
  - Cost per hour breakdown
  - Default ports and features

### Enhanced Server Management
- **`restart <server>`** - Restart a running server
  - Graceful shutdown and restart cycle
  - 1 credit administrative cost
  - Status validation and error handling

- **`start <server>`** - Manually start a stopped server
  - Resume operations after stop
  - Clears stopped timestamp

- **`logs <server> [lines]`** - View server logs
  - Placeholder implementation
  - Up to 100 lines (default 20)
  - Full Kubernetes log streaming coming in v1.4

---

## 🗄️ Database Changes

### New Table: `user_stats`
```sql
CREATE TABLE user_stats (
    discord_id VARCHAR(32) PRIMARY KEY,
    total_servers_created INTEGER DEFAULT 0,
    total_commands_used INTEGER DEFAULT 0,
    total_credits_earned INTEGER DEFAULT 0,
    total_credits_spent INTEGER DEFAULT 0,
    last_command_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (discord_id) REFERENCES users(discord_id)
)
```

**Purpose:** Track user activity and enable statistics/analytics features

---

## 🔧 Technical Improvements

### Code Organization
- Created `v1_3_commands.go` consolidating new features
- Maintained backward compatibility with existing commands
- Enhanced error messages with actionable guidance

### Helper Functions
- `formatDuration()` - Human-readable time formatting
- `getMedal()` - Position-based emoji medals
- Improved database query patterns with fallbacks

### Performance
- Cached leaderboard queries (top 10 limit)
- Efficient user stats retrieval with COALESCE
- Graceful degradation when tables don't exist

---

## 📊 Command Count

**Before v1.3.0:** 28 commands  
**After v1.3.0:** 35 commands (+7 new)

### Breakdown:
- **User Commands:** 20 (+7)
- **Moderator Commands:** 3 (unchanged)
- **Admin Commands:** 7 (unchanged)
- **Owner Commands:** 5 (unchanged)

---

## 🚀 Deployment

**Build:** GitHub Actions (multi-arch: amd64, arm64)  
**Registry:** ghcr.io/wethegamers/agis-bot:v1.3.0  
**Cluster:** wtg-dev (agis-bot-dev namespace)  
**Deployment Status:** ✅ Successful

### Deployment Verification
```
✅ Role sync service started (interval: 10m0s)
✅ Modular command system initialized
✅ Agis bot logged in as Agis-Dev
```

---

## 📝 Usage Examples

### Profile Command
```
@Agis profile
@Agis profile @username
```

### Leaderboards
```
@Agis leaderboard credits
@Agis leaderboard servers
```

### Info Commands
```
@Agis about
@Agis games
```

### Server Management
```
@Agis restart minecraft-server
@Agis start terraria-world
@Agis logs cs2-competitive 50
```

---

## 🔜 What's Next (v1.4 Roadmap)

### Slash Command Migration
- Convert all commands to Discord slash command subcommands
- Implement autocomplete for server names
- Add ephemeral responses for sensitive data

### Enhanced Social Features
- Lobby search functionality
- Server favorites/bookmarks
- Server reviews and ratings

### Economy Extensions
- Credit transaction history
- Gift credits to other users
- Shop system foundation

---

## 🐛 Known Limitations

1. **Logs Command** - Currently placeholder, returns mock data
   - Full Kubernetes log streaming planned for v1.4
   
2. **User Stats Tracking** - Not retroactive
   - Statistics only track from v1.3.0 forward
   - Historical data not populated

3. **Leaderboards** - No caching yet
   - Direct database queries (fast for small datasets)
   - Caching layer planned for v1.4

---

## 📚 Documentation

- **Command Analysis:** [COMMAND_ANALYSIS_NEXTGEN.md](./COMMAND_ANALYSIS_NEXTGEN.md)
- **Command Reference:** [COMMANDS.md](../COMMANDS.md) *(needs update)*
- **Integration Guide:** [AGONES_INTEGRATION.md](./AGONES_INTEGRATION.md)

---

## 🙏 Acknowledgments

This release implements high-priority features identified through:
- Competitive analysis of similar Discord bots
- Review of industry best practices
- Gap analysis against current feature set
- Community feedback and requests

---

**Questions or Issues?** Contact the development team or open an issue on GitHub.

**Upgrade Path:** Automatic via Kubernetes deployment with `imagePullPolicy: Always`

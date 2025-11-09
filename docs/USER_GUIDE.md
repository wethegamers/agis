# AGIS Bot - User Guide

**Version:** 1.7.0  
**Last Updated:** 2025-01-09

Welcome to **WeTheGamers (WTG)** - The community-powered game server hosting platform powered by AGIS Bot!

---

## Table of Contents

1. [Getting Started](#getting-started)
2. [Economy System](#economy-system)
3. [Game Server Management](#game-server-management)
4. [Premium Subscription](#premium-subscription)
5. [Guild Treasury](#guild-treasury)
6. [Community Features](#community-features)
7. [Command Reference](#command-reference)
8. [FAQ](#faq)
9. [Support](#support)

---

## Getting Started

### First Steps

1. **Join the Discord Server**: [discord.gg/wethegamers]
2. **Verify Your Account**: Visit the dashboard to get the Verified role
3. **Claim Your Daily Bonus**: `@AGIS daily` (50 GC for free users, 100 GC for premium)
4. **Check Your Balance**: `@AGIS credits`

### Understanding the Economy

WTG uses a **dual-currency system**:

- **GameCredits (GC)**: Earned through ads, daily bonus, and work commands. Used to rent servers.
- **WTG Coins**: Purchased with real money ($1 = 1 WTG = 1000 GC). Used for premium purchases.

**Conversion**: 1 WTG = 1,000 GC (one-way conversion available)

---

## Economy System

### Earning GameCredits (Free Users)

#### Daily Bonus
```
@AGIS daily
```
- **Reward**: 50 GC (free) or 100 GC (premium)
- **Cooldown**: Once per 24 hours
- **Streak Bonus**: Coming soon!

#### Work Command
```
@AGIS work
```
- **Reward**: 10-30 GC (random)
- **Multiplier**: 3x for premium users (30-90 GC)
- **Cooldown**: 4 hours

#### Watch Ads (Coming Soon)
```
@AGIS earn
```
- **Reward**: 15 GC per ad (45 GC for premium)
- **Multiplier**: Premium users earn 3x (45 GC/ad)
- **Monthly Target**: 200 ads = 3,000 GC (free) or 9,000 GC (premium)

### Monthly Earning Potential

**Free Tier** (no subscription):
- Daily bonus: 50 GC × 30 days = **1,500 GC**
- Work commands: 20 GC avg × 6/day × 30 = **3,600 GC**
- Ad views: 15 GC × 200 ads = **3,000 GC**
- **Total**: ~**8,100 GC/month**

**Premium Tier** ($3.99/month):
- Daily bonus: 100 GC × 30 days = **3,000 GC**
- Work commands (3x): 60 GC avg × 6/day × 30 = **10,800 GC**
- Ad views (3x): 45 GC × 200 ads = **9,000 GC**
- Monthly allowance: **5 WTG = 5,000 GC**
- **Total**: ~**27,800 GC/month**

### Spending GameCredits

GameCredits are used to rent game servers:

| Game | Cost/Hour | 100 Hours |
|------|-----------|-----------|
| Minecraft | 30 GC | 3,000 GC |
| Terraria | 35 GC | 3,500 GC |
| Don't Starve Together | 60 GC | 6,000 GC |
| CS2 | 120 GC | 12,000 GC |
| Valheim | 120 GC | 12,000 GC |
| Project Zomboid | 135 GC | 13,500 GC |
| Rust | 220 GC | 22,000 GC |
| ARK (requires guild) | 240 GC | 24,000 GC |

### Transferring Credits

**Gift to Friends**:
```
@AGIS gift @username 500
```
- Transfer up to 1,000 GC at once
- Non-refundable

**Guild Treasury Deposit**:
```
@AGIS guild deposit [guild-name] 5000
```
- Pool resources with guild members
- **Non-refundable** - becomes guild property

---

## Game Server Management

### Creating a Server

```
@AGIS create <game> [server-name]
```

**Example**:
```
@AGIS create minecraft MyAwesomeServer
```

**Supported Games**:
- `minecraft` - Minecraft: Java Edition (30 GC/hr)
- `terraria` - Terraria (35 GC/hr)
- `dst` - Don't Starve Together (60 GC/hr)
- `cs2` - Counter-Strike 2 (120 GC/hr)
- `gmod` - Garry's Mod (95 GC/hr)
- `valheim` - Valheim (120 GC/hr)
- `7d2d` - 7 Days to Die (130 GC/hr)
- `pz` - Project Zomboid (135 GC/hr)
- `factorio` - Factorio (100 GC/hr)
- `rust` - Rust (220 GC/hr)
- `palworld` - Palworld (180 GC/hr)
- `satisfactory` - Satisfactory (240 GC/hr)
- `ark` - ARK: Survival Evolved (240 GC/hr, **guild required**)

### Managing Servers

**List Your Servers**:
```
@AGIS servers
```

**Get Server Details**:
```
@AGIS diagnostics <server-name>
```

**Stop a Server**:
```
@AGIS stop <server-name>
```
- Credits stop being charged immediately
- Server data preserved for 2 hours
- Use `export` to save data permanently

**Restart a Server**:
```
@AGIS restart <server-name>
```
- Cost: 1 GC
- Useful for applying config changes

**Start a Stopped Server**:
```
@AGIS start <server-name>
```
- Resume a previously stopped server

**Delete a Server**:
```
@AGIS delete <server-name>
```
- Type `confirm delete mine` when prompted
- **Permanent deletion** - cannot be undone
- Export saves first!

### Server Backups

**Export Server Data**:
```
@AGIS export <server-name>
```
- Saves server data to cloud storage (Minio)
- Encrypted and compressed
- 30-day retention
- **Free** for all users

**List Your Backups**:
```
@AGIS imports
```

**Restore a Backup** (Coming Soon):
```
@AGIS import <backup-id> <new-server-name>
```

---

## Premium Subscription

### Benefits

**Premium Subscription: $3.99/month**

1. **5 WTG Monthly Allowance** ($5 value)
2. **3x Earning Multiplier**
   - Ads: 15 GC → 45 GC
   - Work: 20 GC → 60 GC average
3. **Enhanced Daily Bonus** (50 GC → 100 GC)
4. **Free Server Rent** (3,000 GC/month value)
5. **Premium Discord Role** (exclusive badge)
6. **Priority Support** (faster response times)

### Value Calculation

**Monthly Value**: $5 WTG + $9 GC earnings + $3 free server = **$17 value**  
**Cost**: $3.99/month  
**ROI**: **4.3x return on investment**

### How to Subscribe

1. Visit: https://wethegamers.org/shop
2. Select "Premium Subscription - $3.99/month"
3. Complete Stripe checkout
4. Benefits activate **instantly** (zero wait)

### Check Subscription Status

```
@AGIS subscribe status
```

Shows:
- Current tier (Free or Premium)
- Expiration date
- Days remaining
- Active benefits
- Current balances

### Cancel Subscription

```
@AGIS subscribe cancel
```

- **Benefits remain active until expiration date**
- No refunds (standard practice)
- Can reactivate anytime

---

## Guild Treasury

**Guilds** are the ultimate team feature - pool resources to afford Titan-tier servers impossible for solo players!

### Why Guilds?

**Solo Player Math (Rust 220 GC/hr)**:
- Premium user: 27,800 GC/month
- Can afford: ~126 hours/month

**Guild Math (5 premium members)**:
- Combined: 139,000 GC/month
- Can afford: ~579 hours/month
- **Plus** access to ARK (240 GC/hr) which requires guilds

### Creating a Guild

```
@AGIS guild create "Elite Raiders"
```

- You become the guild **owner**
- Can invite members
- Manage guild treasury

### Guild Roles

- **Owner**: Full control, can invite admins and members
- **Admin**: Invite members, authorize spending
- **Member**: Deposit credits, view treasury

### Depositing to Guild

```
@AGIS guild deposit "Elite Raiders" 5000
```

⚠️ **WARNING: NON-REFUNDABLE**

Once deposited, credits belong to the guild forever. Even if you leave, you cannot get them back.

### Guild Benefits

1. **Shared Treasury**: Pool resources with trusted friends
2. **Titan Servers**: Access ARK (240 GC/hr) and other high-end games
3. **Contribution Tracking**: See who's contributing fairly
4. **Co-Management**: Multiple admins can manage servers

### Guild Commands

**View Guild Info**:
```
@AGIS guild info "Elite Raiders"
```

**Invite Member**:
```
@AGIS guild invite "Elite Raiders" @username
```
(Owner/Admin only)

**View Members**:
```
@AGIS guild members "Elite Raiders"
```
Shows contribution leaderboard

**List Your Guilds**:
```
@AGIS guild list
```

**Leave Guild**:
```
@AGIS guild leave "Elite Raiders"
```
⚠️ Deposits are NOT refunded

---

## Community Features

### Public Lobby

**Browse Public Servers**:
```
@AGIS publiclobby
```

**Make Your Server Public**:
```
@AGIS togglepublic <server-name>
```

**Search Servers**:
```
@AGIS search minecraft
```

### Server Reviews

**Write a Review**:
```
@AGIS review <server-id> 5 Amazing server with great community!
```

- Rating: 1-5 stars
- Comment: Max 500 characters
- One review per server (can update)

**Read Reviews**:
```
@AGIS reviews <server-id>
```

Shows:
- Average rating
- Review count
- 5 most recent reviews

### Leaderboards

**Top Players by Credits**:
```
@AGIS leaderboard credits
```

**Top Server Owners**:
```
@AGIS leaderboard servers
```

### Social Commands

**View Profile**:
```
@AGIS profile [@user]
```

**Transaction History**:
```
@AGIS transactions
```

**Achievements** (Coming Soon):
```
@AGIS achievements
```

---

## Command Reference

### Quick Commands

| Command | Description | Cooldown |
|---------|-------------|----------|
| `help` | Show all commands | - |
| `credits` | Check balance | - |
| `daily` | Claim daily bonus | 24h |
| `work` | Earn random GC | 4h |
| `servers` | List your servers | - |
| `create <game> [name]` | Create server | - |
| `stop <name>` | Stop server | - |
| `delete <name>` | Delete server | - |
| `diagnostics <name>` | Server details | - |
| `publiclobby` | Browse servers | - |
| `subscribe` | Manage premium | - |

### All Commands by Category

**User Commands** (everyone):
- `help`, `manual`, `man` - Help system
- `credits`, `credits_earn` - Balance and earning info
- `daily` - Daily bonus (50 GC or 100 GC premium)
- `work` - Work for GC (4hr cooldown)
- `profile [@user]` - View user profile
- `servers` - List your servers
- `create <game> [name]` - Create game server
- `stop <server>` - Stop server
- `start <server>` - Start stopped server
- `restart <server>` - Restart running server
- `delete <server>` - Delete server
- `diagnostics <server>` - Server diagnostics
- `export <server>` - Backup server data
- `publiclobby` - Browse public servers
- `togglepublic <server>` - Toggle public visibility
- `search <query>` - Search servers
- `review <id> <1-5> <comment>` - Review server
- `reviews <id>` - View reviews
- `gift @user <amount>` - Gift credits
- `transactions` - Transaction history
- `leaderboard [type]` - View leaderboards
- `subscribe` - Manage subscription
- `guild <action>` - Guild management
- `shop` - Browse WTG packages
- `buy <item-id>` - Purchase from shop
- `convert <wtg-amount>` - WTG to GC conversion
- `inventory` - View purchases
- `ping` - Bot latency

**Mod Commands** (moderators+):
- `mod servers` - View all servers
- `mod control <server> <action>` - Control any server
- `mod delete <server>` - Delete any server

**Admin Commands** (administrators+):
- `admin status` - System status
- `admin restart` - Restart bot
- `logchannel <#channel>` - Set log channel
- `adopt <server> @user` - Transfer server ownership
- `pricing list` - View game pricing
- `pricing update <game> <cost>` - Update pricing
- `pricing add <game> <cost>` - Add new game pricing

**Owner Commands** (bot owner only):
- `owner` - Owner control panel

---

## FAQ

### Economy

**Q: Can I get a refund for unused GameCredits?**  
A: No, GameCredits are non-refundable. Budget wisely!

**Q: What happens if my server runs out of credits?**  
A: Server stops automatically. Credits stop being charged. Restart when you have more GC.

**Q: Can I buy GameCredits directly?**  
A: You buy WTG Coins ($1 = 1 WTG), then convert to GC (1 WTG = 1,000 GC).

**Q: Why is premium 3x multiplier instead of 2x?**  
A: Economy Plan v4.0 updated to 3x to make premium more valuable and enable guild economics.

### Servers

**Q: How long do stopped servers persist?**  
A: 2 hours. After that, they're scheduled for cleanup. Export saves first!

**Q: Can I change my server's game type?**  
A: No, create a new server and export/import saves.

**Q: Why do some games cost more?**  
A: Prices reflect actual infrastructure costs (CPU, RAM). Rust needs more resources than Minecraft.

**Q: Why does ARK require a guild?**  
A: ARK costs 240 GC/hr (heavily modded). Solo users can't afford it profitably. Guilds pool resources.

### Premium

**Q: Do I get a refund if I cancel early?**  
A: No, but benefits remain active until expiration date.

**Q: Can I gift premium to someone?**  
A: Not yet, coming soon!

**Q: Does premium ever go on sale?**  
A: Occasionally during special events. Join Discord for announcements.

### Guilds

**Q: Can I get my deposits back if I leave?**  
A: **No**. Guild deposits are non-refundable by design. Only deposit with trusted friends.

**Q: What happens if guild owner leaves?**  
A: Ownership transfers to oldest admin, or oldest member if no admins.

**Q: Can guilds be deleted?**  
A: Only by owner. All members notified. Remaining balance is lost.

---

## Support

### Getting Help

**In Discord**:
```
@AGIS help
@AGIS manual <command>
```

**Submit a Ticket**: Use #support channel in Discord

**Bug Reports**: #bug-reports channel

**Feature Requests**: #suggestions channel

### Common Issues

**"Insufficient credits"**:
- Check balance: `@AGIS credits`
- Earn more: `@AGIS daily`, `@AGIS work`
- Consider premium for 3x multiplier

**"Server not starting"**:
- Check diagnostics: `@AGIS diagnostics <name>`
- May take 2-3 minutes to provision
- Check status: Server may show "creating"

**"Command not working"**:
- Make sure you're Verified (visit dashboard)
- Check cooldowns (daily, work commands)
- Mention bot: `@AGIS command`

### Contact

- **Discord**: https://discord.gg/wethegamers
- **Website**: https://wethegamers.org
- **Email**: support@wethegamers.org (business inquiries only)

---

## Quick Start Checklist

- [ ] Join Discord server
- [ ] Get Verified role
- [ ] Claim daily bonus (`@AGIS daily`)
- [ ] Earn some credits (`@AGIS work`)
- [ ] Create your first server (`@AGIS create minecraft MyServer`)
- [ ] Join the public lobby (`@AGIS publiclobby`)
- [ ] Consider premium for 3x earnings
- [ ] Join or create a guild for Titan servers

Welcome to **WeTheGamers** - Happy gaming! 🎮

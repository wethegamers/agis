# AGIS Bot - Complete Command Reference

## 🎯 Overview
AGIS (Agones GameServer Integration System) is a Discord bot that manages game servers through Kubernetes and Agones. It provides real-time server status, automated deployment, and comprehensive monitoring.

## 🔗 Key Features
- **Live Kubernetes Integration** - Real-time server status from Agones GameServers
- **Automated Deployment** - One-command server creation with automatic configuration
- **Cost Management** - Credit-based system with automatic billing
- **Public Lobby** - Share servers with the community
- **Enhanced Diagnostics** - Deep server health monitoring
- **Multi-Environment Support** - Development, staging, and production deployments

---

## 📚 Command Categories

### 🎮 User Commands (All Members)

#### **Server Management**
- `servers` - List your servers with live Kubernetes status
- `create <game> [name]` - Deploy new Agones GameServer
- `stop <server>` - Stop server to save credits  
- `delete <server>` - Delete your own server permanently
- `export <server>` - Export save files before cleanup

#### **Diagnostics & Testing**
- `diagnostics <server>` - Complete server health check with Kubernetes metrics
- `ping [server]` - Test connectivity to bot or server

#### **Credits & Economy**
- `credits` - Check your credit balance
- `credits earn` - Access ad dashboard (best earnings!)
- `work` - Complete infrastructure tasks (1h cooldown)
- `daily` - Claim daily bonus credits

#### **Guild Economy**
- `guild-create <name>` - Create a guild treasury and become owner
- `guild-invite <@user> <guild_id>` - Invite a member (owner/admin)
- `guild-deposit <guild_id> <amount>` - Deposit your GameCredits into the guild treasury
- `guild-treasury <guild_id>` - View balance and top contributors
- `guild-join <guild_id>` - How to join (invite required)

#### **Public Lobby**
- `lobby list` - Browse all public servers
- `lobby add <server> [description]` - Share your server publicly
- `lobby remove <server>` - Make server private
- `lobby my` - View your public servers

#### **General**
- `help` - Show this help menu

---

### 🛡️ Moderator Commands

#### **Server Oversight**
- `mod-servers` - View all user servers across platform
- `mod-control <user> <server> <action>` - Control any user's server
  - Actions: `stop`, `restart`, `info`, `logs`
- `mod-delete <server-id>` - Delete a user's server
- `confirm-delete <server-id>` - Confirm server deletion

---

### ⚙️ Admin Commands

#### **Cluster Management**
- `admin status` - Agones & Kubernetes cluster health status
- `admin pods` - List Kubernetes pods across namespaces
- `admin nodes` - List cluster nodes and resource usage
- `log-channel` - Configure Discord logging channels

#### **Bot Management**
- `admin-restart` - Restart the AGIS bot
- `admin-restart confirm` - Confirm restart
- `admin-restart confirm --force` - Force restart

#### **Credit Management**
- `admin credits add @user <amount>` - Add credits to user
- `admin credits remove @user <amount>` - Remove credits from user
- `admin credits check @user` - Check user balance

---

### 👑 Owner Commands

#### **Role Management**
- `owner set-admin <@role>` - Set admin role
- `owner set-mod <@role>` - Set moderator role
- `owner list-roles` - Show configured roles
- `owner remove-admin <@role>` - Remove admin role
- `owner remove-mod <@role>` - Remove moderator role

---

## 🎮 Supported Games & Costs

| Game | Cost/Hour | Default Port | Features |
|------|-----------|--------------|----------|
| **Minecraft** | 5 credits | 25565 | Java Edition, Mods supported |
| **CS2** | 8 credits | 27015 | Counter-Strike 2, Custom maps |
| **Terraria** | 3 credits | 7777 | Multiplayer worlds, Mods |
| **Garry's Mod** | 6 credits | 27015 | Custom gamemodes, Addons |

---

## 💡 Usage Examples

### Creating a Server
```
create minecraft my-survival-world
create cs2 competitive-server
create terraria modded-world
```

### Server Management
```
servers                    # List all your servers
diagnostics minecraft1     # Detailed health check
stop expensive-server      # Stop to save credits
delete old-server          # Permanent deletion
export important-server    # Download save files
```

### Public Lobby
```
lobby list                 # Browse community servers
lobby add minecraft1 "Friendly survival server!"
lobby remove private-server
lobby my                   # Your public servers
```

### Diagnostics & Monitoring
```
ping                       # Test bot connectivity
ping minecraft1           # Test specific server
diagnostics survival-world # Full health report
```

---

## 🔧 Advanced Features

### **Real-Time Status Integration**
- Live connection to Kubernetes API
- Agones GameServer lifecycle management
- Real-time resource usage monitoring
- Automatic health checks

### **Enhanced Diagnostics**
- Kubernetes pod status
- Resource consumption (CPU/RAM)
- Network connectivity tests
- Game-specific health checks
- Connection information

### **Automated Lifecycle Management**
- Automatic server deployment via Agones
- Credit-based billing system
- Cleanup scheduling for inactive servers
- Save file export before deletion

### **Multi-Environment Support**
- Development (`agones-dev` namespace)
- Staging environment
- Production environment
- Separate configurations per environment

---

## 🆘 Troubleshooting

### **Server Won't Start**
1. Check credit balance: `credits`
2. Verify server status: `diagnostics <server>`
3. Check for naming conflicts: `servers`

### **Connection Issues**
1. Test bot connectivity: `ping`
2. Check server status: `ping <server>`
3. Verify server is running: `servers`

### **Credit Issues**
1. Check balance: `credits`
2. Earn more: `credits earn` or `work`
3. Contact admin for credit adjustments

---

## 📊 Business Model

### **Free Tier**
- Earn credits through ads & work tasks
- Community-driven gameplay
- Access to all game types

### **Premium ($0.99/month)**
- Unlimited servers
- 2x ad earnings
- 100 monthly bonus credits
- Priority support

---

## 🔗 Getting Started

1. **Check your credits**: `credits`
2. **Create your first server**: `create minecraft`
3. **Monitor deployment**: `diagnostics <server-name>`
4. **Join when ready**: Connection info in `servers`
5. **Share with friends**: `lobby add <server-name>`

---

## 📞 Support

- Use `help` for quick command reference
- Use `ping` to test connectivity
- Use `diagnostics` for server issues
- Contact administrators for technical support

---

*AGIS Bot - Powered by Kubernetes, Agones, and community spirit! 🎮*

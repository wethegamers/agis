# Verified Role Features

**Date**: 2025-11-06  
**Version**: Added in commit `802abde`  
**Status**: ✅ Implemented and Deployed

---

## Overview

The agis-bot now includes two important features for managing the "Verified" Discord role:

1. **Sticky Verified Role** - Automatically re-adds the verified role if removed
2. **Audit Logging** - Logs verification events to Discord audit channel

---

## Feature 1: Sticky Verified Role

### **Purpose**
Once a user is verified (via WordPress integration or other means), the verified role should be permanent and cannot be removed manually, even by administrators.

### **How It Works**

1. **Event Monitoring**: Bot listens for `GuildMemberUpdate` events
2. **Role Change Detection**: Detects when verified role is removed from a user
3. **Automatic Restoration**: Immediately re-adds the verified role
4. **Audit Logging**: Logs the protection action to audit channel

### **Implementation**

**File**: `internal/bot/events.go`

```go
// HandleGuildMemberUpdate monitors role changes and makes verified role sticky
// If a verified user has their role removed, it will be automatically re-added
func (eh *EventHandlers) HandleGuildMemberUpdate(s *discordgo.Session, event *discordgo.GuildMemberUpdate) {
    // Check if user had verified role before update
    // Check if user has verified role after update
    // If role was removed, automatically re-add it
    // Log to audit channel
}
```

### **Configuration**

**Environment Variables**:
- `VERIFIED_ROLE_ID` - Discord role ID for the verified role
- `DISCORD_GUILD_ID` - Guild ID where role protection is active
- `LOG_CHANNEL_AUDIT` - Channel ID for audit logs

**Required Discord Intents**:
```go
IntentsGuildMembers // Required to receive GuildMemberUpdate events
```

### **Behavior**

#### **When Role is Removed**
1. Event detected: Verified role removed from user
2. Log message: `[RoleProtection] Verified role removed from Username#1234 (123...), re-adding (sticky)`
3. Action: Re-add verified role via Discord API
4. Discord log: Sent to audit channel with details
5. Result: User has verified role again (< 1 second)

#### **Audit Channel Message**
```
✅ Verified role automatically restored for Username#1234

Details:
- User ID: 1234567890
- Username: Username#1234
- Action: role_restored
- Reason: sticky_verified_role
```

### **Edge Cases**

**Q: What if bot doesn't have permission to add roles?**
- A: Error is logged to console and audit channel: `Failed to re-add verified role`

**Q: Can the role be removed by server owner?**
- A: No, the bot will re-add it automatically (role is truly "sticky")

**Q: What about bots removing the role?**
- A: Bot will restore it regardless of who/what removed it

**Q: Can this be bypassed?**
- A: Only by stopping the bot or removing its role management permissions

### **Testing**

1. Verify a user (give them the verified role)
2. Manually remove the verified role from that user
3. Observe: Role is immediately re-added
4. Check audit channel for protection log

---

## Feature 2: Audit Logging for Verification

### **Purpose**
When users are verified via the WordPress API integration, log the event to Discord's audit channel for transparency and record-keeping.

### **How It Works**

1. **API Verification**: WordPress calls `/api/verify-user` with user details
2. **Role Assignment**: Bot assigns verified role to the user
3. **Audit Log**: Event logged to Discord audit channel
4. **Database Log**: Event recorded in system_logs table

### **Implementation**

**File**: `internal/http/server.go`

```go
// After successfully adding verified role
if loggingService != nil {
    userTag := fmt.Sprintf("%s#%s", member.User.Username, member.User.Discriminator)
    loggingService.LogAudit(
        payload.DiscordID,
        "user_verified",
        fmt.Sprintf("✅ User %s has been verified via API", userTag),
        map[string]interface{}{
            "user_id":  payload.DiscordID,
            "username": userTag,
            "source":   "wordpress_api",
            "action":   "verified_role_assigned",
        },
    )
}
```

### **Configuration**

**Environment Variables**:
- `LOG_CHANNEL_AUDIT` - Channel ID where verification logs are sent
- `VERIFY_API_SECRET` - Shared secret for WordPress API authentication

### **Log Message Format**

**Discord Audit Channel**:
```
✅ User Username#1234 has been verified via API

Details:
- User ID: 1234567890
- Username: Username#1234
- Source: wordpress_api
- Action: verified_role_assigned
- Timestamp: 2025-11-06 18:25:00
```

**Console Log**:
```
verify-user: successfully verified user 1234567890 (Username)
```

**Database Log** (`system_logs` table):
```sql
{
  "level": "warn",
  "category": "audit",
  "user_id": "1234567890",
  "action": "user_verified",
  "message": "✅ User Username#1234 has been verified via API",
  "details": {
    "user_id": "1234567890",
    "username": "Username#1234",
    "source": "wordpress_api",
    "action": "verified_role_assigned"
  }
}
```

### **API Response**

**Success (Role Added)**:
```json
{
  "success": true,
  "message": "role_assigned"
}
```

**Success (Already Verified)**:
```json
{
  "success": true,
  "message": "already_verified"
}
```

**Note**: Audit logs are only sent when role is newly assigned, not when user is already verified.

---

## Integration with WordPress

### **API Endpoint**
```
POST /api/verify-user
Content-Type: application/json
X-WTG-Secret: <shared_secret>

{
  "discord_id": "1234567890",
  "username": "Username" // optional
}
```

### **Flow**

1. User registers on WordPress site
2. User links Discord account (OAuth)
3. WordPress calls agis-bot API to verify user
4. Bot assigns verified role
5. Audit log sent to Discord channel
6. User is now verified in Discord server

---

## Monitoring & Debugging

### **Check if Features are Active**

```bash
# Check bot logs for event handler registration
kubectl -n agis-bot-dev logs deployment/agis-bot | grep "event handlers"

# Check if guild members intent is enabled
kubectl -n agis-bot-dev logs deployment/agis-bot | grep "Intents"

# Check audit channel configuration
kubectl -n agis-bot-dev exec deployment/agis-bot -- env | grep LOG_CHANNEL_AUDIT
```

### **Test Sticky Role**

```bash
# 1. Manually remove verified role from a user in Discord
# 2. Check bot logs
kubectl -n agis-bot-dev logs -f deployment/agis-bot | grep RoleProtection

# Expected output:
# [RoleProtection] Verified role removed from Username#1234 (123...), re-adding (sticky)
# [RoleProtection] Successfully re-added verified role to Username#1234
```

### **Test Audit Logging**

```bash
# Call verification API
curl -X POST https://agis-bot.dev.wethegamers.org/api/verify-user \
  -H "Content-Type: application/json" \
  -H "X-WTG-Secret: $VERIFY_API_SECRET" \
  -d '{"discord_id": "1234567890", "username": "TestUser"}'

# Check audit channel in Discord for log message
# Check bot logs
kubectl -n agis-bot-dev logs deployment/agis-bot | grep "verify-user"
```

### **Query Audit Logs from Database**

```sql
-- Recent verification events
SELECT timestamp, user_id, action, message, details
FROM system_logs
WHERE category = 'audit'
  AND action IN ('user_verified', 'verified_role_protected')
ORDER BY timestamp DESC
LIMIT 10;

-- Count verifications per day
SELECT DATE(timestamp), COUNT(*)
FROM system_logs
WHERE category = 'audit' AND action = 'user_verified'
GROUP BY DATE(timestamp)
ORDER BY DATE(timestamp) DESC;
```

---

## Configuration Reference

### **Required Environment Variables**

```bash
# Core Configuration
DISCORD_TOKEN=Bot_<token>
DISCORD_GUILD_ID=<guild_id>
VERIFIED_ROLE_ID=<role_id>

# API Configuration
VERIFY_API_SECRET=<shared_secret_with_wordpress>

# Logging Configuration
LOG_CHANNEL_AUDIT=<channel_id_for_audit_logs>
```

### **Discord Bot Permissions**

Required permissions:
- `Manage Roles` - To add/restore verified role
- `View Audit Log` - Optional, for transparency
- `Send Messages` - To send audit logs to channel
- `Read Message History` - To access channels

Required intents:
- `GUILD_MEMBERS` - To receive GuildMemberUpdate events
- `GUILDS` - To access guild/member information

### **Discord Developer Portal Settings**

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Select your bot application
3. Go to "Bot" section
4. Enable "Privileged Gateway Intents":
   - ✅ **Server Members Intent** (REQUIRED for sticky role)
   - ✅ **Message Content Intent** (for commands)
5. Save changes
6. Restart bot to apply

---

## Troubleshooting

### **Issue: Sticky role not working**

**Symptoms**: Role can be removed and stays removed

**Checks**:
1. Verify `GUILD_MEMBERS` intent is enabled in Discord Developer Portal
2. Check bot logs for event registration: `grep "event handlers" logs`
3. Verify `VERIFIED_ROLE_ID` environment variable is set correctly
4. Confirm bot has `Manage Roles` permission
5. Ensure bot role is ABOVE verified role in role hierarchy

**Fix**:
```bash
# Check environment variable
kubectl -n agis-bot-dev exec deployment/agis-bot -- env | grep VERIFIED_ROLE_ID

# Check bot logs
kubectl -n agis-bot-dev logs deployment/agis-bot | grep -A 5 "event handlers"

# Restart deployment if needed
kubectl -n agis-bot-dev rollout restart deployment/agis-bot
```

### **Issue: No audit logs appearing**

**Symptoms**: Verification works but no Discord message

**Checks**:
1. Verify `LOG_CHANNEL_AUDIT` is set correctly
2. Confirm bot has permission to send messages in audit channel
3. Check if logging service is initialized
4. Verify channel ID is correct (use developer mode in Discord)

**Fix**:
```bash
# Check channel configuration
kubectl -n agis-bot-dev exec deployment/agis-bot -- env | grep LOG_CHANNEL

# Test logging manually (if you have debug command)
# Or trigger a verification and check logs
kubectl -n agis-bot-dev logs deployment/agis-bot | grep "LogAudit"
```

### **Issue: API verification fails**

**Symptoms**: API returns error, role not assigned

**Common Errors**:
- `not_configured` - Missing environment variables
- `member_not_found` - User not in Discord server
- `unauthorized` - Wrong API secret
- `failed_to_add_role` - Bot permission issue

**Debug**:
```bash
# Check configuration
kubectl -n agis-bot-dev exec deployment/agis-bot -- env | grep VERIFY

# Check API logs
kubectl -n agis-bot-dev logs deployment/agis-bot | grep "verify-user"

# Test API manually
curl -v -X POST https://agis-bot.dev.wethegamers.org/api/verify-user \
  -H "Content-Type: application/json" \
  -H "X-WTG-Secret: $VERIFY_API_SECRET" \
  -d '{"discord_id": "<your_discord_id>"}'
```

---

## Security Considerations

### **Sticky Role Protection**

**Pros**:
- ✅ Prevents accidental role removal
- ✅ Prevents malicious role removal by rogue mods
- ✅ Ensures verified status is permanent
- ✅ Audit trail of all protection events

**Cons**:
- ⚠️ Server owner cannot manually unverify users
- ⚠️ Requires bot to be online to enforce
- ⚠️ Could be seen as aggressive by some admins

**Workarounds**:
- To manually unverify: Temporarily stop the bot, remove role, restart bot
- Or: Implement an admin command to whitelist role removals
- Or: Add database flag to disable stickiness per user

### **API Security**

**Current**:
- Shared secret authentication via `X-WTG-Secret` header
- Constant-time comparison prevents timing attacks
- HTTPS encryption for API calls

**Recommendations**:
- ✅ Use strong, random API secrets (32+ characters)
- ✅ Rotate secrets periodically
- ✅ Monitor audit logs for unusual activity
- ✅ Rate limit API endpoint (TODO)
- ⚠️ Consider adding IP allowlist for WordPress servers

---

## Future Enhancements

### **Potential Improvements**

1. **Configurable Stickiness**
   - Add `VERIFIED_ROLE_STICKY=true/false` env var
   - Allow per-user overrides in database

2. **Admin Unverify Command**
   - `/unverify @user` command for mods
   - Requires mod role and logs to audit channel
   - Temporarily disables stickiness for that action

3. **Rate Limiting**
   - Limit API calls per IP/hour
   - Prevent abuse of verification endpoint

4. **Verification Expiry**
   - Optional: Verified role expires after X days
   - Requires re-verification via WordPress

5. **Multi-Server Support**
   - Support multiple guild IDs
   - Different verified roles per server

6. **DM Notifications**
   - Send DM to user when verified
   - Send DM when role is protected/restored

---

## References

- **Code**: 
  - `internal/bot/events.go` - Sticky role handler
  - `internal/http/server.go` - Verification API with logging
  - `main.go` - Event handler registration
  
- **Documentation**:
  - [WordPress Integration](WORDPRESS_INTEGRATION.md)
  - [Logging Service](../internal/services/logging.go)
  
- **Discord API**:
  - [Guild Member Update Event](https://discord.com/developers/docs/topics/gateway-events#guild-member-update)
  - [Privileged Intents](https://discord.com/developers/docs/topics/gateway#privileged-intents)

---

## Changelog

- **2025-11-06**: Initial implementation (commit `802abde`)
  - Added sticky verified role feature
  - Added audit logging for verification events
  - Added `GUILD_MEMBERS` intent requirement
  - Wired logging service to HTTP server

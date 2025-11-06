# WordPress Website to Discord Bot Integration

## Overview
This integration allows users who sign up on the WTG WordPress website (wethegamers.org) to automatically receive a "Verified Member" role in Discord. This ensures that only registered website users can interact with the Agis bot.

## Architecture

### WordPress Side
- **REST API Endpoints** (in `functions.php`):
  - `POST /wp-json/wtg/v1/request-bot-access` - Triggers verification
  - `GET /wp-json/wtg/v1/bot-access-status` - Checks verification status
- **User Dashboard**: Interactive button that calls the verification API
- **Discord OAuth**: Users must log in with Discord to capture their Discord ID

### Bot Side
- **API Endpoint**: `POST /api/verify-user`
- **Handler**: `verifyUserHandler()` in `internal/http/server.go`
- **Port**: 9090 (standard Prometheus port, shared with metrics)

## API Specification

### Request from WordPress
```http
POST http://agis-bot.development.svc.cluster.local:3000/api/verify-user
Content-Type: application/json
X-WTG-Secret: <shared_secret>

{
  "discord_id": "290955794172739584",
  "username": "nebakineza"
}
```

### Responses

#### Success - Role Assigned
```json
{
  "success": true,
  "message": "role_assigned"
}
```

#### Success - Already Verified
```json
{
  "success": true,
  "message": "already_verified"
}
```

#### Error - Unauthorized
```json
{
  "error": "unauthorized"
}
```

#### Error - Member Not Found
```json
{
  "error": "member_not_found"
}
```

## Configuration

### Required Environment Variables (in Vault)
The bot expects these secrets at `kubefirst/development/agis-bot`:

- `DISCORD_TOKEN` - Bot token for Discord API access
- `DISCORD_CLIENT_ID` - Discord application client ID
- `DISCORD_GUILD_ID` - The WTG Discord server ID
- `VERIFIED_ROLE_ID` - Discord role ID for "Verified Member"
- `VERIFY_API_SECRET` - Shared secret for WordPress authentication

### WordPress Configuration
In the WordPress admin, configure:

```php
// Bot API endpoint (default)
update_option('wtg_bot_api_url', 'http://agis-bot.development.svc.cluster.local:3000');

// Shared secret (must match VERIFY_API_SECRET in Vault)
update_option('wtg_bot_api_secret', 'your-secret-here');
```

## Security

1. **Header-Based Authentication**: The shared secret is sent in the `X-WTG-Secret` header, not in the request body
2. **Constant-Time Comparison**: Uses `subtle.ConstantTimeCompare()` to prevent timing attacks
3. **Idempotency**: The endpoint can be called multiple times safely - if the user already has the role, it returns success
4. **Member Validation**: Verifies the Discord user exists in the guild before attempting role assignment

## Discord Role Setup

### Creating the "Verified Member" Role

1. Open Discord Server Settings → Roles
2. Create a new role called "Verified Member"
3. Set permissions as needed (typically just basic member permissions)
4. Copy the Role ID:
   - Enable Developer Mode in Discord (User Settings → Advanced)
   - Right-click the role → Copy ID
5. Add the Role ID to Vault as `VERIFIED_ROLE_ID`

### Bot Permissions Required

The bot needs these permissions in Discord:
- `Manage Roles` - To assign the Verified Member role
- `View Server Members` - To check if user exists in guild

**Important**: The bot's role must be positioned **above** the "Verified Member" role in the role hierarchy, otherwise it cannot assign the role.

## Testing

### Local Testing (without Kubernetes)

```bash
# Set environment variables
export DISCORD_TOKEN="your_bot_token"
export DISCORD_GUILD_ID="your_guild_id"
export VERIFIED_ROLE_ID="your_role_id"
export VERIFY_API_SECRET="test-secret"

# Run the bot
go run .

# Test the endpoint
curl -X POST http://localhost:9090/api/verify-user \
  -H "Content-Type: application/json" \
  -H "X-WTG-Secret: test-secret" \
  -d '{"discord_id": "290955794172739584", "username": "testuser"}'
```

### Production Testing

1. Deploy the bot to the Kubernetes cluster
2. Log into wethegamers.org with Discord OAuth
3. Navigate to the user dashboard
4. Click "Request Bot Access"
5. Check Discord to verify the role was assigned
6. Check bot logs for verification:
   ```bash
   kubectl logs -n development deployment/agis-bot -f | grep verify-user
   ```

## Deployment

### 1. Update Vault Secrets

```bash
# Install vault CLI if not already installed
# Access vault at the appropriate endpoint for your cluster

# Add the required secrets
vault kv put kubefirst/development/agis-bot \
  DISCORD_TOKEN="..." \
  DISCORD_CLIENT_ID="..." \
  DISCORD_GUILD_ID="..." \
  VERIFIED_ROLE_ID="..." \
  VERIFY_API_SECRET="..."
```

### 2. Build and Push Docker Image

The GitHub Actions workflow will automatically build and push on commit to main:

```bash
git add internal/http/server.go docs/WORDPRESS_INTEGRATION.md
git commit -m "feat: update verify-user endpoint for WordPress integration"
git push origin main
```

### 3. Deploy to Kubernetes

The Helm chart is configured to use ExternalSecrets to pull from Vault:

```bash
helm upgrade --install agis-bot ./charts/agis-bot \
  --namespace development \
  --values ./charts/agis-bot/values.yaml
```

### 4. Verify Deployment

```bash
# Check pod status
kubectl get pods -n development -l app=agis-bot

# Check logs
kubectl logs -n development deployment/agis-bot -f

# Test the endpoint from within cluster
kubectl run -it --rm test --image=curlimages/curl --restart=Never -- \
  curl -X POST http://agis-bot.development.svc.cluster.local:9090/api/verify-user \
  -H "Content-Type: application/json" \
  -H "X-WTG-Secret: your-secret" \
  -d '{"discord_id": "123456789"}'
```

## Troubleshooting

### Bot not assigning roles

1. **Check bot permissions**: Ensure bot has `Manage Roles` permission
2. **Check role hierarchy**: Bot's role must be above "Verified Member" role
3. **Check logs**: Look for error messages in pod logs
4. **Verify secrets**: Ensure `VERIFIED_ROLE_ID` matches the actual Discord role ID

### WordPress API timing out

1. **Check network**: Ensure the droplet can reach the Kubernetes cluster
2. **Check service**: Verify the bot service is running and accessible
3. **Check secret**: Ensure `wtg_bot_api_secret` in WordPress matches `VERIFY_API_SECRET` in Vault

### "member_not_found" errors

1. **User not in Discord**: User must join the Discord server first
2. **Wrong Guild ID**: Verify `DISCORD_GUILD_ID` matches your Discord server
3. **Bot not in server**: Ensure the bot has been added to the server

## Future Enhancements

### Command Restriction
Add a check in bot commands to require the "Verified Member" role:

```go
func hasVerifiedRole(member *discordgo.Member, verifiedRoleID string) bool {
    for _, roleID := range member.Roles {
        if roleID == verifiedRoleID {
            return true
        }
    }
    return false
}

// In command handler:
member, err := s.GuildMember(guildID, userID)
if err != nil || !hasVerifiedRole(member, verifiedRoleID) {
    s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: "❌ You must register at https://wethegamers.org to use this bot!",
            Flags:   discordgo.MessageFlagsEphemeral,
        },
    })
    return
}
```

### Database Logging
Consider logging verification events to the database for audit purposes:

```go
// After successful role assignment
dbService.LogVerification(payload.DiscordID, payload.Username, time.Now())
```

### Webhook Support
Add a webhook callback from bot to WordPress to confirm role assignment:

```go
// After role assignment
go notifyWordPress(payload.DiscordID, "verified")
```

## References

- Bot Repository: https://github.com/wethegamers/agis-bot
- WordPress Site: https://wethegamers.org
- Discord Developer Portal: https://discord.com/developers/applications
- ExternalSecrets Operator: https://external-secrets.io/

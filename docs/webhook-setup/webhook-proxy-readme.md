# GitHub to Discord Webhook Proxy

## Problem
GitHub webhooks send raw JSON data to Discord, but Discord expects specific message formats. This causes the error:
```
{"message": "Cannot send an empty message", "code": 50006}
```

## Solution
This webhook proxy converts GitHub events to Discord-compatible messages.

## Supported Events
- **Ping**: Webhook connection test
- **Push**: Code commits to repository  
- **Pull Request**: PR opened/closed/merged
- **Issues**: Issue opened/closed/reopened
- **Releases**: New releases published

## Quick Setup

### Option 1: Run Locally (Testing)
```bash
python3 github-discord-webhook-proxy.py
```
Webhook URL: `http://your-server:8080`

### Option 2: Deploy to Kubernetes
```bash
# Build container
docker build -f Dockerfile.webhook-proxy -t github-discord-proxy .

# Deploy to cluster
kubectl create deployment github-discord-proxy --image=github-discord-proxy
kubectl expose deployment github-discord-proxy --type=LoadBalancer --port=80 --target-port=8080
```

### Option 3: Use GitHub Discord Bot (Recommended)
1. In Discord: Server Settings → Integrations
2. Add GitHub bot
3. Connect your repository
4. Configure event subscriptions

## GitHub Webhook Configuration
1. Go to your GitHub repository
2. Settings → Webhooks → Add webhook
3. **Payload URL**: Your proxy URL (e.g., `https://your-domain.com`)
4. **Content type**: `application/json`
5. **Events**: Select events you want (push, pull requests, issues, releases)

## Discord Message Examples

### Push Event
```
📝 Push to main
2 commit(s) pushed to wethegamers/agis-bot
Latest Commit: [a1b2c3d] fix: resolve Discord webhook integration
Author: sebpreece
Branch: main
```

### Pull Request
```
🔀 Pull Request Opened
#42: Add Discord webhook support
Author: sebpreece  
Branch: feature/discord → main
```

### Issues
```
🐛 Issue Opened
#15: Discord webhook returns empty message error
Author: sebpreece
Labels: bug, enhancement
```

## Environment Variables
- `PORT`: Server port (default: 8080)
- `DISCORD_WEBHOOK_URL`: Discord webhook URL (hardcoded in script)

## Customization
Edit `github-discord-webhook-proxy.py` to:
- Add more GitHub event types
- Customize Discord message formatting  
- Add filtering logic
- Include more event details

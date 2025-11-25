# GitHub to Discord Webhook Proxy

## 🎯 **Purpose**
This proxy service receives GitHub webhook events and formats them into Discord-compatible messages, solving the issue where GitHub webhooks send raw JSON that Discord rejects with "Cannot send an empty message".

## 🚀 **Quick Deploy**

```bash
cd deployments/webhook-proxy
./deploy-webhook-proxy.sh
```

## 📋 **Manual Setup**

### 1. **Build and Deploy**
```bash
# Navigate to webhook-proxy directory
cd deployments/webhook-proxy

# Build the container
docker build -f Dockerfile.webhook-proxy -t ghcr.io/wethegamers/github-discord-proxy:latest .

# Push to registry
docker push ghcr.io/wethegamers/github-discord-proxy:latest

# Deploy to Kubernetes
kubectl apply -f k8s-github-webhook-proxy.yaml
```

### 2. **Configure GitHub Webhook**
1. Go to your repository: `https://github.com/wethegamers/agis-bot/settings/hooks`
2. Click **"Add webhook"**
3. Set:
   - **Payload URL:** `https://github-webhook.euw.wtgg.org`
   - **Content type:** `application/json`
   - **Secret:** (leave empty)
   - **SSL verification:** Enable SSL verification
4. **Select events:**
   - ✅ Push events
   - ✅ Pull requests
   - ✅ Issues
   - ✅ Releases
   - ✅ Repository (for all events)
5. Click **"Add webhook"**

## 🎨 **Supported Events & Discord Format**

### **🏓 Ping (Webhook Test)**
- Title: "🏓 GitHub Webhook Connected"
- Color: Green
- Shows repository name

### **📤 Push Events**
- Title: "📤 Push to {repo}"
- Color: Purple
- Shows: Repository, branch, pusher, commit list

### **🔀 Pull Requests**
- Title: "🔀 Pull Request {action}"
- Colors: Green (opened), Red (closed), Purple (merged), Yellow (reopened)
- Shows: Repository, author, branch comparison

### **🐛 Issues**
- Title: "🐛 Issue {action}"
- Colors: Red (opened), Green (closed)
- Shows: Repository, author, issue number

### **🚀 Releases**
- Title: "🚀 New Release: {tag}"
- Color: Green
- Shows: Repository, author, tag, release notes (truncated)

## 🔧 **Configuration**

### **Environment Variables**
- `DISCORD_WEBHOOK_URL`: Your Discord webhook URL (from Vault secret)
- `PORT`: Server port (default: 8080)

### **Kubernetes Resources**
- **Deployment:** `github-discord-proxy` (1 replica)
- **Service:** `github-discord-proxy` (ClusterIP)
- **Ingress:** `github-webhook.euw.wtgg.org` (TLS enabled)

## 🧪 **Testing**

1. **Health Check:**
   ```bash
   curl https://github-webhook.euw.wtgg.org
   # Should return: {"status": "healthy", "service": "github-discord-proxy"}
   ```

2. **GitHub Webhook Test:**
   - Go to your webhook settings
   - Click the webhook you created
   - Click "Recent Deliveries"
   - Click "Redeliver" on any delivery to test

3. **Manual Test:**
   ```bash
   curl -X POST https://github-webhook.euw.wtgg.org \
     -H "Content-Type: application/json" \
     -H "X-GitHub-Event: ping" \
     -d '{"repository": {"full_name": "wethegamers/agis-bot"}}'
   ```

## 🔍 **Monitoring**

### **Check Deployment Status**
```bash
kubectl get pods -l app=github-discord-proxy
kubectl logs -l app=github-discord-proxy
```

### **Check Ingress**
```bash
kubectl get ingress github-discord-proxy
```

### **GitHub Webhook Deliveries**
- Go to: `https://github.com/wethegamers/agis-bot/settings/hooks`
- Click your webhook
- Check "Recent Deliveries" for success/failure status

## 🛠️ **Troubleshooting**

### **Common Issues**

1. **"Cannot send an empty message"**
   - This is fixed by the proxy - it formats all events properly

2. **Webhook not receiving events**
   - Check GitHub webhook delivery status
   - Verify ingress is working: `curl https://github-webhook.euw.wtgg.org`

3. **Discord messages not appearing**
   - Check webhook proxy logs: `kubectl logs -l app=github-discord-proxy`
   - Verify Discord webhook URL is correct in Vault

4. **SSL/TLS issues**
   - Verify cert-manager issued certificate: `kubectl get certificate github-webhook-tls`

## 📊 **Example Discord Messages**

After setup, you'll see formatted messages like:

- **Push:** "📤 Push to agis-bot - 3 commits pushed to main"
- **PR:** "🔀 Pull Request Opened - Fix webhook integration"  
- **Issue:** "🐛 Issue Opened - #42: Webhook proxy not working"
- **Release:** "🚀 New Release: v1.2.3 - Bug fixes and improvements"

This gives you comprehensive GitHub activity notifications in Discord with proper formatting!

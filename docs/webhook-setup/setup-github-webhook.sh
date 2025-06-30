#!/bin/bash
set -e

echo "🔗 GitHub to Discord Webhook Setup Guide"
echo "========================================"
echo ""

# Get the external IP
EXTERNAL_IP=$(kubectl get service github-discord-proxy -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

if [ -z "$EXTERNAL_IP" ]; then
    echo "❌ Could not get external IP for webhook proxy service"
    echo "Run: kubectl get service github-discord-proxy"
    exit 1
fi

echo "✅ Webhook proxy is deployed and accessible at: http://${EXTERNAL_IP}"
echo ""
echo "📋 GitHub Repository Webhook Configuration:"
echo "   1. Go to your GitHub repository: https://github.com/wethegamers/agis-bot"
echo "   2. Navigate to Settings → Webhooks → Add webhook"
echo "   3. Set Payload URL to: http://${EXTERNAL_IP}"
echo "   4. Set Content type to: application/json"
echo "   5. Set Secret to: (leave blank for now)"
echo "   6. Select events:"
echo "      ☐ Push events"
echo "      ☐ Pull request events" 
echo "      ☐ Issue events"
echo "      ☐ Release events"
echo "      ☐ Star events"
echo "   7. Click 'Add webhook'"
echo ""
echo "🧪 Test the webhook:"
echo "   - Push a commit to main branch"
echo "   - Create a pull request"
echo "   - Open/close an issue"
echo ""
echo "📊 Monitor webhook:"
echo "   kubectl logs -f -l app=github-discord-proxy"
echo ""
echo "🔧 Troubleshooting:"
echo "   - Check Discord channel for messages"
echo "   - Check webhook proxy logs: kubectl logs -l app=github-discord-proxy"
echo "   - Test manually: curl -X POST http://${EXTERNAL_IP} -H 'X-GitHub-Event: ping' -d '{}'"
echo ""

# Test webhook endpoint
echo "🔍 Testing webhook endpoint..."
if curl -s -f http://${EXTERNAL_IP} > /dev/null; then
    echo "✅ Webhook endpoint is accessible"
else
    echo "❌ Webhook endpoint is not accessible"
fi

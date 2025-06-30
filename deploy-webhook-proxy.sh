#!/bin/bash
set -e

echo "🚀 Deploying GitHub to Discord Webhook Proxy"

# Build and push the container image
echo "📦 Building webhook proxy container..."
docker build -f Dockerfile.webhook-proxy -t ghcr.io/wethegamers/github-discord-proxy:latest .

echo "📤 Pushing to container registry..."
docker push ghcr.io/wethegamers/github-discord-proxy:latest

echo "🎯 Deploying to Kubernetes..."
kubectl apply -f k8s-github-webhook-proxy.yaml

echo "⏳ Waiting for deployment to be ready..."
kubectl wait --for=condition=available --timeout=60s deployment/github-discord-proxy

echo "🌐 Getting service URL..."
kubectl get ingress github-discord-proxy

echo ""
echo "✅ Webhook proxy deployed successfully!"
echo ""
echo "📋 Next steps:"
echo "1. Go to your GitHub repository settings"
echo "2. Navigate to Settings → Webhooks → Add webhook"
echo "3. Set Payload URL to: https://github-webhook.euw.wtgg.org"
echo "4. Set Content type to: application/json"
echo "5. Select events you want to receive (push, pull requests, issues, releases)"
echo "6. Click 'Add webhook'"
echo ""
echo "🧪 Test the webhook by pushing a commit or creating a pull request"

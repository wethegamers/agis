#!/bin/bash
set -e

REPO="wethegamers/agis-bot"
IMAGE="ghcr.io/${REPO}:latest"
NAMESPACE="development"
DEPLOYMENT="agis-bot"

echo "🔍 Monitoring GitHub Actions workflow for ${REPO}..."
echo "📦 Waiting for image: ${IMAGE}"
echo ""

# Function to check if image exists in GHCR
check_image() {
    # Try to pull image manifest (requires authentication)
    skopeo inspect docker://${IMAGE} --creds="${GITHUB_USERNAME}:${GITHUB_PAT}" >/dev/null 2>&1
    return $?
}

# Wait for the workflow to complete and image to be available
echo "⏳ Waiting for image build to complete..."
echo "   Check progress: https://github.com/${REPO}/actions"
echo ""

RETRY_COUNT=0
MAX_RETRIES=60  # Wait up to 30 minutes (60 * 30s)

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if check_image; then
        echo "✅ Image is available in GHCR!"
        echo ""
        echo "🔄 Restarting deployment in cluster..."
        kubectl -n ${NAMESPACE} rollout restart deployment/${DEPLOYMENT}
        
        echo "⏳ Waiting for rollout to complete..."
        kubectl -n ${NAMESPACE} rollout status deployment/${DEPLOYMENT} --timeout=5m
        
        echo ""
        echo "✅ Deployment updated successfully!"
        echo ""
        echo "📊 Pod status:"
        kubectl -n ${NAMESPACE} get pods -l app=agis-bot
        
        exit 0
    fi
    
    RETRY_COUNT=$((RETRY_COUNT + 1))
    sleep 30
done

echo "❌ Timeout waiting for image build. Check workflow status manually."
echo "   https://github.com/${REPO}/actions"
exit 1

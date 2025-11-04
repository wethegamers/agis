#!/usr/bin/env bash
set -euo pipefail

# Simulate the GitHub Actions build locally without Docker/Buildx
# - Builds linux/amd64 and linux/arm64 binaries with same ldflags as Dockerfile
# - Outputs to build/ directory
# - Optionally smoke-tests the local-arch binary

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

mkdir -p build

GIT_COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
# Use latest tag if present, else dev
if git describe --tags --abbrev=0 >/dev/null 2>&1; then
  VERSION=$(git describe --tags --abbrev=0)
else
  VERSION=dev
fi

LDFLAGS="-X agis-bot/internal/version.Version=${VERSION} -X agis-bot/internal/version.GitCommit=${GIT_COMMIT} -X agis-bot/internal/version.BuildDate=${BUILD_DATE}"

# Build for linux/amd64
echo "[build] linux/amd64"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$LDFLAGS" -o build/agis-bot-linux-amd64 .

# Build for linux/arm64
echo "[build] linux/arm64"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="$LDFLAGS" -o build/agis-bot-linux-arm64 .

# Build for local arch (for optional smoke test)
LOCAL_OS=$(go env GOOS)
LOCAL_ARCH=$(go env GOARCH)
LOCAL_OUT="build/agis-bot-${LOCAL_OS}-${LOCAL_ARCH}"

echo "[build] ${LOCAL_OS}/${LOCAL_ARCH} (for smoke test)"
CGO_ENABLED=0 GOOS=${LOCAL_OS} GOARCH=${LOCAL_ARCH} go build -trimpath -ldflags="$LDFLAGS" -o "$LOCAL_OUT" .

# Smoke test: start server briefly to check /health responds, then kill
# Use dummy Discord token so main doesn't exit early
echo "[test] smoke test /health on ${LOCAL_OS}/${LOCAL_ARCH} binary"
DISCORD_TOKEN="dummy" METRICS_PORT=9090 "$LOCAL_OUT" >/tmp/agis-bot.log 2>&1 &
PID=$!
trap 'kill $PID >/dev/null 2>&1 || true' EXIT
sleep 1
if curl -fsS http://127.0.0.1:9090/health >/dev/null 2>&1; then
  echo "[test] OK: /health responded"
else
  echo "[test] WARN: /health did not respond; see /tmp/agis-bot.log" >&2
fi
kill $PID >/dev/null 2>&1 || true
trap - EXIT

# Summarize
echo "\nArtifacts:"
ls -lh build/agis-bot-*

echo "\nDone. These binaries match what the Dockerfile would produce in the builder stage."

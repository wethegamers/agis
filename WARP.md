# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Common commands

- Install deps
  ```bash path=null start=null
  go mod download
  ```
- Build (local binary)
  ```bash path=null start=null
  mkdir -p bin && go build -trimpath -o bin/agis-bot ./cmd
  ```
- Run locally
  ```bash path=null start=null
  DISCORD_TOKEN={{DISCORD_TOKEN}} DB_HOST={{DB_HOST}} METRICS_PORT=9090 go run ./cmd
  ```
- Lint (lightweight, no golangci configured)
  ```bash path=null start=null
  go vet ./... && test -z "$(gofmt -s -l .)" || (gofmt -s -l .; exit 1)
  ```
- Tests
  ```bash path=null start=null
  # run all (no tests yet present, but command is supported)
  go test ./...
  # run package
  go test ./internal/services
  # run single test by name (regex)
  go test ./internal/services -run '^TestName$'
  ```
- Docker (injects version metadata)
  ```bash path=null start=null
  docker build \
    --build-arg VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo dev) \
    --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
    --build-arg BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
    -t ghcr.io/{{owner}}/agis-bot:dev .
  docker run --rm -p 9090:9090 --env-file .env ghcr.io/{{owner}}/agis-bot:dev
  ```
- Helpful scripts
  ```bash path=null start=null
  # cross-compile like CI and smoke-test /health (may warn if HTTP server is not wired)
  ./scripts/simulate-ghcr-build.sh

  # simple integration check against a running instance on localhost:9090
  ./scripts/test-agis-bot.sh

  # wait for GHCR image (latest) then rollout restart in k8s (needs creds)
  GITHUB_USERNAME={{GITHUB_USERNAME}} GITHUB_PAT={{GITHUB_PAT}} ./scripts/watch-and-deploy.sh
  ```
- Helm (local/dev install)
  ```bash path=null start=null
  # set image repo/tag to a build you pushed
  helm upgrade --install agis-bot charts/agis-bot \
    -n development --create-namespace \
    --set image.repository=ghcr.io/{{owner}}/agis-bot \
    --set image.tag={{tag}}
  ```

## CI/CD and delivery

- GitHub Actions
  - Builds multi-arch images and pushes to GHCR: `.github/workflows/build-and-push.yml`.
  - An additional `build.yml` targets linux/arm64 for alpha tags.
- Argo Workflows
  - Release workflow `.github/workflows/main.yaml` submits `.argo/publish.yaml` then deploys `.argo/deploy.yaml` to environments: development → staging → production, with Discord webhook notifications.

## Runtime endpoints (when HTTP server is enabled)

- Port: 9090
- Endpoints: `/health`, `/healthz`, `/ready`, `/readyz`, `/info`, `/about`, `/version`, `/metrics`

## Configuration (env)

- Discord: `DISCORD_TOKEN`, `DISCORD_CLIENT_ID`, `DISCORD_GUILD_ID`
- Database: `DB_HOST`, `DB_NAME` (default agis), `DB_USER` (default root), `DB_PASSWORD`
- Metrics: `METRICS_PORT` (default 9090)
- WTG: `WTG_DASHBOARD_URL`
- Roles: `ADMIN_ROLES` (comma-separated), `MOD_ROLES`

## High-level architecture

### New goals (ayeT-Studios integration)
- Support Offerwall, Surveywall, and Rewarded Video monetization
- Implement S2S Conversion Callbacks with HMAC-SHA1 signature verification using AYET_API_KEY
- Idempotent rewards via conversionId to prevent duplicate credits
- Expose callback at `/ads/ayet/callback` and accept params: externalIdentifier|uid, currency|amount, conversionId, signature, custom_1..custom_4
- Configure AYET_API_KEY and AYET_CALLBACK_TOKEN via Vault → ExternalSecrets → Deployment env
- Client flow: `/earn` links to web dashboard; Rewarded Video can use client SDK callbacks or S2S
- Ops: record conversions in `ad_conversions` for audit and dedup; metrics via existing Prometheus counters (extend later)

- Entrypoint
  - `cmd/main.go`: main application entry for local builds and running.
  - Docker builds inject version info via ldflags into `internal/version`.
- Configuration
  - `internal/config`: loads env (supports `.env` via `godotenv`), exposes typed config structs.
- HTTP layer
  - `internal/http/server.go`: lightweight server for health/readiness, version, and Prometheus metrics.
- Domain/services
  - `internal/services/database.go`: PostgreSQL access with a “local mode” fallback when `DB_HOST` is empty; manages users, game servers, public lobby, command usage, bot roles, and cleanup scheduling.
  - `internal/services/*`: integration stubs for Agones, notifications, logging, savefiles, and cleanup workers.
  - `internal/agones`: client integration with Agones/Kubernetes APIs.
- Bot/commands
  - `internal/bot/commands/*`: command handlers grouped by role (user/mod/admin/owner) implementing server lifecycle, diagnostics, credits, and lobby features.
- Versioning
  - `internal/version`: `GetBuildInfo()` surfaced on `/info` and `/version` endpoints; populated by ldflags from Docker and scripts.
- Deployment manifests
  - Helm chart: `charts/agis-bot/` (Deployment, Service, Ingress, RBAC, ExternalSecrets, ConfigMap, tests).
  - Argo Workflows: `.argo/` for publish/deploy/release orchestration.
  - CI: `.github/workflows/` for image build/push and multi-env deployment.

## Key docs

- Overview and quick start: `README.md`
- Command reference: `COMMANDS.md`
- Agones integration: `docs/AGONES_INTEGRATION.md`
- Webhook setup: `docs/webhook-setup/`

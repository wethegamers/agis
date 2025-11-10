# AGIS Bot - GitHub Copilot Instructions

## Quick Reference Index

**Jump to**: [Architecture](#architecture) | [Dev Workflows](#development-workflows) | [Config](#configuration--secrets-management) | [Critical Patterns](#critical-patterns) | [K8s Deploy](#kubernetes-deployment) | [A/B Testing](#ab-testing--experimentation) | [Ad Conversion](#ad-conversion-workflows) | [Command Examples](#command-implementation-examples) | [Database](#database-schema--migrations) | [Troubleshooting](#troubleshooting-runbooks)

**Common Tasks**:
- [Add new command](#command-implementation-examples) → Implement `Command` interface, register in `handler.go`
- [Add new env var](#adding-new-features) → Config struct → Vault `{KEY}_{ENV}` → ExternalSecret → Deployment
- [Troubleshoot](#troubleshooting-runbooks) → Bot down | DB issues | Webhooks | Agones | Memory | Ad signatures
- [New game type](#pricing-system-blocker-1---critical) → Add to `game_pricing` table (never hardcode)
- [A/B test](#ab-testing--experimentation) → Create experiment → Assign variants → Record metrics

**Critical Files** (hot path):
- `main.go` - Service init, metrics
- `internal/bot/commands/handler.go` - Command routing (340L)
- `internal/services/database.go` - DB ops (1145L)
- `internal/services/pricing.go` - Dynamic pricing (BLOCKER 1)
- `internal/http/server.go` - Webhooks (705L)

## Project Overview

**AGIS Bot** = Production Discord automation for **WeTheGamers (WTG)** platform. Orchestrates game-server lifecycle via **Agones**, player economy, **Stripe** payments, HTTP/Prometheus telemetry.

**Stack**: Go 1.23 (`agis-bot` module) | discordgo | Agones | K8s client-go | PostgreSQL (raw SQL) | Minio/S3 | Stripe | Prometheus | Sentry | Vault+ExternalSecrets

**Deploy**: Kubernetes+Helm (`charts/agis-bot`) | CI/CD: GitHub Actions → Argo Workflows → ArgoCD | Envs: dev→sta→pro

## Architecture

### Core Components (internal/)

- **bot/commands/** - Discord command handlers. Each implements `Command` interface (`Name()`, `Description()`, `RequiredPermission()`, `Execute(ctx *CommandContext)`)
- **services/** - Business logic layer. Key services:
  - `DatabaseService` - PostgreSQL with **local mode** fallback (when `DB_HOST=""`, uses in-memory state). Always check `db.LocalMode()` before DB operations
  - `AgonesService` - Kubernetes GameServer lifecycle (Fleet allocation, status sync, K8s UID reconciliation)
  - `PricingService` - Database-driven game costs with 5-min cache (BLOCKER 1: never hardcode prices)
  - `GuildTreasuryService` - Shared wallet system (Blue Ocean strategy for Titan-tier servers)
  - `SubscriptionService` - Stripe webhook handling with idempotent processing (BLOCKER 8: zero-touch subscriptions)
  - `AdConversionService` - ayeT-Studios S2S callbacks with HMAC-SHA1 verification
  - `ConsentService` - GDPR compliance for ad monetization
  - `CleanupService` - Background cron for billing and stale resource pruning
- **payment/** - Stripe integration (checkout sessions, webhook signature verification, WTG coin packages)
- **agones/** - Low-level Agones SDK client for Fleet/GameServer operations
- **http/** - HTTP server (port 9090) exposing `/health`, `/healthz`, `/ready`, `/metrics` (Prometheus), `/stripe/webhook`, `/ads/ayet/callback`, WordPress dashboard API
- **config/** - Environment loading via godotenv (`.env` for local dev)
- **version/** - Build metadata injected via ldflags (`Version`, `GitCommit`, `BuildDate`)

### Data Flow

1. **Server Creation**: Discord command → Validate credits via `PricingService` → Allocate from Agones Fleet → Store `kubernetes_uid` in DB → Notify user (DM or channel)
2. **Payment**: User buys WTG → Stripe checkout → Webhook with signature verification → Idempotent credit add (via `stripe_payment_intent_id` unique constraint) → Discord role sync
3. **Billing**: Hourly cleanup cron queries Agones status → Charges running servers → Marks stopped servers for 7-day cleanup
4. **Ad Conversion**: User completes offer → ayeT S2S callback → HMAC verification → Dedup via `conversionId` → Award credits → Metrics

## Development Workflows

### Local Development
```bash
# 1. Install Go 1.23+ tooling
# 2. Copy environment file
cp .env.example .env
# Edit .env: Set DISCORD_TOKEN (required), DB_HOST="" for local mode

# 3. Install dependencies
go mod download

# 4. Run bot (local mode = no DB/K8s required)
go run main.go
# Or use cmd/ entrypoint:
go run ./cmd

# HTTP server starts on :9090 with /health, /metrics endpoints
```

### Building
```bash
# Standard build
go build -o agis-bot .
# Or build to bin/
mkdir -p bin && go build -trimpath -o bin/agis-bot ./cmd

# Production build (with version injection - matches Dockerfile)
go build -ldflags="-X agis-bot/internal/version.Version=v1.7.0 \
  -X agis-bot/internal/version.GitCommit=$(git rev-parse --short HEAD) \
  -X agis-bot/internal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o agis-bot .

# Docker build (multi-stage with version args)
docker build \
  --build-arg VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo dev) \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -t ghcr.io/wethegamers/agis-bot:dev .
```

### Testing
```bash
# Unit tests (services have _test.go with mocks)
go test ./internal/services/...

# Integration tests (require PostgreSQL, use -tags=integration)
go test -tags=integration ./internal/services/...

# Lint (no golangci-lint configured, use standard tools)
go vet ./...
gofmt -s -l . # Should return nothing

# Smoke test against running instance
./scripts/test-agis-bot.sh  # Expects localhost:9090/healthz
```

### CI/CD Pipeline
**GitHub Actions** (`.github/workflows/main.yaml` named "release") on push to `main`:
1. Skip duplicate runs (fkirc/skip-duplicate-actions)
2. Submit **Publish** workflow (`.argo/publish.yaml`) via Argo CLI on self-hosted runner:
   - Kaniko builds container → `ghcr.io/wethegamers/agis-bot:<shortSha>`
   - Helm chart version bump (`.argo/release.yaml`)
3. Submit **Deploy** workflow (`.argo/deploy.yaml`) to environments: `development` → `staging` → `production`
4. Discord webhook notifications (success/failure embeds with commit info)

**Argo Workflow Templates** (`.argo/*.yaml`) expect parameters: `appName`, `branch`, `containerRegistryURL`, `gitUrlNoProtocol`, `shortSha`, `chartDir`, `clusterName`

## Critical Patterns

### Command Context
All commands receive `CommandContext` struct with:
- `Session` - discordgo session
- `Message` - MessageCreate event
- `Args` - Parsed command arguments
- `DB` - DatabaseService
- `Config` - Config struct
- `Permissions` - PermissionChecker
- `PricingService` - Dynamic pricing (BLOCKER 1)
- `EnhancedServer` - Server lifecycle service
- `Notifications` - Discord notification service
- `Agones` - GameServer management (may be nil)

Access services: `ctx.PricingService.GetPricing(gameType)`, `ctx.DB.GetUser(discordID)`. Always check for nil services.

### Service Initialization
Services use `New*Service()` constructors in `main.go`. **Handle nil gracefully** - `AgonesService`, `PricingService` may fail initialization and be nil. Check before use:
```go
if ctx.AgonesService == nil {
    return fmt.Errorf("Agones not available")
}
```

See `main.go` for initialization order (config → DB → logging → Agones → pricing → enhanced server → subscriptions → cleanup).

### Database Patterns
- **No ORM** - Raw SQL with `database/sql` package
- **Migrations**: `internal/database/migrations/*.sql` (numbered, e.g., `005_guild_treasury.sql`)
- **Always use prepared statements** to prevent SQL injection
- **Local mode**: When `DB_HOST=""`, `DatabaseService` uses in-memory maps (`localUsers`, `localConversions`). Check `db.LocalMode()` before DB operations
- **Tables**: `users`, `game_servers`, `guild_treasury`, `guild_members`, `server_reviews`, `user_ad_consent`, `ad_conversions`, `payment_transactions`, `subscriptions`

### Agones Integration
- **Fleets**: `free-tier-fleet.yaml`, `premium-tier-fleet.yaml` define GameServer pools
- **Namespace**: From `AGONES_NAMESPACE` env (default: `agones-dev`). **CRITICAL**: Use `agones-system` for production per namespace conventions
- **Allocation Flow**: `create` command → `AgonesService.AllocateGameServer()` → Store `kubernetes_uid` in DB → Sync status via `LastStatusSync`
- **Reconciliation**: K8s UID links DB record to GameServer resource. Status sync updates `AgonesStatus` field
- **RBAC**: Bot needs permissions on GameServers, Fleets, GameServerAllocations in target namespace

### Pricing System (BLOCKER 1 - Critical)
**Database-driven** costs in `game_pricing` table. **NEVER hardcode prices**.
- Query via `PricingService.GetPricing(gameType)` which returns `GamePricing` struct
- Cache refreshes every 5 minutes (`syncPricing()`)
- Seed data: Minecraft 5gc/hr, CS2 8gc/hr, Terraria 3gc/hr, GMod 6gc/hr (28-39% margins)
- New game types: Add to `game_pricing` table, not code
- See `internal/services/pricing.go`

### Stripe Webhooks (Security-Critical)
**Signature verification is MANDATORY**:
```go
event, err := webhook.ConstructEvent(payload, signature, webhookSecret)
```
- **Idempotency**: Use `payment_transactions.stripe_payment_intent_id` unique constraint to prevent duplicate credits
- **Handlers**: `internal/http/server.go` processes `checkout.session.completed`, `invoice.payment_succeeded`
- **Testing**: Use Stripe CLI for local webhook forwarding (`stripe listen --forward-to localhost:9090/stripe/webhook`)

### Ad Conversion Security (ayeT-Studios)
**HMAC-SHA1 verification required**:
```go
// Signature = HMAC-SHA1(externalIdentifier + currency + conversionId, AYET_API_KEY)
```
- Endpoint: `/ads/ayet/callback`
- **Deduplication**: `ad_conversions.conversion_id` unique constraint
- **Fraud detection**: Track velocity, IP changes, excessive earnings per `AdConversionService`
- See `internal/services/ad_conversion.go`, `internal/http/ayet_handler.go`

### Permissions & RBAC
Three levels in `internal/bot/permissions.go`:
- `PermissionUser` - All verified Discord members
- `PermissionModerator` - Role IDs in `MOD_ROLES` env var
- `PermissionAdmin` - Role IDs in `ADMIN_ROLES` env var

Check permissions: `ctx.Permissions.CheckPermission(userID, guildID, requiredPerm)`. Commands declare required level via `RequiredPermission()` method.

## Configuration & Secrets Management

### Environment Variables
Loaded via godotenv from `.env` (see `internal/config/config.go`). **Never commit secrets**.

**Required**:
- `DISCORD_TOKEN` - Bot token from Discord Developer Portal
- `DISCORD_CLIENT_ID` - Application ID
- `DISCORD_GUILD_ID` - Discord server ID

**Database** (empty `DB_HOST` enables local mode):
- `DB_HOST` - PostgreSQL host (e.g., `postgresql.database.svc.cluster.local`)
- `DB_NAME` - Database name (default: `agis`)
- `DB_USER` - Database user (default: `root`)
- `DB_PASSWORD` - Database password

**Kubernetes/Agones**:
- `AGONES_NAMESPACE` - K8s namespace for GameServers (default: `agones-dev`)
- `GITHUB_TOKEN` - **Required** for GitHub webhook integration (Argo deployment will fail without it)

**Stripe Payments**:
- `STRIPE_SECRET_KEY` - Stripe API key (`sk_live_...` or `sk_test_...`)
- `STRIPE_WEBHOOK_SECRET` - Webhook signature verification (`whsec_...`)
- `STRIPE_SUCCESS_URL` - Payment success redirect
- `STRIPE_CANCEL_URL` - Payment cancel redirect

**Ad Monetization** (ayeT-Studios):
- `AYET_API_KEY` - API key for HMAC-SHA1 verification
- `AYET_CALLBACK_TOKEN` - Shared secret for S2S callbacks
- `AYET_OFFERWALL_URL` - Offerwall embed URL
- `AYET_SURVEYWALL_URL` - Surveywall embed URL
- `AYET_VIDEO_PLACEMENT_ID` - Video ad placement ID

**Monitoring**:
- `METRICS_PORT` - Prometheus port (default: `9090`)
- `SENTRY_DSN` - Sentry.io error tracking
- `DISCORD_WEBHOOK_*` - Discord webhooks for alerts (PAYMENTS, ADS, INFRA, SECURITY, PERFORMANCE, REVENUE, CRITICAL, COMPLIANCE)

**Discord Logging Channels**:
- `LOG_CHANNEL_GENERAL`, `LOG_CHANNEL_USER`, `LOG_CHANNEL_MOD`, `LOG_CHANNEL_ERROR`, `LOG_CHANNEL_CLEANUP`, `LOG_CHANNEL_CLUSTER`, `LOG_CHANNEL_EXPORT`, `LOG_CHANNEL_AUDIT`

### Vault Secret Management
**Path Pattern**: `secret/<env>/agis-bot/<key>`

**CRITICAL NAMING STANDARD** (see `NAMING_STANDARDS.md`):
- Vault secrets **MUST** use `{DESCRIPTOR}_{ENV}` suffix (e.g., `DISCORD_TOKEN_DEV`, `DB_PASSWORD_PRO`)
- Environments: `DEV` (development), `STA` (staging), `PRO` (production)
- Kubernetes secrets **MAY** omit `_ENV` suffix when using namespace isolation
- Examples:
  ```
  secret/dev/agis-bot/DISCORD_TOKEN_DEV
  secret/sta/agis-bot/DISCORD_TOKEN_STA
  secret/pro/agis-bot/DISCORD_TOKEN_PRO
  ```

**Setup Script**: `scripts/vault-add-development-secrets.sh` (reference only, use placeholders)

**Port-Forward for Local Vault Access**:
```bash
kubectl port-forward -n vault svc/vault 8200:8200
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="<token>"
```

### ExternalSecrets Operator
`charts/agis-bot/templates/external-secrets.yaml` maps Vault to `agis-bot-secrets` Kubernetes secret. Deployment consumes via `valueFrom.secretKeyRef`. **When adding new env vars**, update both Vault and ExternalSecret manifest.

### Helm Values
`charts/agis-bot/values.yaml` defaults:
- `replicaCount: 1`
- `image.repository: ghcr.io/wethegamers/agis-bot`
- `service.port: 9090`
- `vaultMountPoint: secret`
- `vaultSecretPath: development/agis-bot` (change per environment)
- `clusterSecretStoreName: vault-kv-secret`
- Linkerd injection enabled via `deploymentAnnotations`

## Kubernetes Deployment

### Namespace Conventions (CRITICAL)
Follow `NAMESPACE_CONVENTIONS.md` and `NAMING_STANDARDS.md`:

**Infrastructure Services** use `-system` suffix:
- `agones-system` - Game server orchestration (follows Agones standard)
- `monitoring-system` - Prometheus, Grafana
- `gitops-system` - ArgoCD
- `security-system` - Vault, cert-manager

**Application Namespaces**: `<app>-<env>`
- `agis-bot-dev`, `agis-bot-sta`, `agis-bot-pro`

**Current State**: Deployments may use `development`, `staging`, `production` namespaces. Migrate to standardized names per `NAMING_STANDARDS.md`.

### Helm Chart Structure
```
charts/agis-bot/
├── Chart.yaml                    # Version v1.7.0, apiVersion v2
├── values.yaml                   # Defaults (replicaCount, image, resources)
├── templates/
│   ├── deployment.yaml           # Main bot deployment
│   ├── external-secrets.yaml     # Vault → K8s secret mapping
│   ├── servicemonitor.yaml       # Prometheus ServiceMonitor (monitoring.coreos.com/v1)
│   ├── ingress.yaml              # NGINX ingress (cert-manager TLS)
│   ├── service.yaml              # ClusterIP service (port 9090)
│   ├── serviceaccount.yaml       # K8s RBAC service account
│   ├── grafana-dashboard-cm.yaml # Grafana dashboard ConfigMap
│   └── tests/test-connection.yaml
```

### Resource Naming (Per NAMING_STANDARDS.md)
**Deployments/Services**: `{application}[-{component}]` (no env suffix when single env per cluster)
- Example: `agis-bot`, `agis-bot-api`

**ConfigMaps/Secrets**: `{application}-{purpose}-{env}`
- Example: `agis-bot-config-dev`, `agis-bot-secrets-pro`

**DNS/Ingress**: `{application}[-{component}].{env}.{domain}`
- Dev: `agis-bot.dev.wethegamers.org`
- Staging: `agis-bot.sta.wethegamers.org`
- Production: `agis-bot.wethegamers.org` (no env prefix) or `agis-bot.pro.wethegamers.org`

### Health & Observability
**HTTP Endpoints** (port 9090):
- `/health`, `/healthz` - Health check (returns `{"status":"ok"}`)
- `/ready`, `/readyz` - Readiness check
- `/info`, `/about`, `/version` - Build metadata
- `/metrics` - Prometheus metrics

**Prometheus Metrics** (registered in `main.go`):
- `agis_commands_total{command,user_id}` - Command execution counter
- `agis_game_servers_total{game_type,status}` - Server inventory gauge
- `agis_credits_transactions_total{transaction_type,user_id}` - Credit operations
- `agis_active_users_total` - Active user gauge
- `agis_ad_conversions_total{provider,type,status}` - Ad conversion tracking
- `agis_ad_rewards_total{provider,type}` - Credits awarded from ads
- `agis_ad_fraud_attempts_total{provider,reason}` - Fraud detection

**ServiceMonitor** scrapes on `/metrics`, labels: `app: agis-bot`, `release: prometheus-operator`

**Grafana Dashboard**: Auto-imported via ConfigMap with label `grafana_dashboard: "1"`

### Deployment Annotations
- `linkerd.io/inject: enabled` - Linkerd sidecar injection
- Keep sidecar compatibility in mind (e.g., startup probe timing)

## Common Pitfalls & Anti-Patterns

### Critical Mistakes (Will Break Production) 🔴
❌ **Bypass PricingService** → All costs MUST query `game_pricing` table (BLOCKER 1)
❌ **Skip webhook signature verification** → Security breach, idempotency failures
❌ **Hardcode secrets** → Use Vault `{KEY}_{ENV}` pattern (e.g., `DISCORD_TOKEN_DEV`)
❌ **Forget K8s UID** → Store `kubernetes_uid` on GameServer allocation for reconciliation
❌ **Commit secrets** → Never. Scripts use placeholders. Vault only.
❌ **Mutable tags** → No `:latest` in production. Use semantic versions or SHA tags

### Common Mistakes (Will Cause Issues) ⚠️
⚠️ **Ignore LocalMode** → Check `db.LocalMode()` before DB ops (in-memory fallback when `DB_HOST=""`)
⚠️ **Assume services exist** → `AgonesService`, `PricingService` may be nil. Check before use
⚠️ **Access globals** → Use `CommandContext` for all dependencies (session, DB, config, services)
⚠️ **Discord rate limits** → Batch operations, exponential backoff on 429
⚠️ **Missing GITHUB_TOKEN** → Argo deployments fail. Required in Vault for webhooks
⚠️ **Wrong Vault path** → Must be `secret/<env>/agis-bot/{KEY}_{ENV}`, not `development` (verbose)

### Decision Trees (Token-Efficient Workflows)

**Add New Command**:
```
1. Create struct: type XCommand struct{} with Command interface
2. Register: h.Register(&XCommand{}) in handler.go
3. Execute(ctx):
   - Parse ctx.Args
   - Check ctx.DB.LocalMode() if DB-dependent
   - Validate ctx.PricingService for game ops (check nil)
   - Use ctx.Agones for servers (check nil)
   - Return embed via ctx.Session
4. Test: DB_HOST="" for local mode
```

**Add Env Var**:
```
1. internal/config/config.go: Add field to struct
2. Vault: vault kv put secret/<env>/agis-bot {KEY}_{ENV}="value"
3. charts/agis-bot/templates/external-secrets.yaml: Add remoteRef
4. charts/agis-bot/templates/deployment.yaml: Add env with secretKeyRef
5. Access: ctx.Config.{Section}.{Key}
```

**Troubleshoot Flow**:
```
Bot down (P0) → kubectl get pods → logs --tail=50 → Discord status → Token
DB errors → nc -zv $DB_HOST 5432 → credentials → max_connections
Webhooks → Signature match → Endpoint reachable → Stripe dashboard
Agones → Fleet status → kubectl get gs | grep Ready → RBAC
Memory → kubectl top → Cache growth → pprof heap
```

## Token-Efficient Patterns

### Code Snippets (Copy-Paste Ready)

**Check Service Availability**:
```go
if ctx.PricingService == nil {
    return fmt.Errorf("pricing service unavailable")
}
if ctx.Agones == nil {
    return fmt.Errorf("Agones unavailable")
}
```

**Get/Create User**:
```go
user, err := ctx.DB.GetOrCreateUser(ctx.Message.Author.ID)
if err != nil {
    return fmt.Errorf("failed to get user: %v", err)
}
```

**Check Credits**:
```go
if user.Credits < costRequired {
    return c.showInsufficientCredits(ctx, user.Credits, costRequired)
}
```

**Verify Webhook Signature (Stripe)**:
```go
event, err := webhook.ConstructEvent(payload, signature, webhookSecret)
if err != nil {
    return fmt.Errorf("signature verification failed: %v", err)
}
```

**Verify HMAC (ayeT)**:
```go
message := params.ExternalIdentifier + params.Currency + params.ConversionID
mac := hmac.New(sha1.New, []byte(apiKey))
mac.Write([]byte(message))
expectedSig := hex.EncodeToString(mac.Sum(nil))
if !hmac.Equal([]byte(expectedSig), []byte(params.Signature)) {
    return ErrInvalidSignature
}
```

**Allocate GameServer**:
```go
gsInfo, err := ctx.Agones.AllocateGameServer(ctx.Context, gameType, serverName, userID)
if err != nil {
    return fmt.Errorf("allocation failed: %v", err)
}
// Store kubernetes_uid for reconciliation
server.KubernetesUID = gsInfo.UID
```

**Discord Embed Pattern**:
```go
embed := &discordgo.MessageEmbed{
    Title: "Title",
    Description: "Description",
    Color: 0x00ff00, // Green success
    Fields: []*discordgo.MessageEmbedField{
        {Name: "Field", Value: "Value", Inline: true},
    },
}
ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
```

### File Path Shortcuts

**Core** (hot path):
- `main.go` → Service init, metrics (481L)
- `i/b/c/handler.go` → Command routing (340L) [`i`=internal, `b`=bot, `c`=commands]
- `i/s/database.go` → DB ops, local mode (1145L)
- `i/s/pricing.go` → Dynamic pricing (239L) **[BLOCKER 1]**
- `i/s/agones.go` → GameServer lifecycle (343L)
- `i/http/server.go` → Webhooks, API (705L)

**Integration**:
- `i/payment/stripe.go` → Checkout, webhooks (223L)
- `i/s/ad_conversion.go` → ayeT S2S, HMAC (525L)
- `i/s/ab_testing.go` → Experiments (294L)
- `i/agones/client.go` → Agones SDK (284L)

**Config/Deploy**:
- `charts/agis-bot/` → Helm chart
- `.argo/{publish,deploy}.yaml` → CI/CD workflows
- `.github/workflows/main.yaml` → GH Actions
- `Dockerfile` → Multi-stage build

**Standards**:
- `BLACKBOX.md` → AI agent context
- `WARP.md` → Terminal commands
- `NAMING_STANDARDS.md` → **CRITICAL** naming rules
- `NAMESPACE_CONVENTIONS.md` → K8s namespaces

**Docs**:
- `docs/OPS_MANUAL.md` → Operations (1043L)
- `docs/QUICK_REFERENCE.md` → On-call (402L)
- `docs/VAULT_SECRETS_SETUP.md` → Secrets (300L)
- `docs/AGONES_INTEGRATION.md` → Agones setup (237L)

## Key Files for Reference

**Notation**: `L` = lines, `i` = internal/, `b` = bot/, `c` = commands/, `s` = services/

**Core Application**:
- `main.go` - Entrypoint, service init, metrics registration, HTTP server (481L)
- `cmd/main.go` - Alternative entrypoint (routes to root main.go)
- `i/b/c/handler.go` - Command registration and routing (340L)
- `i/s/database.go` - DB ops with local mode (1145L)
- `i/s/agones.go` - GameServer lifecycle (343L)
- `i/config/config.go` - Env loading, typed config
- `i/version/version.go` - Build metadata (ldflags)

**Integration Points**:
- `i/http/server.go` - HTTP handlers (705L: health, metrics, webhooks, API)
- `i/payment/stripe.go` - Stripe checkout, webhook processing (223L)
- `i/s/ad_conversion.go` - ayeT S2S callbacks, HMAC (525L)
- `i/agones/client.go` - Agones SDK operations (284L)

**Infrastructure**:
- `charts/agis-bot/` - Helm (Deployment, Service, Ingress, ExternalSecrets, RBAC)
- `.argo/publish.yaml` - Container build (Kaniko)
- `.argo/deploy.yaml` - Multi-env deployment
- `.github/workflows/main.yaml` - GH Actions release
- `Dockerfile` - Multi-stage build, version injection

**Standards & Conventions**:
- `BLACKBOX.md` - Comprehensive project context
- `WARP.md` - Terminal commands, CI/CD
- `NAMING_STANDARDS.md` - **CRITICAL**: Resource/secret/DNS naming (v1.1.0)
- `NAMESPACE_CONVENTIONS.md` - K8s namespace standards (`-system` for infra)

## Documentation Locations

**Operations**:
- `docs/OPS_MANUAL.md` - Complete O&M guide (1043 lines: architecture, deployment, DB, monitoring, backups, security, incidents)
- `docs/QUICK_REFERENCE.md` - Print-ready on-call card (402 lines: 30s health checks, emergency procedures)
- `docs/DEPLOYMENT_GUIDE_V2.md` - Production deployment steps (565 lines)
- `docs/VAULT_SECRETS_SETUP.md` - Vault secret configuration guide (300 lines)

**User Documentation**:
- `README.md` - Project overview, features, quick start
- `COMMANDS.md` - Discord command reference (219 lines)
- `docs/USER_GUIDE.md` - Complete user guide (591 lines with examples)

**Integration Guides**:
- `docs/AGONES_INTEGRATION.md` - Agones setup, RBAC, Vault secrets
- `docs/WORDPRESS_INTEGRATION.md` - WordPress dashboard API integration
- `docs/webhook-setup/` - GitHub/Discord webhook configuration

**Development**:
- `docs/BLOCKER_1_COMPLETED.md` through `docs/BLOCKER_8_COMPLETED.md` - Project milestones
- `docs/INTEGRATION_TESTS.md` - Integration test strategy
- `docs/GRAFANA_SETUP.md` - Grafana dashboard configuration
- `docs/SENTRY_SETUP_GUIDE.md` - Sentry error monitoring setup

## Development Conventions

**Go Style**:
- Idiomatic Go with standard formatting (`gofmt`, `go vet`)
- Modules under `internal/` with clear package boundaries
- No ORM - raw SQL with prepared statements
- Error wrapping: `fmt.Errorf("context: %w", err)`

**Adding New Features**:
1. **Config**: Add to `internal/config/config.go` struct
2. **Vault**: Add secret to `secret/<env>/agis-bot/{KEY}_{ENV}` (uppercase, env suffix)
3. **ExternalSecret**: Update `charts/agis-bot/templates/external-secrets.yaml`
4. **Deployment**: Update `charts/agis-bot/templates/deployment.yaml` env vars
5. **Documentation**: Update relevant docs (OPS_MANUAL.md, USER_GUIDE.md)

**Kubernetes Resources**:
- Update Helm templates in `charts/agis-bot/templates/`
- Follow naming standards: `{application}-{purpose}-{env}`
- Keep Argo/GitHub workflow parameters in sync with Helm values
- Test locally with `helm template` before committing

**CI/CD Changes**:
- Validate Argo Workflow syntax before merging
- Test workflow submission: `argo-linux-amd64 submit .argo/publish.yaml --wait --log`
- Ensure Discord webhook notifications are configured

**Never**:
- Hardcode prices (use PricingService)
- Commit secrets (use Vault + ExternalSecrets)
- Skip webhook signature verification (security-critical)
- Use mutable image tags like `:latest` in production

---

## A/B Testing & Experimentation

### A/B Testing Service (`internal/services/ab_testing.go`)

**Purpose**: Test reward multipliers, pricing strategies, and feature variants to optimize engagement and revenue.

**Core Concepts**:
- **Experiments**: Time-bound tests with 2+ variants
- **Traffic Allocation**: Percentage of users entering experiment (e.g., 20% = 0.2)
- **Variant Allocation**: Distribution within experiment (e.g., 50/50 split)
- **Deterministic Assignment**: User always gets same variant (hash-based)
- **Sticky Sessions**: Users don't switch variants mid-experiment

**Creating an Experiment**:
```go
experiment := &services.ExperimentConfig{
    ID:           "reward_test_001",
    Name:         "Ad Reward Multiplier Test",
    Description:  "Test 1.5x vs 2.0x multipliers",
    StartDate:    time.Now(),
    EndDate:      time.Now().Add(7 * 24 * time.Hour), // 7 days
    TrafficAlloc: 0.20, // 20% of users
    TargetMetric: "conversion_rate",
    Status:       "draft",
    Variants: []services.Variant{
        {
            ID:          "control",
            Name:        "Control Group",
            Allocation:  0.5, // 50% of experiment traffic
            Config:      map[string]interface{}{"multiplier": 1.5},
            Description: "Current 1.5x multiplier",
        },
        {
            ID:          "variant_a",
            Name:        "High Reward",
            Allocation:  0.5,
            Config:      map[string]interface{}{"multiplier": 2.0},
            Description: "Test 2.0x multiplier",
        },
    },
}

abService.CreateExperiment(experiment)
abService.UpdateExperimentStatus(experiment.ID, "running")
```

**Getting User's Variant**:
```go
variant, err := abService.GetVariant(userID, "reward_test_001")
if err != nil {
    // Experiment not running or user not eligible
    return defaultBehavior()
}
if variant == nil {
    // User not in experiment (outside traffic allocation)
    return defaultBehavior()
}

// Apply variant config
multiplier := variant.Config["multiplier"].(float64)
rewardAmount := baseReward * multiplier
```

**Recording Metrics**:
```go
// Record conversion event
abService.RecordEvent(userID, experimentID, "conversion", 1.0)

// Record revenue event
abService.RecordEvent(userID, experimentID, "revenue", revenueAmount)

// Record fraud detection
abService.RecordEvent(userID, experimentID, "fraud", 1.0)
```

**Analysis**: Use `GetExperimentResults(experimentID)` to retrieve aggregated metrics per variant (conversion rate, revenue per user, sample size, fraud rate).

---

## Ad Conversion Workflows

### ayeT-Studios Integration (`internal/services/ad_conversion.go`)

**Flow Overview**:
1. User completes offer on ayeT platform (Offerwall/Surveywall/Video)
2. ayeT sends S2S callback to `/ads/ayet/callback`
3. Bot verifies HMAC-SHA1 signature
4. Check for duplicate via `conversion_id` unique constraint
5. Verify user consent (GDPR compliance)
6. Apply fraud detection rules
7. Award credits with premium multiplier if applicable
8. Record metrics

**Signature Verification (CRITICAL)**:
```go
// ayeT signature = HMAC-SHA1(externalIdentifier + currency + conversionId, AYET_API_KEY)
func verifyAyetSignature(params AyetCallbackParams, apiKey string) error {
    message := params.ExternalIdentifier + params.Currency + params.ConversionID
    mac := hmac.New(sha1.New, []byte(apiKey))
    mac.Write([]byte(message))
    expectedSig := hex.EncodeToString(mac.Sum(nil))
    
    if !hmac.Equal([]byte(expectedSig), []byte(params.Signature)) {
        return ErrInvalidSignature
    }
    return nil
}
```

**Fraud Detection Rules**:
- **Excessive Velocity**: >10 conversions in 1 hour
- **IP Hopping**: >3 different IPs in 24 hours
- **Excessive Earnings**: >500 GC in 24 hours (configurable threshold)
- **Duplicate Conversion**: Same `conversion_id` (DB unique constraint)

**Reward Algorithm** (`internal/services/reward_algorithm.go`):
```go
// Base reward from provider's currency conversion
baseReward := params.Amount * conversionRate

// Apply premium multiplier (3x for premium users)
multiplier := 1.0
if user.IsPremium() {
    multiplier = 3.0
}

// Apply A/B test variant if in experiment
variant, _ := abService.GetVariant(userID, "active_reward_experiment")
if variant != nil {
    multiplier = variant.Config["multiplier"].(float64)
}

finalReward := int(float64(baseReward) * multiplier)
```

**Endpoints**:
- `POST /ads/ayet/callback` - S2S conversion callback (HMAC verified)
- Params: `externalIdentifier` (Discord ID), `currency`, `amount`, `conversionId`, `signature`, `custom_1..4`

**Database Schema** (`ad_conversions` table):
```sql
CREATE TABLE ad_conversions (
    id SERIAL PRIMARY KEY,
    discord_id VARCHAR(32) NOT NULL,
    conversion_id VARCHAR(255) NOT NULL UNIQUE, -- Idempotency
    provider VARCHAR(50) NOT NULL,              -- "ayet"
    type VARCHAR(50) NOT NULL,                  -- "offerwall", "surveywall", "video"
    amount INTEGER NOT NULL,                    -- Game Credits awarded
    multiplier DECIMAL(3,2) DEFAULT 1.0,        -- Applied multiplier
    status VARCHAR(20) DEFAULT 'pending',       -- "pending", "completed", "fraud"
    fraud_reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**GDPR Compliance**:
- User must give consent via `/consent agree` command before viewing ads
- Consent stored in `user_ad_consent` table with timestamp, IP country, version
- EU users (based on IP) require explicit consent
- Users can withdraw consent via `/consent withdraw` (all ad features disabled)

---

## Command Implementation Examples

### Standard Command Pattern

All commands implement the `Command` interface:

```go
type Command interface {
    Name() string
    Description() string
    RequiredPermission() bot.Permission
    Execute(ctx *CommandContext) error
}
```

### Example 1: Simple User Command (Daily Credits)

```go
// internal/bot/commands/daily.go
type DailyCommand struct{}

func (c *DailyCommand) Name() string {
    return "daily"
}

func (c *DailyCommand) Description() string {
    return "Claim daily bonus credits (24h cooldown)"
}

func (c *DailyCommand) RequiredPermission() bot.Permission {
    return bot.PermissionUser
}

func (c *DailyCommand) Execute(ctx *CommandContext) error {
    user, err := ctx.DB.GetOrCreateUser(ctx.Message.Author.ID)
    if err != nil {
        return fmt.Errorf("failed to get user: %v", err)
    }
    
    // Check cooldown (24 hours)
    if time.Since(user.LastDaily) < 24*time.Hour {
        nextDaily := user.LastDaily.Add(24 * time.Hour)
        embed := &discordgo.MessageEmbed{
            Title: "⏰ Daily Bonus On Cooldown",
            Description: fmt.Sprintf("Come back <t:%d:R>", nextDaily.Unix()),
            Color: 0xffaa00,
        }
        _, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
        return err
    }
    
    // Award credits
    dailyAmount := 10
    if err := ctx.DB.AddCredits(user.DiscordID, dailyAmount, "daily_bonus"); err != nil {
        return fmt.Errorf("failed to add credits: %v", err)
    }
    
    // Update cooldown
    if err := ctx.DB.UpdateLastDaily(user.DiscordID); err != nil {
        return err
    }
    
    embed := &discordgo.MessageEmbed{
        Title: "🎁 Daily Bonus Claimed!",
        Description: fmt.Sprintf("You earned **%d credits**", dailyAmount),
        Color: 0x00ff00,
        Fields: []*discordgo.MessageEmbedField{
            {Name: "New Balance", Value: fmt.Sprintf("%d credits", user.Credits+dailyAmount), Inline: true},
            {Name: "Next Daily", Value: "<t:" + fmt.Sprint(time.Now().Add(24*time.Hour).Unix()) + ":R>", Inline: true},
        },
    }
    _, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
    return err
}
```

### Example 2: Server Management Command (Create Server)

```go
// internal/bot/commands/server_management.go
func (c *CreateServerCommand) Execute(ctx *CommandContext) error {
    // 1. Parse arguments
    if len(ctx.Args) == 0 {
        return c.showUsage(ctx)
    }
    
    gameType := strings.ToLower(ctx.Args[0])
    serverName := fmt.Sprintf("%s-%s", gameType, ctx.Message.Author.Username)
    if len(ctx.Args) > 1 {
        serverName = ctx.Args[1]
    }
    
    // 2. Validate game type via PricingService (BLOCKER 1)
    if ctx.PricingService == nil {
        return fmt.Errorf("pricing service not available")
    }
    
    pricing, err := ctx.PricingService.GetPricing(gameType)
    if err != nil {
        return c.showAvailableGames(ctx, gameType)
    }
    
    // 3. Check user credits
    user, err := ctx.DB.GetOrCreateUser(ctx.Message.Author.ID)
    if err != nil {
        return fmt.Errorf("failed to get user: %v", err)
    }
    
    if user.Credits < pricing.CostPerHour {
        return c.showInsufficientCredits(ctx, user.Credits, pricing.CostPerHour, gameType)
    }
    
    // 4. Allocate GameServer from Agones Fleet
    if ctx.Agones == nil {
        return fmt.Errorf("Agones not available - contact administrator")
    }
    
    gsInfo, err := ctx.Agones.AllocateGameServer(ctx.Context, gameType, serverName, user.DiscordID)
    if err != nil {
        return fmt.Errorf("failed to allocate server: %v", err)
    }
    
    // 5. Create DB record with K8s UID
    server := &services.GameServer{
        DiscordID:     user.DiscordID,
        Name:          serverName,
        GameType:      gameType,
        Status:        "starting",
        Address:       gsInfo.Address,
        Port:          int(gsInfo.Port),
        CostPerHour:   pricing.CostPerHour,
        KubernetesUID: gsInfo.UID,
        AgonesStatus:  string(gsInfo.Status),
    }
    
    if err := ctx.DB.CreateServer(server); err != nil {
        // Cleanup: Delete allocated GameServer
        ctx.Agones.DeleteGameServer(ctx.Context, gsInfo.Name)
        return fmt.Errorf("failed to create server record: %v", err)
    }
    
    // 6. Send notification (DM or channel based on --here flag)
    notifyChannel := ctx.Message.Author.ID // DM by default
    if notifyInChannel {
        notifyChannel = ctx.Message.ChannelID
    }
    
    embed := &discordgo.MessageEmbed{
        Title: "🚀 Server Deploying",
        Description: fmt.Sprintf("**%s** is being deployed", serverName),
        Color: 0x00ccff,
        Fields: []*discordgo.MessageEmbedField{
            {Name: "Game Type", Value: pricing.DisplayName, Inline: true},
            {Name: "Cost", Value: fmt.Sprintf("%d GC/hour", pricing.CostPerHour), Inline: true},
            {Name: "Status", Value: "⏳ Starting", Inline: false},
            {Name: "Estimated Time", Value: "2-5 minutes", Inline: true},
        },
    }
    
    if notifyInChannel {
        ctx.Session.ChannelMessageSendEmbed(notifyChannel, embed)
    } else {
        userChannel, _ := ctx.Session.UserChannelCreate(notifyChannel)
        ctx.Session.ChannelMessageSendEmbed(userChannel.ID, embed)
    }
    
    return nil
}
```

### Example 3: Admin Command (Mod Control)

```go
// internal/bot/commands/mod.go
type ModControlCommand struct{}

func (c *ModControlCommand) RequiredPermission() bot.Permission {
    return bot.PermissionModerator
}

func (c *ModControlCommand) Execute(ctx *CommandContext) error {
    // Permission check already done by handler
    // Moderators can control ANY server, not just their own
    
    if len(ctx.Args) < 2 {
        return fmt.Errorf("usage: modcontrol <action> <server_id>")
    }
    
    action := strings.ToLower(ctx.Args[0])
    serverID := ctx.Args[1]
    
    // Get server (don't filter by owner)
    server, err := ctx.DB.GetServerByID(serverID)
    if err != nil {
        return fmt.Errorf("server not found")
    }
    
    switch action {
    case "stop":
        return c.stopServer(ctx, server)
    case "restart":
        return c.restartServer(ctx, server)
    case "delete":
        return c.deleteServer(ctx, server)
    default:
        return fmt.Errorf("unknown action: %s", action)
    }
}
```

**Command Registration** (`internal/bot/commands/handler.go`):
```go
func (h *CommandHandler) registerCommands() {
    // User commands
    h.Register(&DailyCommand{})
    h.Register(&CreateServerCommand{})
    
    // Mod commands
    h.Register(&ModControlCommand{})
    
    // Admin commands  
    h.Register(&AdminStatusCommand{})
}
```

---

## Database Schema & Migrations

### Schema Overview

**Core Tables**:

1. **users** - Discord user profiles
```sql
CREATE TABLE users (
    discord_id VARCHAR(32) PRIMARY KEY,
    credits INTEGER DEFAULT 0,
    tier VARCHAR(20) DEFAULT 'free',
    last_daily TIMESTAMP,
    last_work TIMESTAMP,
    servers_used INTEGER DEFAULT 0,
    join_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

2. **game_servers** - Active and historical servers
```sql
CREATE TABLE game_servers (
    id SERIAL PRIMARY KEY,
    discord_id VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    game_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    address VARCHAR(255),
    port INTEGER,
    cost_per_hour INTEGER NOT NULL,
    kubernetes_uid VARCHAR(255),        -- K8s UID for reconciliation
    agones_status VARCHAR(50),          -- Agones GameServer status
    last_status_sync TIMESTAMP,         -- Last K8s status sync
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    stopped_at TIMESTAMP,
    cleanup_at TIMESTAMP,               -- 7 days after stopped
    FOREIGN KEY (discord_id) REFERENCES users(discord_id)
);
CREATE INDEX idx_game_servers_discord_id ON game_servers(discord_id);
CREATE INDEX idx_game_servers_status ON game_servers(status);
CREATE INDEX idx_game_servers_kubernetes_uid ON game_servers(kubernetes_uid);
```

3. **guild_treasury** - Shared guild wallets (BLOCKER 4)
```sql
CREATE TABLE guild_treasury (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) UNIQUE NOT NULL,
    guild_name VARCHAR(255) NOT NULL,
    owner_id VARCHAR(255) NOT NULL,
    balance INTEGER DEFAULT 0 CHECK (balance >= 0),  -- Non-refundable
    total_deposits INTEGER DEFAULT 0,
    total_spent INTEGER DEFAULT 0,
    member_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(discord_id) ON DELETE CASCADE
);
```

4. **guild_members** - Guild membership tracking
```sql
CREATE TABLE guild_members (
    guild_id VARCHAR(255) NOT NULL,
    discord_id VARCHAR(255) NOT NULL,
    total_deposits INTEGER DEFAULT 0,
    role VARCHAR(20) DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member')),
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (guild_id, discord_id),
    FOREIGN KEY (guild_id) REFERENCES guild_treasury(guild_id) ON DELETE CASCADE,
    FOREIGN KEY (discord_id) REFERENCES users(discord_id) ON DELETE CASCADE
);
```

5. **server_reviews** - Community ratings (BLOCKER 5)
```sql
CREATE TABLE server_reviews (
    id SERIAL PRIMARY KEY,
    server_id INTEGER NOT NULL,
    reviewer_id VARCHAR(255) NOT NULL,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT NOT NULL CHECK (LENGTH(comment) <= 500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (server_id, reviewer_id),  -- One review per user per server
    FOREIGN KEY (server_id) REFERENCES game_servers(id) ON DELETE CASCADE,
    FOREIGN KEY (reviewer_id) REFERENCES users(discord_id) ON DELETE CASCADE
);
```

6. **user_ad_consent** - GDPR compliance (BLOCKER 7)
```sql
CREATE TABLE user_ad_consent (
    user_id BIGINT PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    consented BOOLEAN NOT NULL DEFAULT FALSE,
    consent_timestamp TIMESTAMPTZ,
    withdrawn_timestamp TIMESTAMPTZ,
    ip_country VARCHAR(2),              -- ISO 3166-1 alpha-2
    gdpr_version VARCHAR(20) NOT NULL DEFAULT 'v1.0',
    consent_method VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

7. **ad_conversions** - Ad monetization tracking
```sql
CREATE TABLE ad_conversions (
    id SERIAL PRIMARY KEY,
    discord_id VARCHAR(32) NOT NULL,
    conversion_id VARCHAR(255) NOT NULL UNIQUE,  -- Idempotency key
    provider VARCHAR(50) NOT NULL,               -- "ayet"
    type VARCHAR(50) NOT NULL,                   -- "offerwall", "surveywall", "video"
    amount INTEGER NOT NULL,                     -- Game Credits awarded
    multiplier DECIMAL(3,2) DEFAULT 1.0,         -- Applied multiplier
    status VARCHAR(20) DEFAULT 'pending',
    fraud_reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (discord_id) REFERENCES users(discord_id)
);
CREATE INDEX idx_ad_conversions_conversion_id ON ad_conversions(conversion_id);
```

8. **payment_transactions** - Stripe payments (BLOCKER 8)
```sql
CREATE TABLE payment_transactions (
    id SERIAL PRIMARY KEY,
    discord_id VARCHAR(32) NOT NULL,
    stripe_payment_intent_id VARCHAR(255) UNIQUE,  -- Idempotency
    amount_cents INTEGER NOT NULL,
    wtg_coins INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (discord_id) REFERENCES users(discord_id)
);
```

9. **subscriptions** - Premium subscriptions
```sql
CREATE TABLE subscriptions (
    id SERIAL PRIMARY KEY,
    discord_id VARCHAR(32) UNIQUE NOT NULL,
    stripe_subscription_id VARCHAR(255) UNIQUE,
    status VARCHAR(20) NOT NULL,
    current_period_start TIMESTAMP,
    current_period_end TIMESTAMP,
    cancel_at_period_end BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (discord_id) REFERENCES users(discord_id)
);
```

### Migration Pattern

Migrations live in `internal/database/migrations/` as numbered SQL files:
- `005_guild_treasury.sql` - Guild treasury system
- `006_server_reviews.sql` - Server reviews
- `007_gdpr_ad_consent.sql` - GDPR consent tracking

**Creating a Migration**:
1. Create `00X_feature_name.sql`
2. Include CREATE TABLE, indexes, foreign keys
3. Add comments for documentation
4. Test locally with `psql -f migration.sql`
5. Apply via `internal/services/database.go` or manual `psql` execution

**Migration Safety**:
- Use `IF NOT EXISTS` for idempotency
- Never drop columns in production (add nullable columns instead)
- Create indexes with `CONCURRENTLY` in production
- Test rollback strategy before applying

---

## Troubleshooting Runbooks

### Bot Not Responding

**Symptoms**: Commands don't work, bot shows offline in Discord

**Quick Diagnosis**:
```bash
# 1. Check pod status
kubectl get pods -n production -l app=agis-bot

# 2. Check recent logs
kubectl logs -n production -l app=agis-bot --tail=50

# 3. Check Discord API status
curl https://status.discord.com/api/v2/status.json
```

**Common Causes & Solutions**:

| Cause | Diagnosis | Solution |
|-------|-----------|----------|
| Invalid Discord token | Logs show "authentication failed" | Verify `DISCORD_TOKEN` in Vault matches Discord portal |
| Pod crash loop | Pod status `CrashLoopBackOff` | Check logs for startup errors, verify secrets exist |
| Network policy | Logs show "connection refused" | Verify network policy allows Discord API egress |
| Rate limited | Logs show "429 Too Many Requests" | Wait for rate limit reset, review command usage |

### Database Connection Failures

**Symptoms**: "failed to connect to database" errors

**Quick Diagnosis**:
```bash
# 1. Test connectivity from bot pod
kubectl exec -it -n production deploy/agis-bot -- /bin/sh
nc -zv $DB_HOST 5432

# 2. Check PostgreSQL status
kubectl get pods -n production -l app=postgresql

# 3. Verify credentials
kubectl get secret agis-bot-secrets -n production -o jsonpath='{.data.DB_PASSWORD}' | base64 -d
```

**Common Causes & Solutions**:

| Cause | Diagnosis | Solution |
|-------|-----------|----------|
| Wrong credentials | "authentication failed for user" | Update Vault secrets, restart pod |
| PostgreSQL down | Pod not running | Check PostgreSQL logs, restart if needed |
| Max connections reached | "too many connections" | Increase `max_connections` in PostgreSQL config |
| Network segmentation | Connection timeout | Verify network policy allows `agis-bot` → `postgresql` |

### Stripe Webhook Failures

**Symptoms**: Payments processed but users don't receive credits

**Quick Diagnosis**:
```bash
# 1. Check webhook logs
kubectl logs -n production -l app=agis-bot | grep "stripe webhook"

# 2. Check Stripe dashboard
# Go to Stripe Dashboard > Developers > Webhooks > Recent Deliveries
# Look for failed deliveries

# 3. Verify webhook secret
kubectl get secret agis-bot-secrets -n production -o jsonpath='{.data.STRIPE_WEBHOOK_SECRET}' | base64 -d
```

**Common Causes & Solutions**:

| Cause | Diagnosis | Solution |
|-------|-----------|----------|
| Signature mismatch | Logs show "invalid signature" | Update `STRIPE_WEBHOOK_SECRET` to match Stripe dashboard |
| Webhook endpoint unreachable | Stripe shows 500/timeout errors | Check ingress configuration, verify TLS certificate |
| Idempotency collision | DB constraint violation | Normal behavior, verify transaction exists in DB |
| Missing user | "user not found" error | Ensure user exists before payment (create if needed) |

### Agones GameServer Allocation Failures

**Symptoms**: `create` command fails with "failed to allocate server"

**Quick Diagnosis**:
```bash
# 1. Check Fleet status
kubectl get fleet -n agones-system

# 2. Check available GameServers
kubectl get gameserver -n agones-system | grep Ready

# 3. Check Agones logs
kubectl logs -n agones-system -l app=agones-controller
```

**Common Causes & Solutions**:

| Cause | Diagnosis | Solution |
|-------|-----------|----------|
| No ready GameServers | Fleet shows 0 ready replicas | Scale fleet: `kubectl scale fleet <name> --replicas=5 -n agones-system` |
| RBAC permission denied | Logs show "forbidden: cannot allocate" | Verify `agis-bot` ServiceAccount has allocation permissions |
| Fleet not found | "fleet not found" error | Check fleet exists in correct namespace, verify `AGONES_NAMESPACE` env var |
| Resource limits | Node shows resource pressure | Add nodes to cluster or reduce GameServer resource requests |

### High Memory Usage / OOMKilled

**Symptoms**: Pods restarting, `OOMKilled` in pod status

**Quick Diagnosis**:
```bash
# 1. Check current memory usage
kubectl top pods -n production -l app=agis-bot

# 2. Check pod events
kubectl describe pod -n production <pod-name>

# 3. Get memory profile
kubectl exec -it -n production deploy/agis-bot -- curl http://localhost:9090/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

**Common Causes & Solutions**:

| Cause | Diagnosis | Solution |
|-------|-----------|----------|
| Memory limit too low | Memory usage at/near limit | Increase `resources.limits.memory` in Helm values |
| Cache growth | PricingService cache growing unbounded | Review cache TTL, implement size limits |
| Goroutine leak | High goroutine count in pprof | Review goroutine lifecycle, ensure cleanup |
| Large Discord events | Spike during high activity | Implement event batching, increase memory temporarily |

### Ad Conversion Signature Failures

**Symptoms**: Logs show "invalid signature" for ad callbacks

**Quick Diagnosis**:
```bash
# 1. Check recent callback logs
kubectl logs -n production -l app=agis-bot | grep "ayet callback"

# 2. Verify API key
kubectl get secret agis-bot-secrets -n production -o jsonpath='{.data.AYET_API_KEY}' | base64 -d

# 3. Test signature locally
echo -n "userID123coins50conversion-abc" | openssl dgst -sha1 -hmac "YOUR_API_KEY"
```

**Common Causes & Solutions**:

| Cause | Diagnosis | Solution |
|-------|-----------|----------|
| Wrong API key | Signature never matches | Verify `AYET_API_KEY` matches ayeT dashboard |
| String concatenation order | Signature sometimes fails | Ensure order: `externalIdentifier + currency + conversionId` |
| Encoding issues | Non-ASCII characters in params | URL-decode parameters before verification |
| Key rotation | Suddenly all callbacks fail | Update `AYET_API_KEY` after provider rotation |

### Emergency Procedures

**Bot Completely Down (P0)**:
1. Check Discord status: `https://status.discord.com`
2. Verify pod running: `kubectl get pods -n production -l app=agis-bot`
3. If crashed, check logs: `kubectl logs -n production -l app=agis-bot --previous`
4. Quick restart: `kubectl rollout restart deployment/agis-bot -n production`
5. If Vault issue: Port-forward and verify secrets manually
6. Escalate to on-call engineer if >5 min downtime

**Database Corruption (P0)**:
1. Identify affected tables via error logs
2. Stop write operations: Scale bot to 0 replicas
3. Restore from latest backup (see `docs/OPS_MANUAL.md` Backup & Recovery)
4. Verify data integrity: Check row counts, run sample queries
5. Scale bot back up, monitor for errors
6. Document incident in post-mortem

**Payment Issues (P1)**:
1. Pause new payments: Contact Stripe support to disable webhooks temporarily
2. Identify failed transactions: Query `payment_transactions` table for `status = 'failed'`
3. Manual credit adjustment: Use admin command or direct DB update
4. Verify webhook signature matches current secret
5. Re-enable webhooks, monitor for 1 hour
6. Notify affected users via Discord

---

## Ultra-Condensed Cheat Sheet (Copy This for LLM Context Windows)

```yaml
PROJECT: AGIS Bot - WTG Discord game server automation
STACK: Go 1.23 | discordgo | Agones | K8s | PostgreSQL | Stripe | Prometheus
MODULE: agis-bot
DEPLOY: Helm → K8s | CI/CD: GH Actions → Argo Workflows → ArgoCD

CRITICAL_FILES:
  - main.go (481L) - Init, metrics
  - internal/bot/commands/handler.go (340L) - Command routing
  - internal/services/database.go (1145L) - DB+LocalMode
  - internal/services/pricing.go (239L) - BLOCKER 1 pricing
  - internal/http/server.go (705L) - Webhooks

ARCHITECTURE:
  - Commands: Implement Command interface (Name, Desc, Perm, Execute)
  - Services: DatabaseService, AgonesService, PricingService, SubscriptionService
  - Data Flow: Discord → Validate → Agones Fleet → DB (w/ kubernetes_uid) → Notify

CRITICAL_RULES:
  - ❌ NEVER hardcode prices → Use PricingService.GetPricing()
  - ❌ NEVER skip webhook signature verification
  - ❌ NEVER commit secrets → Vault: secret/<env>/agis-bot/{KEY}_{ENV}
  - ✅ ALWAYS check db.LocalMode() before DB ops
  - ✅ ALWAYS check service != nil (Agones, Pricing may fail init)
  - ✅ ALWAYS store kubernetes_uid for GameServer reconciliation
  - ✅ ALWAYS use CommandContext (session, DB, config, services)

NAMING_STANDARDS:
  - Vault: secret/<env>/agis-bot/{DESCRIPTOR}_{ENV} (e.g., DISCORD_TOKEN_DEV)
  - Namespaces: <app>-<env> (agis-bot-dev) OR <service>-system (agones-system)
  - DNS: <app>.<env>.<domain> (agis-bot.dev.wethegamers.org)
  - Envs: DEV (development), STA (staging), PRO (production)

DB_TABLES:
  - users (discord_id, credits, tier, last_daily, last_work)
  - game_servers (id, discord_id, name, game_type, status, kubernetes_uid, agones_status)
  - guild_treasury (guild_id, balance, total_deposits, member_count)
  - server_reviews (server_id, reviewer_id, rating 1-5, comment)
  - user_ad_consent (user_id, consented, consent_timestamp, gdpr_version)
  - ad_conversions (discord_id, conversion_id UNIQUE, provider, amount, multiplier)
  - payment_transactions (stripe_payment_intent_id UNIQUE, amount_cents, wtg_coins)
  - subscriptions (stripe_subscription_id, status, current_period_end)

COMMON_TASKS:
  New Command: struct → Command interface → handler.Register() → Execute(ctx)
  New Env: config.go → Vault → ExternalSecret → Deployment.yaml → ctx.Config
  New Game: INSERT INTO game_pricing (game_type, cost_per_hour, display_name)
  Troubleshoot: kubectl get pods → logs --tail=50 → Check status endpoints

WEBHOOKS:
  - Stripe: webhook.ConstructEvent(payload, signature, secret)
  - ayeT: HMAC-SHA1(externalID + currency + conversionID, apiKey)
  - Idempotency: stripe_payment_intent_id UNIQUE, conversion_id UNIQUE

A/B_TESTING:
  - Create experiment: ExperimentConfig with variants
  - Assign: GetVariant(userID, experimentID) returns config map
  - Record: RecordEvent(userID, experimentID, eventType, value)

TROUBLESHOOT_PRIORITY:
  P0 Bot Down: pods → logs → Discord API → token
  P0 DB Down: nc -zv $DB_HOST 5432 → credentials → max_connections
  P1 Webhooks: signature → endpoint → Stripe dashboard
  P1 Agones: Fleet status → kubectl get gs | grep Ready → RBAC
  P2 Memory: kubectl top → cache growth → pprof heap

COMMANDS:
  # Local dev
  cp .env.example .env && DB_HOST="" go run main.go
  # Build with version
  go build -ldflags="-X agis-bot/internal/version.Version=v1.7.0"
  # Test
  go test ./internal/services/...
  # Deploy
  helm upgrade agis-bot charts/agis-bot -n production --set image.tag=v1.7.1
  # Debug
  kubectl logs -n production -l app=agis-bot --tail=100
  kubectl exec -it -n production deploy/agis-bot -- /bin/sh
  # Restart
  kubectl rollout restart deployment/agis-bot -n production

DOCS:
  - BLACKBOX.md - AI agent context
  - WARP.md - Terminal commands
  - NAMING_STANDARDS.md - Resource naming (CRITICAL)
  - docs/OPS_MANUAL.md - Operations (1043L)
  - docs/QUICK_REFERENCE.md - On-call (402L)
  - docs/USER_GUIDE.md - User documentation (591L)
```

**Token Optimization Tips**:
- Reference this cheat sheet in LLM system prompts
- Use file path shortcuts: `i/s/database.go` = `internal/services/database.go`
- Copy code snippets from "Token-Efficient Patterns" section
- Use decision trees for multi-step workflows
- Check anti-patterns (❌) before implementing

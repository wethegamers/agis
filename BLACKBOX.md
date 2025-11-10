# AGIS Bot – BLACKBOX Context

## Project Overview
AGIS Bot (Advanced Gaming Integration System) is the production Discord automation for the **WeTheGamers (WTG)** platform. Written in Go 1.23, it orchestrates game-server lifecycle management through **Agones**, maintains the player economy, automates payments via **Stripe**, and exposes operational telemetry over HTTP/Prometheus. The service is deployed to Kubernetes with a Helm chart (`charts/agis-bot`) and relies heavily on Vault-managed secrets surfaced through External Secrets.

## Technology Stack
- **Language / Runtime:** Go 1.23, module root `agis-bot`
- **Primary Libraries:** `discordgo`, `agones.dev/agones`, `prometheus/client_golang`, `sentry-go`, `stripe-go`, `k8s.io/client-go`
- **Persistence:** PostgreSQL (SQL access via `internal/services`), Minio/S3 for backups, Redis is not used
- **Infra Integrations:** HashiCorp Vault + External Secrets Operator, Argo Workflows, GitHub Actions, Linkerd sidecar, Stripe webhooks
- **Containerization:** Multi-stage Dockerfiles (`Dockerfile`, `webhook-proxy.Dockerfile`)

## Repository Layout Highlights
```
.argo/                      # Argo Workflow templates (publish/deploy/release)
.github/workflows/          # GitHub Actions pipeline (release workflow with Argo submits)
charts/agis-bot/            # Helm chart (Deployment, ExternalSecret, Service, Ingress, etc.)
cmd/, internal/, main.go    # Go application modules and entry point
scripts/                    # Operational scripts (Vault bootstrap, deployments, testing)
docs/                       # Extensive operator & setup documentation
deployments/                # Auxiliary components (GitHub Discord webhook proxy)
```

## Key Components & Services
- `main.go`: Boots configuration, telemetry, Kubernetes client, Discord command handler, Stripe integration, ad conversion service, consent management, cleanup cron, HTTP server (health/metrics/API endpoints).
- `internal/`: Modular packages for bot commands, services (database, logging, payments, metrics), HTTP handlers, configuration loading, and versioning.
- `charts/agis-bot/templates/external-secrets.yaml`: Maps Vault properties to Kubernetes secrets consumed by the Deployment.
- `scripts/vault-add-development-secrets.sh`: Sample script for populating `secret/development/agis-bot` in Vault. Use as reference—do **not** check in real secrets.

## Environment & Secret Management
- Secrets live in Vault under `secret/<environment>/agis-bot` (default mount `secret/`, path `development/agis-bot` per Helm values).
- Required keys include `DISCORD_TOKEN`, `DISCORD_CLIENT_ID`, `DISCORD_GUILD_ID`, DB credentials, Stripe configuration, ad network tokens, logging webhook URLs, verification secrets, and **`GITHUB_TOKEN`** for GitHub webhook integrations.
- For local development, copy `.env.example` to `.env`. Many values (Discord/Stripe) are mandatory for full functionality; fallback defaults exist for non-critical paths.
- External Secrets Operator pulls the Vault data into `agis-bot-secrets`, which the Deployment consumes through `valueFrom.secretKeyRef` entries. Ensure any new environment variable is added both to Vault and the ExternalSecret manifest.

## Local Development Workflow
1. Clone repository and install Go 1.23 tooling.
2. `cp .env.example .env` and fill required variables (Discord tokens, database DSN, etc.). For Vault-backed setups, port-forward to Vault (`kubectl port-forward -n vault svc/vault 8200:8200`) and run/update `scripts/vault-add-development-secrets.sh` with real values.
3. Provision PostgreSQL and apply migrations (see `internal/services/database.go` and related SQL helpers).
4. Launch the bot:
   ```bash
   go run main.go
   ```
   The HTTP server exposes health (`/healthz`, `/readyz`) and Prometheus metrics on port 9090.
5. For the GitHub webhook proxy, refer to `deployments/github-discord-webhook-proxy.py` and `webhook-proxy.Dockerfile`.

## Testing & Quality Gates
- **Unit/Integration Tests:**
  ```bash
  go test ./...
  ```
- **Smoke / Health Test:** `scripts/test-agis-bot.sh` expects the service running locally and checks `http://localhost:9090/healthz`.
- **Static Analysis:** No dedicated lint target is committed; adopt standard Go tooling (`go fmt`, `go vet`, `golangci-lint`) before committing.
- **CI Expectations:** Pipelines run on self-hosted runners; ensure code builds and tests locally before pushing to avoid blocking the Argo workflows.

## Build & Release Pipeline
- Main workflow: `.github/workflows/main.yaml` (named `release`). On push to `main` it:
  1. Skips duplicates.
  2. Submits `.argo/publish.yaml` via Argo CLI to build/push container images (GHCR `ghcr.io/wethegamers/agis-bot`).
  3. On success, triggers `.argo/deploy.yaml` for the development environment, then staged jobs for staging and production.
  4. Sends Discord notifications for publish/deploy success or failure.
- Argo Workflow templates (`.argo/*.yaml`) expect parameters such as `appName`, `branch`, `clusterName`, and `shortSha`.
- Helm chart versions are bumped and promoted via the release workflow (`.argo/release.yaml`).

## Kubernetes Deployment Notes
- Deployment (`charts/agis-bot/templates/deployment.yaml`) injects dozens of environment variables from `agis-bot-secrets`. Review when adding new config.
- Linkerd injection enabled via annotations; keep sidecar compatibility in mind.
- Ingress defaults to `bot-api.wethegamers.org` with TLS managed by cert-manager using `letsencrypt-prod` issuer.
- ServiceMonitor & Grafana dashboard ConfigMap are templated for Prometheus/Grafana integration.
- For GitOps promotion, see the GitOps repository referenced in Argo templates (`registry/clusters/...`).

## Observability & Operations
- Metrics: Prometheus counters/gauges/histograms for commands, servers, credit transactions, ad conversions (`prometheus.MustRegister` in `main.go`).
- Logging: Discord channel notifications via `internal/services/logging.go`; configurable through environment variables.
- Error Monitoring: Optional Sentry DSN; initialize via `services.NewErrorMonitor`.
- Cleanup Service: Background goroutine that prunes stale data (`services.NewCleanupService`).
- Consent & Compliance: GDPR consent storage and ad conversion tracking are initialized from `internal/services`.

## Documentation Touchpoints
- `README.md`: Project summary, features, and quick start.
- `COMMANDS.md`: Comprehensive Discord command catalog.
- `docs/AGONES_INTEGRATION.md`: Agones setup, Vault secret commands (includes `github_token` reference).
- `docs/OPS_MANUAL.md`: 1k+ line operations manual covering architecture, deployment, DB, monitoring, backups, security, incidents.
- `docs/README.md`: Deployment quick reference, status checklist.

## Development Conventions & Tips
- Follow idiomatic Go style; modules live under `internal/` with clear package boundaries.
- Add new configuration through `internal/config` and ensure the value is exposed via Vault + External Secrets.
- When introducing Kubernetes resources, update Helm templates and default values in `charts/agis-bot/values.yaml`.
- Keep Argo/GitHub workflow parameters in sync with Helm values and GitOps repo structure.
- Never commit real secrets; scripts in `scripts/` contain placeholders only.
- For GitHub webhook integrations, verify that `GITHUB_TOKEN` exists in Vault and is referenced in downstream services to prevent Argo deployment failures (`cannot find secret data for key: "github_token"`).

## Next Investigation Targets
- If adding new integrations, inspect `internal/http` for API handlers and ensure routes are protected with consent checks where applicable.
- For game server lifecycle changes, review `internal/services` and `internal/bot/commands` implementations.
- Whenever modifying CI/CD, validate Argo submissions (`argo-linux-amd64 submit ... --wait --log`) before merging into `main`.

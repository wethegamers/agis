# agis-bot

A Go-based Discord bot for the WTG platform.

## Features
- Role-based permissions (admin, mod, user)
- Admin, mod, and user command sets
- Public lobby management
- Economy system
- Game server management (Minecraft, CS2, Terraria, GMod)
- Diagnostics and metrics
- Database integration
- Discord rich embeds and context-aware help

## Getting Started
1. Configure environment variables (`.env` or use Vault-managed secrets via ExternalSecret in Kubernetes)
2. Set up PostgreSQL database (deployed via Bitnami Helm chart; see GitOps repo)
3. Create Discord bot application
4. Build and run the bot (see CI/CD and Argo Workflows for automated deployment)

## Deployment
- Use the provided Helm chart for Kubernetes deployment (see `charts/agis-bot`)
- All secrets (including `DISCORD_TOKEN` and DB credentials) are managed via Vault and ExternalSecrets
- ArgoCD manages deployment for dev, staging, and prod environments
- Images are built and published via Argo Workflows, triggered by GitHub Actions
- See `.argo/` for Argo Workflow templates

## Documentation
- Setup guide: `README.md`
- Environment config: `.env.example` (for local dev; in-cluster uses Vault)
- All commands documented in the help system

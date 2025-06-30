# AGIS Bot - Deployment Guide

## Quick Reference

### Repository Structure After Cleanup
- **Core Application**: `cmd/`, `internal/`, `main.go`
- **CI/CD**: `.argo/`, `.github/workflows/`
- **Kubernetes**: `charts/agis-bot/`
- **Additional Deployments**: `deployments/`
- **Documentation**: `docs/`

### Current Status ✅
- ✅ Agones GameServer integration complete
- ✅ CI/CD pipeline working (GitHub Actions + Argo Workflows)
- ✅ Discord webhook notifications configured
- ✅ Multi-environment deployment (dev/staging/prod)
- ✅ Repository structure cleaned and organized
- ✅ Kubeconfig authentication resolved

### Deployment Environments
1. **Development**: Auto-deployed on main branch
2. **Staging**: Follows development deployment
3. **Production**: Final stage after staging validation

### Key Components
- **Discord Bot**: Main AGIS bot application
- **Database**: PostgreSQL via Bitnami Helm chart
- **Secrets**: Vault + ExternalSecrets integration
- **Game Servers**: Agones Fleet management
- **Monitoring**: Discord notifications for CI/CD events

### GitHub Webhook Proxy
Located in `deployments/` directory:
- `github-discord-webhook-proxy.py` - Python webhook server
- `Dockerfile.webhook-proxy` - Container build file

For setup instructions, see `docs/webhook-setup/`.

## Next Steps
Monitor the CI/CD pipeline and Discord notifications to ensure all deployments are successful.

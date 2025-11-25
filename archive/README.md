# Archive

This directory contains historical documentation and deployment artifacts that are no longer actively used but preserved for reference.

## Structure

### deployment-milestones/
Weekly milestone tracking documents from the initial development phases:
- `WEEK1_*` - Week 1 deployment milestones (Agones integration, Sentry setup)
- `WEEK2_*` - Week 2 feature implementations (GitHub Actions, A/B testing, guild provisioning, webhook verification)

**Status**: Completed and archived. See `docs/BLOCKER_*_COMPLETED.md` for final feature documentation.

### legacy-docs/
Superseded documentation and status files:
- `DEPLOYMENT_STATUS.md` - Old deployment tracking (superseded by `docs/DEPLOYMENT_GUIDE_V2.md`)
- `DEPLOYMENT_READY.md` - Initial deployment readiness checklist
- `MIGRATION_COMPLETE.md` - Migration completion summary
- `PROJECT_COMPLETION_SUMMARY.md` - Project phase completion summary
- `INFRASTRUCTURE_SUMMARY.md` - Infrastructure setup summary
- `WEBHOOK_DEPLOYMENT_STATUS.md` - Webhook deployment tracking (superseded by `deployments/webhook-proxy/`)
- `WEBHOOK_TEST.md` - Webhook testing documentation
- `V1_7_0_DEPLOYMENT_CHECKLIST.md` - Version-specific deployment checklist (superseded by `docs/DEPLOYMENT_GUIDE_V2.md`)
- `VAULT_SETUP_CHECKLIST.md` - Vault setup checklist (superseded by `docs/VAULT_SECRETS_SETUP.md`)
- `QUICK_START_v1.7.0.md` - Version-specific quick start (superseded by `README.md`)
- `repomix-agis-bot.md` - Historical repomix output
- `log` - Old log file

**Status**: Replaced by current documentation in `docs/` directory.

## Current Documentation

For active documentation, see:
- `/README.md` - Project overview and quick start
- `/docs/OPS_MANUAL.md` - Complete operations manual
- `/docs/QUICK_REFERENCE.md` - On-call reference card
- `/docs/DEPLOYMENT_GUIDE_V2.md` - Current deployment guide
- `/docs/USER_GUIDE.md` - User documentation
- `/docs/BLOCKER_*_COMPLETED.md` - Feature implementation documentation
- `.github/copilot-instructions.md` - AI agent instructions (comprehensive project context)

## Deployment Artifacts

Active deployment configurations have been moved to:
- `/deployments/k8s/fleets/` - Agones fleet configurations
- `/deployments/webhook-proxy/` - Webhook proxy deployment files
- `/docs/setup/` - Setup and configuration guides
- `/charts/agis-bot/` - Helm chart templates

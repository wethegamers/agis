# AGIS Bot

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/wethegamers/branding/main/logo/wtg-agis-light-flat.svg">
    <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/wethegamers/branding/main/logo/wtg-agis-dark-flat.svg">
    <img src="https://raw.githubusercontent.com/wethegamers/branding/main/logo/wtg-agis-dark-flat.svg" alt="AGIS Logo" width="300">
  </picture>
</p>

<!-- Dynamic Badges (live data) -->
<p align="center">
  <a href="https://github.com/wethegamers/agis-bot/actions/workflows/ci.yaml"><img src="https://img.shields.io/github/actions/workflow/status/wethegamers/agis-bot/ci.yaml?branch=main&style=flat-square&logo=github&label=CI" alt="CI Status"></a>
  <a href="https://github.com/wethegamers/agis-bot/releases"><img src="https://img.shields.io/github/v/release/wethegamers/agis-bot?style=flat-square&logo=github&label=Release" alt="Release"></a>
  <a href="https://goreportcard.com/report/github.com/wethegamers/agis-bot"><img src="https://goreportcard.com/badge/github.com/wethegamers/agis-bot?style=flat-square" alt="Go Report Card"></a>
  <a href="https://github.com/wethegamers/agis-bot/blob/main/LICENSE"><img src="https://img.shields.io/github/license/wethegamers/agis-bot?style=flat-square" alt="License"></a>
  <a href="http://makeapullrequest.com"><img src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square" alt="PRs Welcome"></a>
</p>

<!-- Technology Stack Badges -->
<p align="center">
  <img src="https://raw.githubusercontent.com/wethegamers/branding/main/logo/badge-go.svg" alt="Go 1.23">
  <img src="https://raw.githubusercontent.com/wethegamers/branding/main/logo/badge-kubernetes.svg" alt="Kubernetes Native">
  <img src="https://raw.githubusercontent.com/wethegamers/branding/main/logo/badge-agones.svg" alt="Agones SDK">
  <img src="https://raw.githubusercontent.com/wethegamers/branding/main/logo/badge-discord.svg" alt="Discord Bot">
  <img src="https://raw.githubusercontent.com/wethegamers/branding/main/logo/badge-postgresql.svg" alt="PostgreSQL 16">
</p>

Advanced Gaming Integration System (AGIS) Bot for Discord - A Kubernetes-native Discord bot with Agones GameServer management capabilities.

**Version:** 1.7.0  
**Status:** Production Ready 🚀

## Overview

AGIS Bot powers **WeTheGamers (WTG)** - a community-driven game server hosting platform with guild economics, dynamic pricing, and zero-touch payment automation.

## Features

### Core Platform
- **16 Game Types**: Minecraft, Terraria, CS2, Valheim, Rust, ARK, Palworld, and more
- **Dynamic Pricing System**: Database-driven costs (28-39% margins) with 5-min cache
- **Guild Treasury**: Shared wallets enabling Titan-tier servers (Blue Ocean strategy)
- **Server Reviews**: 1-5 star ratings with community feedback (unique to WTG)
- **Public Lobby**: Browse and join community servers with search

### Economy & Monetization
- **Dual Currency**: GameCredits (earned) + WTG Coins (purchased)
- **Premium Subscription**: $3.99/mo with 3x multiplier, 5 WTG allowance, 100 GC daily
- **Stripe Integration**: Zero-touch payment automation with webhook fulfillment
- **Automated Subscriptions**: Auto-apply benefits, background expiration, revenue tracking

### Technical Excellence
- **Cloud-Native**: Kubernetes + Agones + PostgreSQL + Minio
- **Zero-Touch Operations**: Automated payments, subscriptions, server lifecycle
- **Production Ready**: Health endpoints, Prometheus metrics, disaster recovery (RTO 30min)
- **CI/CD**: GitHub Actions + Argo Workflows with multi-environment deployments
- **Security**: Vault secrets, ExternalSecrets, RBAC, network policies

## Project Structure

```
├── .argo/                 # Argo Workflows for CI/CD
├── .github/               # GitHub Actions workflows
├── charts/                # Helm chart for Kubernetes deployment
├── cmd/                   # Application entrypoints
├── internal/              # Internal Go packages
│   ├── agones/           # Agones SDK client
│   ├── bot/              # Discord bot and commands
│   ├── config/           # Configuration management
│   ├── database/         # Database migrations and seeds
│   ├── http/             # HTTP server (webhooks, metrics, API)
│   ├── payment/          # Stripe integration
│   └── services/         # Business logic services
├── scripts/               # Build and deployment scripts
├── deployments/           # Kubernetes deployment resources
│   ├── k8s/              # Kubernetes manifests
│   │   └── fleets/       # Agones Fleet configurations
│   ├── webhook-proxy/    # GitHub-Discord webhook proxy
│   ├── grafana/          # Grafana dashboards
│   ├── migrations/       # Database migrations
│   └── sentry/           # Sentry configuration
├── docs/                  # Documentation
│   ├── setup/            # Setup and configuration guides
│   ├── webhook-setup/    # Webhook configuration
│   └── archive/          # Historical documentation
└── archive/               # Legacy milestone tracking
```

## Quick Start

### Local Development
1. Copy `.env.example` to `.env` and configure
2. Set up PostgreSQL database
3. Run: `go run main.go`

### Kubernetes Deployment
- **Production**: Managed via ArgoCD with GitOps
- **Development**: Use Helm chart in `charts/agis-bot/`
- **Secrets**: Vault integration with ExternalSecrets

## CI/CD Pipeline

- **GitHub Actions**: Triggers on main branch pushes
- **Argo Workflows**: Container builds and multi-environment deployments
- **Discord Notifications**: Real-time CI/CD status updates
- **Environments**: Development → Staging → Production

## Documentation

### For Users
- **[User Guide](docs/USER_GUIDE.md)**: Complete guide for Discord users (591 lines)
  - Getting started, economy system, premium benefits
  - All 60+ commands with examples
  - Guild treasury guide, server management, FAQ

### For Operators
- **[Operations Manual](docs/OPS_MANUAL.md)**: Full O&M guide for DevOps/SRE (1,042 lines)
  - Architecture, deployment, monitoring, scaling
  - Database management, backup/recovery procedures
  - Security, troubleshooting runbooks, incident response
- **[Quick Reference](docs/QUICK_REFERENCE.md)**: Print-ready on-call card (401 lines)
  - 30-second health check, emergency procedures
  - Common operations, monitoring queries, useful aliases

### Technical Documentation
- **Blockers Completed**: `docs/BLOCKER_{1,2,3,4,5,6,8}_COMPLETED.md`
- **Commands**: See `COMMANDS.md` or use `@AGIS help` in Discord
- **Setup Guides**: `docs/webhook-setup/`, `docs/AGONES_INTEGRATION.md`
- **Database**: `internal/database/migrations/`, `internal/database/seeds/`

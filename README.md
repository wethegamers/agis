# AGIS Bot

Advanced Gaming Integration System (AGIS) Bot for Discord - A Kubernetes-native Discord bot with Agones GameServer management capabilities.

## Overview

A Go-based Discord bot for the WTG platform with comprehensive game server management, role-based permissions, and cloud-native deployment.

## Features

- **Game Server Management**: Agones GameServer integration for Minecraft, CS2, Terraria, GMod
- **Role-Based Access**: Admin, moderator, and user command sets with proper permissions
- **Public Lobby Management**: Community interaction and server coordination
- **Economy System**: User balance tracking and transaction management  
- **Database Integration**: PostgreSQL backend with automated migrations
- **Rich Discord Integration**: Embeds, context-aware help, and interactive commands
- **Cloud-Native**: Kubernetes deployment with GitOps, CI/CD, and secrets management

## Project Structure

```
├── .argo/                 # Argo Workflows for CI/CD
├── .github/               # GitHub Actions workflows
├── charts/                # Helm chart for Kubernetes deployment
├── cmd/                   # Application entrypoints
├── internal/              # Internal Go packages
├── scripts/               # Build and deployment scripts
├── deployments/           # Additional deployment resources
│   ├── github-discord-webhook-proxy.py
│   └── Dockerfile.webhook-proxy
└── docs/                  # Documentation
    ├── deployment/        # Deployment and completion docs
    └── webhook-setup/     # Webhook configuration guides
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

- **Commands**: See `COMMANDS.md` or use `/help` in Discord
- **Changelog**: `CHANGELOG.md`
- **Setup Guides**: `docs/webhook-setup/`
- **Deployment Status**: `docs/deployment/`

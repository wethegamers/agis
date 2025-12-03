# AGIS Bot

<p align="center">
  <img src="https://raw.githubusercontent.com/wethegamers/branding/main/logo/wtg-agis-colour.svg" alt="AGIS Logo" width="70%">
</p>

<!-- Dynamic Badges -->
<p align="center">
  <a href="https://github.com/wethegamers/agis/actions/workflows/ci.yaml"><img src="https://img.shields.io/github/actions/workflow/status/wethegamers/agis/ci.yaml?branch=main&style=flat-square&logo=github&label=CI" alt="CI Status"></a>
  <a href="https://github.com/wethegamers/agis/tags"><img src="https://img.shields.io/github/v/tag/wethegamers/agis?style=flat-square&logo=github&label=Version&sort=semver" alt="Latest Tag"></a>
  <a href="https://github.com/wethegamers/agis/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-BSL--1.1-blue?style=flat-square" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/wethegamers/agis"><img src="https://goreportcard.com/badge/github.com/wethegamers/agis?style=flat-square" alt="Go Report Card"></a>
  <a href="http://makeapullrequest.com"><img src="https://img.shields.io/badge/PRs-welcome-brightgreen?style=flat-square" alt="PRs Welcome"></a>
</p>

<!-- Technology Stack Badges -->
<p align="center">
  <img src="https://raw.githubusercontent.com/wethegamers/branding/main/logo/badge-go.svg" alt="Go 1.24">
  <img src="https://raw.githubusercontent.com/wethegamers/branding/main/logo/badge-kubernetes.svg" alt="Kubernetes Native">
  <img src="https://raw.githubusercontent.com/wethegamers/branding/main/logo/badge-agones.svg" alt="Agones SDK">
  <img src="https://raw.githubusercontent.com/wethegamers/branding/main/logo/badge-discord.svg" alt="Discord Bot">
  <img src="https://raw.githubusercontent.com/wethegamers/branding/main/logo/badge-postgresql.svg" alt="PostgreSQL 16">
</p>

Advanced Gaming Integration System (AGIS) is a Kubernetes-native Discord bot for managing game server infrastructure using Agones.

## Overview

AGIS powers **WeTheGamers (WTG)** - a community-driven game server platform with guild-based server management.

## Features

- **Game Server Management**: Deploy and manage game servers via Discord commands
- **Agones Integration**: Native Kubernetes game server orchestration
- **Guild System**: Community-based server ownership and management
- **Economy System**: In-game currency for server rentals and upgrades
- **Cloud-Native**: Built for Kubernetes with GitOps deployment

## Project Structure

```
├── charts/                # Helm chart for Kubernetes deployment
├── cmd/                   # Application entrypoints
├── main.go                # Application entry point
├── deployments/           # Kubernetes manifests and migrations
└── docs/                  # Documentation
```

> **Note**: Core business logic is maintained in a separate private repository (`agis-core`).

## Quick Start

### Prerequisites
- Go 1.24+
- PostgreSQL 16+
- Kubernetes cluster with Agones

### Local Development
```bash
cp .env.example .env
# Configure your environment variables
go run main.go
```

### Kubernetes Deployment
Deploy using the Helm chart in `charts/agis-bot/` or via ArgoCD.

## Documentation

- **[Contributing](CONTRIBUTING.md)**: How to contribute
- **[Security](SECURITY.md)**: Security policy and vulnerability reporting
- **[Code of Conduct](CODE_OF_CONDUCT.md)**: Community guidelines

## License

This project is licensed under the [Business Source License 1.1](LICENSE). See the LICENSE file for details.

# AGIS Architecture

## Overview

AGIS is built with a modular architecture designed for:
- **Maintainability**: Clear separation of concerns
- **Testability**: Components can be unit tested in isolation
- **Scalability**: Premium features loaded via Open Core pattern
- **Observability**: Comprehensive Prometheus metrics

## Package Structure

```
agis/
├── main.go                 # Entry point, bootstrapping
├── cmd/                    # CLI commands (future)
├── internal/               # Private application code
│   ├── app/                # Application lifecycle (planned)
│   ├── bot/                # Discord bot handlers (planned)
│   ├── config/             # Configuration management (planned)
│   ├── database/           # Database layer (planned)
│   ├── health/             # Kubernetes health probes (planned)
│   ├── metrics/            # Prometheus metrics ✅
│   ├── opensaas/           # Web API integration (planned)
│   ├── server/             # Game server management (planned)
│   └── scheduler/          # Server scheduling (planned)
├── pkg/                    # Public API (if needed)
└── configs/                # Configuration files
```

## Metrics Package

The `internal/metrics` package provides Prometheus metrics for all AGIS components:

### Command Metrics
- `agis_commands_total` - Total Discord commands executed
- `agis_command_duration_seconds` - Command execution latency

### Server Metrics  
- `agis_game_servers_total` - Number of managed game servers
- `agis_server_operations_total` - Server operations count

### Economy Metrics
- `agis_credits_transactions_total` - Credit transactions
- `agis_credits_amount_total` - Credit amounts by type

### User Metrics
- `agis_active_users_total` - Active user count
- `agis_users_by_tier` - Users by subscription tier

### Database Metrics
- `agis_database_operations_total` - DB operations
- `agis_database_latency_seconds` - DB operation latency

### API Metrics
- `agis_api_requests_total` - REST API requests
- `agis_api_request_duration_seconds` - API latency

### Build Info
- `agis_build_info` - Build version, commit, date

## Open Core Architecture

AGIS uses an Open Core model:

### Public Repository (agis)
- Core Discord bot functionality
- Basic game server commands
- Credits system
- Health probes and metrics
- BSD-3-Clause + BSL-1.1 licensed

### Private Repository (agis-core)
- Premium Discord features
- Advanced server scheduling
- Multi-game support
- Enterprise integrations
- Private/proprietary

The private module is loaded via Go build tags and a `go.mod` replace directive (removed in production builds).

## Configuration

Configuration is loaded from:
1. Environment variables (primary)
2. `.env` file (development)
3. Kubernetes ConfigMaps/Secrets (production)

Key variables:
- `DISCORD_TOKEN` - Bot authentication
- `DATABASE_URL` - PostgreSQL connection
- `ADMIN_GUILD_ID` - Admin Discord server
- `METRICS_PORT` - Prometheus metrics (default: 9090)
- `API_PORT` - REST API (default: 8080)

## Health Probes

Kubernetes health endpoints:
- `GET /healthz` - Liveness probe (is process alive?)
- `GET /readyz` - Readiness probe (ready for traffic?)
- `GET /health/detailed` - Detailed component health

## Future Refactoring Plan

1. **Phase 1**: Extract metrics (✅ Complete)
2. **Phase 2**: Extract health checks
3. **Phase 3**: Extract configuration management
4. **Phase 4**: Extract database layer
5. **Phase 5**: Extract bot handlers
6. **Phase 6**: Extract scheduler
7. **Phase 7**: Add OpenSaaS integration

Each phase maintains backward compatibility.

## OpenSaaS Integration (Planned)

REST API endpoints for web dashboard integration:

### User Endpoints
- `GET /api/v1/user/me` - Current user
- `POST /api/v1/user/link-discord` - Link Discord account
- `GET /api/v1/user/credits` - Get credit balance

### Server Endpoints
- `GET /api/v1/servers` - List user's servers
- `POST /api/v1/servers` - Create server
- `DELETE /api/v1/servers/{id}` - Delete server
- `POST /api/v1/servers/{id}/control` - Start/stop/restart

### Payment Endpoints
- `POST /api/v1/payments/checkout` - Create checkout session
- `GET /api/v1/payments/history` - Payment history
- `POST /api/v1/payments/webhook/stripe` - Stripe webhooks

### Subscription Endpoints
- `GET /api/v1/subscription` - Current subscription
- `POST /api/v1/subscription/upgrade` - Upgrade tier
- `POST /api/v1/subscription/cancel` - Cancel subscription

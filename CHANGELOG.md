# Changelog

All notable changes to the agis-bot project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### 🏗️ Internal Package Architecture (Best Practice Refactoring)
- **internal/app**: Application lifecycle management with graceful shutdown
  - Service registry pattern for components
  - Startup/shutdown hooks
  - Background service support with context cancellation
- **internal/config**: Type-safe configuration with environment variable binding
  - Nested configuration structures (Server, Database, Discord, Tracing)
  - Validation support with default values
  - Hot-reloadable configuration via file watching
- **internal/health**: Kubernetes-compatible health probes
  - Liveness (`/healthz`) and readiness (`/readyz`) endpoints
  - Detailed health status (`/health/detailed`)
  - Component-based health checks (database, Discord, Kubernetes)
- **internal/logging**: Structured logging with slog
  - JSON and text output formats
  - Context-based logger propagation
  - Component/user/guild tagging
  - Dynamic log level adjustment
- **internal/metrics**: Prometheus metrics with promauto
  - Build info metrics
  - Custom metric registration
  - Pre-defined business metrics
- **internal/middleware**: HTTP middleware components
  - Request ID injection
  - Prometheus metrics collection
  - Token bucket rate limiting
  - CORS headers
  - Panic recovery
  - Middleware chaining
- **internal/opensaas**: WordPress OpenSaaS integration
  - JWT authentication handler
  - User verification endpoints
- **internal/tracing**: OpenTelemetry distributed tracing
  - OTLP gRPC exporter integration
  - HTTP middleware with trace context
  - Span creation helpers
  - Configurable sampling
- **internal/version**: Build information management
  - Compile-time version injection via ldflags
  - Version HTTP handlers (simple, runtime, full)

#### 📝 Comprehensive Test Coverage
- 100+ unit tests across all internal packages
- Parallel test execution support
- Table-driven tests following Go best practices

#### 📚 Documentation
- `docs/INTEGRATION_GUIDE.md`: Complete integration guide for internal packages
- Migration patterns from standard library to internal packages

### Changed
- **main.go**: Integrated internal packages for tracing and metrics
  - OpenTelemetry initialization with environment-based configuration
  - Build info metrics using internal/version

## [0.4.0] - 2025-06-30

### � Development Release - CI/CD Integration & Discord Notifications

This development release adds CI/CD pipeline integration, Discord notification system, and GitHub webhook proxy. The bot is approximately 30-40% production ready and still in active development.

### Added

#### 📢 Complete Discord Integration System
- **CI/CD Pipeline Notifications**: Rich Discord embeds for all GitHub Actions workflow stages
  - Container publishing success/failure notifications
  - Development, staging, and production deployment status
  - Integration testing results and pipeline completion
- **Argo Workflow Integration**: Discord notifications for Argo workflow success/failure events
- **GitHub Webhook Proxy**: Custom Kubernetes-deployed service for GitHub repository events
  - Deployed at `http://74.220.19.34` with LoadBalancer access
  - Supports issues, pull requests, stars, forks, releases, and repository events
  - Prevents duplicate notifications by filtering CI/CD events handled by workflows

#### 🔧 Production-Ready CI/CD Pipeline
- **Multi-Stage Deployment**: Automated publish → development → staging → production pipeline
- **Container Registry Integration**: GHCR publishing with automated image tagging
- **Environment-Specific Deployments**: Proper environment isolation and configuration
- **Comprehensive Testing**: Integration test stage with Discord notifications

#### 🏗️ Kubernetes Infrastructure
- **Webhook Proxy Service**: Production-ready deployment with ConfigMap-based code injection
- **External Secrets Integration**: Vault-backed secret management for Discord webhooks
- **LoadBalancer Services**: External access for GitHub webhook integration
- **Health Monitoring**: Kubernetes liveness and readiness probes

#### 📚 Comprehensive Documentation
- **Deployment Guide**: Complete webhook setup and configuration documentation
- **Setup Scripts**: Automated webhook configuration and testing tools
- **Status Documentation**: Real-time deployment status and troubleshooting guides

### Changed

#### 🔄 CI/CD Architecture Overhaul
- **Notification Strategy**: Migrated from scattered notifications to unified Discord system
- **Webhook Integration**: Centralized GitHub event handling via dedicated proxy service
- **Environment Configuration**: Proper secret management and environment variable handling
- **Deployment Process**: Streamlined multi-environment deployment with status tracking

#### 💻 Development Workflow
- **Branch Strategy**: Optimized for main-branch CI/CD with proper environment promotion
- **Testing Integration**: Automated testing with comprehensive failure reporting
- **Error Handling**: Enhanced error reporting with direct links to workflow logs

### Fixed

#### 🐛 Critical Production Issues
- **Webhook Proxy Deployment**: Resolved container deployment issues with ConfigMap approach
- **Secret Management**: Fixed Discord webhook URL access across Kubernetes namespaces
- **External Access**: Configured LoadBalancer service for reliable GitHub webhook delivery
- **Notification Duplicates**: Eliminated duplicate notifications between CI/CD and webhook events

#### 🔧 Infrastructure Improvements
- **Service Discovery**: Fixed external IP allocation and service exposure
- **Container Configuration**: Resolved Python dependency installation in production environment
- **Health Checks**: Proper HTTP endpoint configuration for Kubernetes probes

### Technical Details

#### 🌐 Service Architecture
```
GitHub Repository → GitHub Webhook → Webhook Proxy (K8s) → Discord Channel
                                   ↗ LoadBalancer IP: 74.220.19.34
                                   
CI/CD Pipeline → GitHub Actions → Argo Workflows → Discord Notifications
```

#### 📦 New Components
- `github-discord-webhook-proxy.py` - Python webhook translation service
- `k8s-github-webhook-proxy-configmap.yaml` - Kubernetes deployment manifests
- `setup-github-webhook.sh` - Automated webhook configuration script
- `WEBHOOK_DEPLOYMENT_STATUS.md` - Comprehensive deployment documentation

#### 🔗 External Integrations
- **GitHub Webhooks**: Repository events → Discord notifications
- **Discord API**: Rich embed formatting with environment-specific styling
- **Kubernetes LoadBalancer**: External webhook endpoint exposure
- **Vault Secrets**: Secure webhook URL management

### Migration Notes

#### ⚠️ Breaking Changes
- **Webhook Configuration**: GitHub repository webhook must be configured manually
- **Secret Requirements**: Discord webhook URL must be available in Kubernetes secrets
- **Network Requirements**: External LoadBalancer access required for webhook proxy

#### 🚀 Deployment Requirements
1. **Kubernetes Cluster**: Must support LoadBalancer services
2. **External Secrets**: Vault integration for Discord webhook URL
3. **GitHub Repository Access**: Admin access required for webhook configuration
4. **Discord Channel**: Webhook URL configured for target Discord channel

### 📋 Setup Checklist
- ✅ Webhook proxy deployed to Kubernetes (`http://74.220.19.34`)
- ✅ CI/CD pipeline with Discord notifications active
- ✅ Argo workflow Discord integration configured
- ✅ External secrets configured for webhook URLs
- ⏳ **Manual Step Required**: Configure GitHub repository webhook

### 🎯 Next Steps
- Configure GitHub repository webhook at: https://github.com/wethegamers/agis-bot/settings/hooks
- Monitor Discord channel for comprehensive notification coverage
- Test end-to-end integration with repository events

---

## [0.2.0] - 2025-06-29

### Added
- **Agones GameServer Integration**: Full integration with Agones for Kubernetes-native game server management
- **Real-time Status Monitoring**: Live GameServer status tracking via Kubernetes APIs
- **Fleet-based Server Allocation**: Proper GameServer allocation using Agones Fleets (free-tier, premium-tier)
- **Enhanced User Notifications**: DM notifications at each deployment stage with detailed feedback
- **Live Connection Info**: Automatic delivery of server IP:Port as soon as servers are ready
- **Advanced Diagnostics**: Real-time server health checks and troubleshooting commands
- **Kubernetes UID Tracking**: Unique server identification for accurate lifecycle management
- **Enhanced Database Schema**: New columns for Agones/Kubernetes integration
  - `kubernetes_uid` - Unique GameServer identifier  
  - `agones_status` - Live Agones GameServer status
  - `last_status_sync` - Status synchronization timestamp
  - `stopped_at`, `cleanup_at` - Lifecycle tracking
  - `error_message` - Detailed error information

### New Services
- **AgonesService**: GameServer allocation, status monitoring, and Fleet management
- **NotificationService**: User DM notifications with deployment stage tracking
- **EnhancedServerService**: Orchestrates server lifecycle with live status integration

### Enhanced Commands
- **servers**: Shows live connection info, real-time status, and detailed server information
- **diagnostics**: Live Kubernetes/Agones status checks with connectivity verification
- **server_management**: Fleet-based allocation with comprehensive user feedback

### Changed
- **Server Creation Workflow**: Now uses Agones GameServerAllocation API instead of direct creation
- **Status Display**: Real-time status from Kubernetes instead of database-only tracking
- **Error Handling**: Enhanced error messages with actionable troubleshooting guidance
- **Notification System**: Proactive DM updates throughout server deployment process

### Dependencies
- Added `agones.dev/agones v1.38.0` for GameServer management
- Added `k8s.io/client-go v0.28.4` for Kubernetes API integration
- Added `k8s.io/apimachinery v0.28.4` for Kubernetes object handling

### Technical Improvements
- Fleet-based architecture following Agones best practices
- Proper Kubernetes controller integration (no bypassing)
- Live API queries for accurate status reporting
- Enhanced command context with new service dependencies
- Comprehensive database migration for new schema

## [0.1.11] - 2025-06-29

### Added
- Enhanced logging system with dedicated channel support
- LOG_CHANNEL_* environment variables for different log types
- Database schema migrations for logging system

### Changed
- Updated Helm chart to version 0.1.11
- Enhanced deployment template with comprehensive logging environment variables
- Improved CI/CD pipeline to match metaphor standard

### Removed
- Old chart packages (0.1.2-0.1.6) cleaned up
- ADMIN_ROLES and MOD_ROLES variables (replaced with proper RBAC)

## [0.1.10] - 2025-06-28

### Fixed
- Ingress template to properly use values instead of hardcoded hostname
- Dockerfile to build from correct main.go and update to port 8080
- Deployment template to use secretKeyRef instead of Values.env

### Removed
- NOTES.txt template to fix dependency chart issues

## [0.1.9] - 2025-06-28

### Fixed
- Added missing chartDir parameter to publish workflow
- Updated publish.yaml to match metaphor structure

## [0.1.8] - 2025-06-28

### Added
- ImagePullSecrets support to deployment template

### Fixed
- CI/CD pipeline configuration

## [0.1.7] - 2025-06-28

### Changed
- Modernized CI/CD pipeline to match metaphor standard

## [0.1.6] - 2025-06-28

### Changed
- Set safe default values in values.yaml to prevent ingress conflicts
- Clarified environment variable overrides

## [0.1.5] - 2025-06-28

### Removed
- Legacy and confusing external-secrets.yaml templates
- Cleanup Helm chart structure

### Fixed
- ArgoCD Application manifests to use correct Helm chart version

## [0.1.4] - 2025-06-28

### Fixed
- Removed test-connection.yaml to fix Helm parse error

## [0.1.3] - 2025-06-28

### Added
- Updated Helm chart and documentation for 0.1.1 release
- Improved chart templates and structure

## [0.1.2] - 2025-06-28

### Added
- Initial Helm chart and templates
- Basic deployment configuration

## [0.1.1] - 2025-06-28

### Added
- Initial project scaffolding from metaphor template
- Basic Discord bot structure
- Kubernetes deployment manifests
- CI/CD pipeline setup

## [0.1.0] - 2025-06-28

### Added
- Initial commit: agis-bot project scaffolded from metaphor template
- Basic project structure and configuration

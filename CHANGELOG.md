# Changelog

All notable changes to the agis-bot project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

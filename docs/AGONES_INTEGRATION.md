# Agones Integration Guide

## Overview

agis-bot is now fully integrated with Agones for game server management on your Kubernetes cluster. This integration provides Discord-based controls for managing game servers, fleets, and allocations.

## Features

### Discord Commands

- `!servers` - List all game servers with their current status
- `!fleet [name]` - Show fleet status (defaults to agis-dev-fleet)
- `!allocate [fleet]` - Allocate a game server from a fleet
- `!scale <fleet> <replicas>` - Scale fleet (admin only)
- `!create [name]` - Create a new game server
- `!delete <name>` - Delete a game server
- `!status [name]` - Show detailed server status

### Kubernetes Resources

The integration manages the following Agones resources:
- **GameServers**: Individual game server instances
- **Fleets**: Collections of game servers
- **FleetAutoscalers**: Automatic scaling policies
- **GameServerAllocations**: Server allocation management

## Architecture

```
Discord Users
     ↓
agis-bot (Discord Bot)
     ↓
Agones Client Library
     ↓
Kubernetes API
     ↓
Agones Controller
     ↓
Game Servers (Pods)
```

## Configuration

### Environment Variables

- `AGONES_NAMESPACE`: Namespace where Agones resources are deployed (default: agones-system)
- `DISCORD_TOKEN`: Discord bot token
- `DISCORD_CLIENT_ID`: Discord application client ID
- `GITHUB_TOKEN`: GitHub token for webhook integration
- `WEBHOOK_SECRET`: Secret for webhook validation

### Vault Secrets

Secrets are stored in Vault at `development/agis-bot`:

```bash
vault kv put secret/development/agis-bot \
  discord_token="<token>" \
  discord_client_id="<id>" \
  github_token="<token>" \
  webhook_secret="<secret>" \
  agones_namespace="agones-system"
```

### ConfigMap Settings

The bot configuration is managed via ConfigMap in Kubernetes:

```yaml
agones:
  namespace: agones-system
  fleet_name: agis-dev-fleet
  allocator_endpoint: agones-allocator-service.agones-system.svc.cluster.local:443
  metrics_enabled: true

discord:
  command_prefix: "!"
  admin_roles:
    - "Admin"
    - "GameMaster"
```

## RBAC Permissions

agis-bot requires the following permissions:

### Agones Resources
- Full control over GameServers, Fleets, and FleetAutoscalers
- Create and list GameServerAllocations

### Kubernetes Resources
- Read access to Pods, Nodes, Services, ConfigMaps
- Create Events for audit logging

## Deployment

### 1. Deploy via ArgoCD

The deployment is managed through ArgoCD applications:

```bash
# Apply ArgoCD applications
kubectl apply -f registry/clusters/wtg-dev/agones-dev.yaml
kubectl apply -f registry/clusters/wtg-dev/agis-bot.yaml
```

### 2. Configure Secrets

```bash
# Run the setup script
./scripts/setup-agis-bot-secrets.sh
```

### 3. Verify Deployment

```bash
# Check agis-bot pod
kubectl get pods -n development -l app=agis-bot

# Check logs
kubectl logs -n development -l app=agis-bot

# Check service
kubectl get svc -n development agis-bot
```

## Integration with Agones

### Fleet Management

agis-bot manages the `agis-dev-fleet` which is configured with:
- **Initial replicas**: 2
- **Min replicas**: 2
- **Max replicas**: 10
- **Scheduling**: Packed (optimizes resource usage)
- **Auto-scaling**: Buffer-based (maintains 2 ready servers)

### Server Lifecycle

1. **Creation**: Servers are created from the fleet template
2. **Allocation**: Players request servers via Discord
3. **Usage**: Server transitions to Allocated state
4. **Shutdown**: Server is terminated after use
5. **Replacement**: Fleet autoscaler creates new servers

### Monitoring

The bot provides real-time monitoring through:
- Discord status updates
- Prometheus metrics export
- Kubernetes events
- Health and readiness probes

## Webhook Integration

agis-bot exposes webhook endpoints for:
- GitHub notifications
- Agones autoscaling callbacks
- External game server events

Webhooks are available at:
- `https://agis-bot.wtg-dev.lan/webhook/github`
- `https://agis-bot.wtg-dev.lan/webhook/agones`

## Troubleshooting

### Common Issues

1. **Bot not responding to commands**
   - Check Discord token in Vault
   - Verify bot has correct permissions in Discord server
   - Check logs: `kubectl logs -n development -l app=agis-bot`

2. **Cannot allocate servers**
   - Verify fleet has available servers: `kubectl get fleet -n agones-system`
   - Check allocation policy in ConfigMap
   - Ensure RBAC permissions are correct

3. **Connection errors to Agones**
   - Verify agones-system namespace exists
   - Check ServiceAccount permissions
   - Ensure Agones controller is running

### Debug Commands

```bash
# Check Agones status
kubectl get gs -n agones-system
kubectl get fleet -n agones-system

# Check agis-bot logs
kubectl logs -n development -l app=agis-bot -f

# Test allocation manually
kubectl create -f - <<EOF
apiVersion: allocation.agones.dev/v1
kind: GameServerAllocation
metadata:
  generateName: test-allocation-
  namespace: agones-system
spec:
  required:
    matchLabels:
      agones.dev/fleet: agis-dev-fleet
EOF
```

## Performance Optimizations

The cluster has been optimized for gaming workloads:
- **CPU**: Performance governor enabled
- **Network**: TCP optimizations for 20-30% latency reduction
- **Storage**: SSD-optimized I/O schedulers
- **Memory**: Optimized swappiness and cache settings

## Next Steps

1. **Configure Discord Server**
   - Add bot to your Discord server
   - Create appropriate channels (#game-servers, #server-status)
   - Set up roles (Admin, GameMaster)

2. **Deploy Real Game Servers**
   - Replace simple-game-server with your actual game
   - Update fleet configuration for your game requirements
   - Configure appropriate resource limits

3. **Set up Monitoring**
   - Import Agones Grafana dashboards
   - Configure alerts for server availability
   - Set up log aggregation

4. **Scale for Production**
   - Increase fleet min/max replicas
   - Configure multi-region allocation
   - Implement game-specific health checks
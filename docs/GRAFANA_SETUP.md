# Grafana Dashboard Setup

This guide covers setting up Grafana dashboards for AGIS Bot ad conversion metrics.

## Prerequisites

- Grafana instance accessible (e.g., Grafana Cloud or self-hosted)
- Prometheus data source configured in Grafana
- AGIS Bot exposing metrics on `:9090/metrics`

## Dashboard Installation

### Option 1: Import JSON (Recommended)

1. Open Grafana UI → Dashboards → Import
2. Upload `deployments/grafana/ad-metrics-dashboard.json`
3. Select your Prometheus data source
4. Click "Import"

### Option 2: Provisioning (GitOps)

Add to your Grafana Helm values or configmap:

```yaml
dashboards:
  default:
    agis-ad-metrics:
      json: |
        # Paste contents of ad-metrics-dashboard.json
```

### Option 3: Terraform

```hcl
resource "grafana_dashboard" "agis_ad_metrics" {
  config_json = file("${path.module}/../../deployments/grafana/ad-metrics-dashboard.json")
}
```

## Dashboard Panels

### Top Row (Real-time KPIs)
- **Conversion Rate** (24h) - Green >80%, Yellow >50%, Red <50%
- **Total Revenue** - Cumulative Game Credits distributed
- **Fraud Rate** - Percentage of fraud attempts (alert if >10%)
- **Active Conversions** - Last 5 minutes activity

### Mid Section (Time Series)
- **Conversions Over Time** - By ad type (offerwall/surveywall/video) + fraud attempts
- **Revenue by Ad Type** - GC/sec breakdown
- **Callback Latency** - P95/P99 by provider

### Bottom Section (Distribution)
- **Conversions by User Tier** - Pie chart (free/premium/premium_plus)
- **Fraud Detection Breakdown** - Pie chart by reason (velocity/ip_hopping/excessive_earnings)
- **Hourly Revenue Trend** - Full-width time series

## Alert Configuration

Recommended alerts to configure in Grafana:

### High Fraud Rate
```yaml
expr: (sum(agis_ad_fraud_attempts_total) / (sum(agis_ad_conversions_total) + sum(agis_ad_fraud_attempts_total))) * 100 > 10
for: 5m
severity: warning
```

### Low Conversion Rate
```yaml
expr: rate(agis_ad_conversions_total{status="completed"}[1h]) < 0.5
for: 10m
severity: warning
```

### High Callback Latency
```yaml
expr: histogram_quantile(0.95, sum(rate(agis_ad_callback_latency_seconds_bucket[5m])) by (le)) > 2
for: 5m
severity: warning
```

### Zero Conversions
```yaml
expr: sum(rate(agis_ad_conversions_total[15m])) == 0
for: 15m
severity: critical
```

## Prometheus Configuration

Ensure AGIS Bot metrics endpoint is scraped:

```yaml
scrape_configs:
  - job_name: 'agis-bot'
    static_configs:
      - targets: ['agis-bot:9090']
    scrape_interval: 15s
```

For Kubernetes (ServiceMonitor):

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: agis-bot
  namespace: development
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: agis-bot
  endpoints:
    - port: metrics
      interval: 15s
```

## Metrics Reference

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `agis_ad_conversions_total` | Counter | provider, type, status | Total conversions |
| `agis_ad_rewards_total` | Counter | provider, type | Total GC distributed |
| `agis_ad_fraud_attempts_total` | Counter | provider, reason | Fraud attempts |
| `agis_ad_callback_latency_seconds` | Histogram | provider, status | Callback latency |
| `agis_ad_conversions_by_tier_total` | Counter | tier | Conversions by tier |

## Troubleshooting

### Dashboard shows "No Data"
- Verify Prometheus is scraping AGIS Bot: `http://prometheus:9090/targets`
- Check metrics are exposed: `curl http://agis-bot:9090/metrics`
- Confirm data source is selected in dashboard settings

### Metrics not updating
- Check `METRICS_PORT` environment variable is set (default: 9090)
- Verify HTTP server is enabled in AGIS Bot
- Review Prometheus scrape errors in logs

### Incorrect calculations
- Ensure counter metrics are used with `rate()` or `increase()` functions
- Verify label matching in PromQL queries
- Check for counter resets (pod restarts)

## Production Recommendations

1. **Data Retention**: Configure Prometheus retention for at least 30 days
2. **High Availability**: Run multiple Prometheus replicas with Thanos/Cortex
3. **Alerting**: Set up Alertmanager with Discord/Slack notifications
4. **Backup**: Export dashboards to version control (this repo)
5. **Access Control**: Use Grafana RBAC for viewer/editor permissions

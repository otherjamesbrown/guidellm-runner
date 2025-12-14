# GuideLLM-Runner Observability Architecture

## Overview

This document describes how to integrate guidellm-runner load testing metrics with the ai-aas platform observability stack.

## Architecture Options

### Option 1: Central Grafana with Multiple Prometheus Sources (Recommended)

```
┌─────────────────────────────────────────────────────────────────┐
│                     Central Grafana                              │
│            (Can be in dev cluster or standalone)                 │
└────────┬────────────────┬────────────────┬─────────────────────┘
         │                │                │
         ▼                ▼                ▼
┌─────────────┐   ┌─────────────┐   ┌─────────────────────┐
│ Dev Cluster │   │ Staging     │   │ guidellm-runner     │
│ Prometheus  │   │ Prometheus  │   │ Prometheus          │
│ (vLLM,DCGM) │   │ (vLLM,DCGM) │   │ (load test metrics) │
└─────────────┘   └─────────────┘   └─────────────────────┘
```

**Pros:**
- Simple to implement
- Each component is independent
- Can run load tester from anywhere

**Cons:**
- Need to manage multiple data sources in Grafana
- Cross-source queries require mixed queries

**Implementation:**
1. Deploy guidellm-runner with its own Prometheus
2. Add all Prometheus instances as data sources in central Grafana
3. Create dashboards with panels from multiple sources

### Option 2: Prometheus Remote Write (Centralized Metrics)

```
┌─────────────────────────────────────────────────────────────────┐
│                 Central Prometheus/Thanos                        │
│              (Single source of truth for all metrics)            │
└────────┬────────────────┬────────────────┬─────────────────────┘
         ▲                ▲                ▲
         │ remote_write   │ remote_write   │ remote_write
         │                │                │
┌─────────────┐   ┌─────────────┐   ┌─────────────────────┐
│ Dev Cluster │   │ Staging     │   │ guidellm-runner     │
│ Prometheus  │   │ Prometheus  │   │ Prometheus          │
└─────────────┘   └─────────────┘   └─────────────────────┘
```

**Pros:**
- All metrics in one place
- Easy correlation queries (PromQL joins)
- Single data source in Grafana

**Cons:**
- More complex setup
- Requires central Prometheus/Thanos/Mimir
- Network connectivity requirements

### Option 3: Run Load Tester In-Cluster (Simplest)

```
┌─────────────────────────────────────────────────────────────────┐
│                      Dev Cluster                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────────────────┐ │
│  │  vLLM Pods  │  │ DCGM Export │  │  guidellm-runner Pod     │ │
│  │  :8000      │  │  (GPU)      │  │  :9090 (ServiceMonitor)  │ │
│  └──────┬──────┘  └──────┬──────┘  └────────────┬─────────────┘ │
│         │                │                      │               │
│         └────────────────┼──────────────────────┘               │
│                          ▼                                      │
│              ┌───────────────────────┐                          │
│              │   Cluster Prometheus  │                          │
│              │   (all metrics)       │                          │
│              └───────────┬───────────┘                          │
│                          ▼                                      │
│              ┌───────────────────────┐                          │
│              │       Grafana         │                          │
│              └───────────────────────┘                          │
└─────────────────────────────────────────────────────────────────┘
```

**Pros:**
- All metrics in cluster Prometheus automatically
- Easy correlation (same labels, same time)
- No external dependencies

**Cons:**
- Load tester inside cluster (might affect results)
- Need to deploy per cluster

## Recommended: Hybrid Approach

For your use case (dev + staging clusters), I recommend:

### Phase 1: In-Cluster Load Tester (Quick Win)

Deploy guidellm-runner as a Kubernetes Deployment in each cluster:

```yaml
# Deploy in each cluster's monitoring namespace
apiVersion: apps/v1
kind: Deployment
metadata:
  name: guidellm-runner
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: guidellm-runner
  template:
    metadata:
      labels:
        app: guidellm-runner
    spec:
      containers:
      - name: runner
        image: guidellm-runner:latest
        ports:
        - containerPort: 9090
          name: metrics
        volumeMounts:
        - name: config
          mountPath: /app/configs
      volumes:
      - name: config
        configMap:
          name: guidellm-runner-config
---
apiVersion: v1
kind: Service
metadata:
  name: guidellm-runner
  namespace: monitoring
  labels:
    app: guidellm-runner
spec:
  ports:
  - port: 9090
    targetPort: 9090
    name: metrics
  selector:
    app: guidellm-runner
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: guidellm-runner
  namespace: monitoring
  labels:
    release: kube-prometheus-stack
spec:
  selector:
    matchLabels:
      app: guidellm-runner
  endpoints:
  - port: metrics
    interval: 15s
```

### Phase 2: Central Grafana (Multi-Cluster View)

Use one Grafana (e.g., in dev cluster) with multiple data sources:

```yaml
# In Grafana data sources
datasources:
  - name: prometheus-dev
    type: prometheus
    url: http://kube-prometheus-stack-prometheus.monitoring:9090

  - name: prometheus-staging
    type: prometheus
    url: https://prometheus.staging.otherjamesbrown.com
    # Or via port-forward/service mesh
```

## Dashboard Design

### Combined Load Test Dashboard

Create a dashboard with these sections:

1. **Load Test Status** (from guidellm-runner)
   - Request rate sent
   - Success/failure rate
   - Client-observed latency (TTFT, E2E)

2. **Server Performance** (from vLLM)
   - Server-side throughput (tokens/sec)
   - Server-side latency (TTFT, ITL)
   - Queue depth

3. **Resource Utilization** (from DCGM)
   - GPU utilization
   - GPU memory
   - Power consumption

4. **Correlation Panel**
   - Client latency vs Server latency
   - Request rate vs GPU utilization

### Example Queries

```promql
# Client-side (guidellm-runner)
rate(guidellm_requests_total{environment="development"}[1m])
histogram_quantile(0.95, rate(guidellm_ttft_seconds_bucket[5m]))

# Server-side (vLLM)
vllm:avg_generation_throughput_toks_per_s{model_name=~".*mistral.*"}
histogram_quantile(0.95, rate(vllm:time_to_first_token_seconds_bucket[5m]))

# GPU (DCGM)
avg(DCGM_FI_DEV_GPU_UTIL)
sum(DCGM_FI_DEV_POWER_USAGE)
```

## Configuration per Environment

### Development Cluster

```yaml
# configs/config-development.yaml
environments:
  development:
    targets:
      - name: mistral-7b
        # Use cluster-internal URL
        url: http://mistral-7b-predictor.development.svc.cluster.local:8012/v1/chat/completions
        model: mistralai/Mistral-7B-Instruct-v0.3
        api_key: ${API_KEY}
```

### Staging Cluster

```yaml
# configs/config-staging.yaml
environments:
  staging:
    targets:
      - name: mistral-7b
        url: http://mistral-7b-predictor.staging.svc.cluster.local:8012/v1/chat/completions
        model: mistralai/Mistral-7B-Instruct-v0.3
        api_key: ${API_KEY}
```

## Label Strategy

To enable correlation across metrics:

| Source | Labels | Purpose |
|--------|--------|---------|
| guidellm-runner | `environment`, `target`, `model` | Load test identification |
| vLLM | `model_name`, `pod`, `namespace` | Server-side correlation |
| DCGM | `exported_pod`, `modelName`, `node` | GPU correlation |

Ensure `model` label in guidellm-runner matches `model_name` in vLLM for easy joining.

## Implementation Steps

1. **Dockerize guidellm-runner** (Dockerfile exists)
2. **Create Helm chart** for deployment
3. **Add ServiceMonitor** for Prometheus scraping
4. **Deploy to each cluster**
5. **Configure Grafana data sources**
6. **Create combined dashboard**

## Future: External Load Testing

For more realistic load testing (external to cluster):

1. Run guidellm-runner externally
2. Use Prometheus remote_write to push metrics to cluster Prometheus
3. Or use central Thanos/Mimir as aggregation layer

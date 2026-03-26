# KubeGuard 🛡️

[![Validate](https://github.com/bukx/kubeguard/actions/workflows/ci.yml/badge.svg)](https://github.com/bukx/kubeguard/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/bukx/kubeguard)](https://goreportcard.com/report/github.com/bukx/kubeguard)
![Go](https://img.shields.io/badge/Go-00ADD8?style=flat&logo=go&logoColor=white)
![Kubernetes](https://img.shields.io/badge/Kubernetes-326CE5?style=flat&logo=kubernetes&logoColor=white)

**A Kubernetes operator that enforces SLOs, guards pod best practices, and automates namespace lifecycle management.**

KubeGuard brings SRE discipline to your cluster as a set of Custom Resource Definitions (CRDs) and controllers that run inside the cluster, continuously reconciling desired state.

---

## ✨ Features

### 🤖 LLM Ops Ready
KubeGuard's guardrails map cleanly onto GPU-heavy LLM inference clusters. See [docs/llm-guardrails.md](docs/llm-guardrails.md) for concrete policies and SLO examples.


### 🎯 SLO Controller
Define Service Level Objectives as Kubernetes resources. KubeGuard monitors them, calculates error budgets in real-time, and auto-generates Prometheus `PrometheusRule` alert rules.

```yaml
apiVersion: kubeguard.bukx.dev/v1alpha1
kind: ServiceLevelObjective
metadata:
  name: checkout-availability
  namespace: production
spec:
  service: checkout-api
  description: "Checkout API availability SLO"
  target: "99.9"
  window: 30d
  indicator:
    type: availability
    totalQuery: 'sum(rate(http_requests_total{service="checkout-api"}[5m]))'
    errorQuery: 'sum(rate(http_requests_total{service="checkout-api",code=~"5.."}[5m]))'
  alerting:
    burnRateThresholds:
      - severity: critical
        shortWindow: 5m
        longWindow: 1h
        factor: 14.4
      - severity: warning
        shortWindow: 30m
        longWindow: 6h
        factor: 6
```

**What it does:**
- Calculates current error budget remaining (%)
- Generates multi-window, multi-burn-rate Prometheus alerts
- Updates `.status` with real-time SLO compliance
- Emits Kubernetes events on SLO breaches

### 🛡️ Pod Guardian (Admission Webhook)
A validating and mutating admission webhook that enforces pod security best practices:

| Check | Action | Default |
|-------|--------|---------|
| Missing resource limits | **Reject** or inject defaults | Reject |
| Missing liveness/readiness probes | **Warn** | Warn |
| Running as root | **Reject** | Reject |
| Missing security context | **Mutate** — inject `runAsNonRoot: true` | Mutate |
| Privileged containers | **Reject** | Reject |
| Latest tag images | **Warn** | Warn |

Configure policies per namespace via `PodGuardPolicy` CRD:

```yaml
apiVersion: kubeguard.bukx.dev/v1alpha1
kind: PodGuardPolicy
metadata:
  name: production-policy
  namespace: production
spec:
  enforcement: strict  # strict | permissive | audit
  rules:
    requireResourceLimits: true
    requireProbes: true
    disallowRoot: true
    disallowPrivileged: true
    disallowLatestTag: true
    maxCPULimit: "2"
    maxMemoryLimit: "4Gi"
```

### 🏗️ Namespace Lifecycle Controller
Automate namespace provisioning with RBAC, quotas, network policies, and monitoring — all from a single CR:

```yaml
apiVersion: kubeguard.bukx.dev/v1alpha1
kind: ManagedNamespace
metadata:
  name: team-payments
spec:
  owner: payments-team@company.com
  environment: production
  expiresAt: "2025-12-31T00:00:00Z"  # Optional TTL
  resourceQuota:
    hard:
      cpu: "20"
      memory: "40Gi"
      pods: "100"
  limitRange:
    defaultLimit:
      cpu: "500m"
      memory: "512Mi"
    defaultRequest:
      cpu: "100m"
      memory: "128Mi"
  networkPolicy:
    allowIngress: true
    allowEgress: true
    allowedNamespaces:
      - istio-system
      - monitoring
  rbac:
    - role: admin
      subjects:
        - kind: Group
          name: payments-team
    - role: view
      subjects:
        - kind: Group
          name: platform-team
  monitoring:
    enablePrometheus: true
    alertEmail: payments-team@company.com
```

**What it does:**
- Creates the namespace with labels and annotations
- Applies ResourceQuota and LimitRange
- Creates NetworkPolicy for isolation
- Binds RBAC roles to specified subjects
- Sets up Prometheus ServiceMonitor
- Auto-deletes expired namespaces (with warning events)

---

## 🏗 Architecture

```
                    ┌─────────────────────────────────┐
                    │       Kubernetes API Server      │
                    └─────┬─────────────┬─────────────┘
                          │             │
              Watch CRDs  │             │  Admission Webhooks
                          │             │
               ┌──────────▼──────┐ ┌────▼──────────────┐
               │   Controllers   │ │   Pod Guardian     │
               │                 │ │   (Webhook Server) │
               │ ┌─────────────┐ │ │                    │
               │ │ SLO         │ │ │ • Validate pods    │
               │ │ Controller  │ │ │ • Mutate defaults  │
               │ ├─────────────┤ │ │ • Enforce policies │
               │ │ Namespace   │ │ └────────────────────┘
               │ │ Lifecycle   │ │
               │ │ Controller  │ │
               │ ├─────────────┤ │
               │ │ PodGuard    │ │
               │ │ Policy Ctrl │ │
               │ └─────────────┘ │
               └────────┬────────┘
                        │
            Generates   │
                        │
               ┌────────▼────────┐
               │  Prometheus     │
               │  Alert Rules    │
               │  (PrometheusRule│
               │   resources)    │
               └─────────────────┘
```

## 🚀 Quick Start

### Prerequisites
- Kubernetes cluster (v1.25+)
- `kubectl` configured
- `cert-manager` installed (for webhook TLS)
- Prometheus Operator (optional, for SLO alerts)

### Install

```bash
# Install CRDs
kubectl apply -f config/crd/bases/

# Deploy the operator
kubectl apply -f config/manager/

# Deploy webhook configuration
kubectl apply -f config/webhook/

# Create a sample SLO
kubectl apply -f config/samples/slo.yaml

# Create a managed namespace
kubectl apply -f config/samples/managed-namespace.yaml

# Create a pod guard policy
kubectl apply -f config/samples/pod-guard-policy.yaml
```

### Build from source

```bash
# Build
make build

# Run tests
make test

# Build container image
make docker-build IMG=bukx/kubeguard:latest

# Push to registry
make docker-push IMG=bukx/kubeguard:latest

# Deploy to cluster
make deploy IMG=bukx/kubeguard:latest
```

## 📈 Metrics

KubeGuard exposes Prometheus metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `kubeguard_slo_error_budget_remaining` | Gauge | Error budget remaining (0-1) |
| `kubeguard_slo_compliance` | Gauge | Current SLO compliance (0-1) |
| `kubeguard_pods_rejected_total` | Counter | Pods rejected by admission webhook |
| `kubeguard_pods_mutated_total` | Counter | Pods mutated by admission webhook |
| `kubeguard_namespaces_managed` | Gauge | Number of managed namespaces |
| `kubeguard_namespaces_expired_total` | Counter | Namespaces cleaned up due to TTL |

## 📁 Project Structure

```
├── api/v1alpha1/              # CRD type definitions
│   ├── slo_types.go
│   ├── podguardpolicy_types.go
│   ├── managednamespace_types.go
│   └── groupversion_info.go
├── internal/
│   ├── controller/            # Reconciliation controllers
│   │   ├── slo_controller.go
│   │   ├── namespace_controller.go
│   │   └── podguard_controller.go
│   └── webhook/               # Admission webhook handlers
│       └── pod_validator.go
├── config/
│   ├── crd/bases/             # Generated CRD manifests
│   ├── rbac/                  # RBAC for the operator
│   ├── manager/               # Operator deployment
│   ├── webhook/               # Webhook configuration
│   └── samples/               # Example CRs
├── .github/workflows/         # CI pipeline
├── Dockerfile
├── Makefile
└── go.mod
```

## 🤝 Contributing

PRs welcome! Please:
1. Fork the repo
2. Create a feature branch
3. Write tests for new functionality
4. Submit a PR

## 📜 License

Apache License 2.0

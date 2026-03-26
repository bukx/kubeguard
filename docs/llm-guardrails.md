# Operating LLM Inference Workloads with KubeGuard

Large Language Model (LLM) inference services are resource hungry, latency sensitive, and often subject to unpredictable bursts of traffic. KubeGuard can enforce the right guardrails so that an LLM platform remains stable even as model sizes and request rates grow.

## Why LLM workloads are tricky

- **Spiky demand:** Launch events, prompt-chaining pipelines, or agentic workloads can create bursty traffic that overwhelms GPU pods.
- **Cost sensitivity:** GPUs are expensive. Without SLOs and right-sizing, costs explode quickly.
- **Security:** Prompt ingestion services often need to run untrusted user code and require strict pod policies.

## Applying KubeGuard features

| KubeGuard Capability | LLM Use Case |
|----------------------|-------------|
| **SLO Controller** | Track latency/availability SLOs for `/generate` endpoints, automatically create burn-rate alerts, and feed on-call dashboards.
| **Namespace Lifecycle** | Provision isolated namespaces per model family (e.g., `gpt4`, `mixtral`) with quotas and TTLs for experiments.
| **PodGuard Policy** | Enforce GPU request/limit symmetry, disable privileged mode, and require probes on Triton/DeepSpeed pods.

### Example PodGuardPolicy for GPU pods

```yaml
apiVersion: kubeguard.bukx.dev/v1alpha1
kind: PodGuardPolicy
metadata:
  name: llm-gpu-policy
  namespace: llm-prod
spec:
  enforcement: strict
  rules:
    requireResourceLimits: true
    disallowPrivileged: true
    disallowRoot: true
    disallowLatestTag: true
    maxCPULimit: "8"
    maxMemoryLimit: "64Gi"
```

### LLM-specific SLO example

```yaml
apiVersion: kubeguard.bukx.dev/v1alpha1
kind: ServiceLevelObjective
metadata:
  name: llm-latency-slo
  namespace: llm-prod
spec:
  service: llm-gateway
  description: "P95 latency under 2.5s for GPT-style completions"
  target: "99"
  window: 7d
  indicator:
    type: latency
    totalQuery: 'sum(rate(llm_requests_total{service="llm-gateway"}[5m]))'
    errorQuery: 'sum(rate(llm_requests_total{service="llm-gateway",status="error"}[5m]))'
```

## Next steps

1. Create dedicated namespaces per model or tenant with `ManagedNamespace`.
2. Define latency/availability SLOs per endpoint.
3. Apply PodGuard policies that match GPU pod requirements.
4. Use generated PrometheusRules to alert on degradation before it becomes an outage.

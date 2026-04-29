# Heimdall Configuration Recipes

Complete `heimdall-values.yaml` examples for common deployment patterns.
Copy a recipe, replace `<...>` placeholders, and save as your `heimdall-values.yaml`.

**Verification status:**
- **[verified]** — Directly sourced from docs, test configs, or chart defaults
- **[unverified]** — Constructed from plugin specs; functionally valid but not tested in production

---

## Recipe 1: Basic aggregate (quickstart) [verified]

Source: `website/docs/getting-started/quickstart.mdx`, `test/e2e/quality/config/heimdall-values.yaml.tmpl`

All pods are equal. Route to the pod with the shortest waiting queue.
Use for: getting started, small deployments, homogeneous pods.

```yaml
global:
  imagePullSecrets:
    - name: moreh-registry

config:
  apiVersion: inference.networking.x-k8s.io/v1alpha1
  kind: EndpointPickerConfig
  plugins:
    - type: single-profile-handler
    - type: queue-scorer
    - type: max-score-picker
  schedulingProfiles:
    - name: default
      plugins:
        - pluginRef: queue-scorer
        - pluginRef: max-score-picker

gateway:
  name: mif
  gatewayClassName: istio        # or kgateway

inferencePool:
  targetPorts:
    - number: 8000
```

---

## Recipe 2: PD-disaggregated with queue scoring [verified]

:::warning
`pd-profile-handler` is legacy and has been replaced by [`disagg-profile-handler`](../../../website/docs/reference/heimdall/plugins.mdx#disagg-profile-handler). Recipes 2 and 3 below use the canonical `disagg-profile-handler` with the nested `profiles.*` / `deciders.*` schema.
:::

Source: `test/e2e/performance/config/heimdall-values.yaml.tmpl`

Separate prefill and decode pods. Each phase gets its own scheduling profile.
Pods must have label `mif.moreh.io/role` set to `prefill`, `decode`, or `both`.
Use for: large-scale deployments with distinct prefill/decode resource profiles.

```yaml
global:
  imagePullSecrets:
    - name: moreh-registry

config:
  apiVersion: inference.networking.x-k8s.io/v1alpha1
  kind: EndpointPickerConfig
  plugins:
    - type: always-disagg-pd-decider  # must precede disagg-profile-handler (factory-time lookup)
    - type: disagg-headers-handler    # must precede disagg-profile-handler (factory-time lookup)
    - type: prefill-filter
    - type: decode-filter
    - type: queue-scorer
    - type: max-score-picker
    - type: disagg-profile-handler
      parameters:
        profiles:
          prefill: prefill
          decode: decode
        deciders:
          prefill: always-disagg-pd-decider
  schedulingProfiles:
    - name: prefill
      plugins:
        - pluginRef: prefill-filter
        - pluginRef: queue-scorer
        - pluginRef: max-score-picker
    - name: decode
      plugins:
        - pluginRef: decode-filter
        - pluginRef: queue-scorer
        - pluginRef: max-score-picker

gateway:
  name: mif
  gatewayClassName: istio

inferencePool:
  targetPorts:
    - number: 8000
```

---

## Recipe 3: Production PD with KV cache awareness [verified]

Source: `deploy/helm/heimdall/values.yaml` (chart default config), `website/versioned_docs/version-v0.0.0/reference/heimdall_scheduler.md`

PD-disaggregated with `kv-cache-utilization-scorer` and optional saturation detection.
This is the Helm chart's default configuration.
Use for: production environments with varying workloads.

```yaml
global:
  imagePullSecrets:
    - name: moreh-registry

config:
  apiVersion: inference.networking.x-k8s.io/v1alpha1
  kind: EndpointPickerConfig
  plugins:
    - type: always-disagg-pd-decider  # must precede disagg-profile-handler (factory-time lookup)
    - type: disagg-headers-handler    # must precede disagg-profile-handler (factory-time lookup)
    - type: prefill-filter
    - type: decode-filter
    - type: queue-scorer
    - type: kv-cache-utilization-scorer
    - type: max-score-picker
    - type: disagg-profile-handler
      parameters:
        profiles:
          prefill: prefill
          decode: decode
        deciders:
          prefill: always-disagg-pd-decider
  schedulingProfiles:
    - name: prefill
      plugins:
        - pluginRef: prefill-filter
        - pluginRef: queue-scorer
          weight: 1
        - pluginRef: kv-cache-utilization-scorer
          weight: 1
        - pluginRef: max-score-picker
    - name: decode
      plugins:
        - pluginRef: decode-filter
        - pluginRef: queue-scorer
          weight: 1
        - pluginRef: kv-cache-utilization-scorer
          weight: 1
        - pluginRef: max-score-picker
  # Optional: saturation detection (not in chart defaults)
  # saturationDetector:
  #   queueDepthThreshold: 128
  #   kvCacheUtilThreshold: 0.9
  #   metricsStalenessThreshold: 30s

gateway:
  name: mif
  gatewayClassName: istio

serviceMonitor:
  labels:
    release: <prometheusStackRelease>

inferencePool:
  targetPorts:
    - number: 8000
```

---

## Recipe 4: Prefix-cache-aware with session affinity [unverified]

Constructed from plugin specs. Combines prefix locality with session stickiness for
multi-turn workloads. Weight values are illustrative and should be tuned based on
your workload characteristics.
Requires vLLM `--enable-prefix-caching` and client `x-session-token` management.
Weights: session affinity (5) > prefix locality (3) > queue depth (1).
Use for: chatbot / multi-turn conversational applications.

```yaml
global:
  imagePullSecrets:
    - name: moreh-registry

config:
  apiVersion: inference.networking.x-k8s.io/v1alpha1
  kind: EndpointPickerConfig
  plugins:
    - type: single-profile-handler
    - type: queue-scorer
    - type: prefix-cache-scorer
    - type: session-affinity-scorer
    - type: max-score-picker
  schedulingProfiles:
    - name: default
      plugins:
        - pluginRef: queue-scorer
          weight: 1
        - pluginRef: prefix-cache-scorer
          weight: 3
        - pluginRef: session-affinity-scorer
          weight: 5
        - pluginRef: max-score-picker

gateway:
  name: mif
  gatewayClassName: istio

inferencePool:
  targetPorts:
    - number: 8000
```

---

## Recipe 5: LoRA affinity routing [unverified]

Constructed from plugin specs. Routes requests to pods that already have the required
LoRA adapter loaded, reducing adapter swap overhead. Weight values are illustrative
and should be tuned based on your workload characteristics.
Use for: multi-LoRA serving with adapter-aware routing.

```yaml
global:
  imagePullSecrets:
    - name: moreh-registry

config:
  apiVersion: inference.networking.x-k8s.io/v1alpha1
  kind: EndpointPickerConfig
  plugins:
    - type: single-profile-handler
    - type: queue-scorer
    - type: lora-affinity-scorer
    - type: max-score-picker
  schedulingProfiles:
    - name: default
      plugins:
        - pluginRef: queue-scorer
          weight: 1
        - pluginRef: lora-affinity-scorer
          weight: 3
        - pluginRef: max-score-picker

gateway:
  name: mif
  gatewayClassName: istio

inferencePool:
  targetPorts:
    - number: 8000
```

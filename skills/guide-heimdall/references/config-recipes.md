# Heimdall Configuration Recipes

Complete `SchedulingProfile` examples for common routing patterns. Copy a recipe,
replace `<...>` placeholders, and `kubectl apply` it (SchedulingProfile is
cluster-scoped). Bind it from an `AIGateway` as shown at the end.

**Verification status:**
- **[verified]** — sourced from the docs or the product example CRs
- **[unverified]** — constructed from the plugin catalog; functionally valid but not tested here

The available plugins are generated from source — see `website/docs/reference/heimdall/plugins.mdx` for the authoritative catalog.

---

## Recipe 1: End-to-end, in-flight scoring (quickstart) [verified]

Source: `website/docs/getting-started/quickstart.mdx`

All pods serve both roles. Score by in-flight requests and pick the highest.
Use for: getting started, small deployments, homogeneous pods.

```yaml
apiVersion: heimdall.moreh.io/v1alpha1
kind: SchedulingProfile
metadata:
  name: e2e-basic
spec:
  profileHandler: e2e
  plugins:
    - type: inflight-requests-scorer
    - type: max-score-picker
  profiles:
    default:
      pluginRefs:
        - name: inflight-requests-scorer
          weight: 100
        - name: max-score-picker
```

---

## Recipe 2: Prefill/decode disaggregated [verified]

Source: heimdall-aigateway `tests/scenarios/routing-pd-dp/pd-profile.yaml`

Separate prefill and decode pods. The `role-filter` is applied internally per
sub-profile — you declare only scorers and a picker. Each prefill pod must carry
`mif.moreh.io/role: prefill` and each decode pod `mif.moreh.io/role: decode` (the
label is set on the `InferenceService`, directly or via its prefill/decode preset).
Use for: large-scale deployments with distinct prefill/decode resource profiles.

```yaml
apiVersion: heimdall.moreh.io/v1alpha1
kind: SchedulingProfile
metadata:
  name: pd-basic
spec:
  profileHandler: pd
  plugins:
    - type: inflight-requests-scorer
    - type: waiting-requests-scorer
    - type: kv-utilization-scorer
    - type: max-score-picker
  profiles:
    prefill:
      pluginRefs:
        - name: inflight-requests-scorer
          weight: 50
        - name: waiting-requests-scorer
          weight: 30
        - name: kv-utilization-scorer
          weight: 20
        - name: max-score-picker
    decode:
      pluginRefs:
        - name: inflight-requests-scorer
          weight: 50
        - name: waiting-requests-scorer
          weight: 30
        - name: kv-utilization-scorer
          weight: 20
        - name: max-score-picker
```

---

## Recipe 3: End-to-end with KV-cache awareness [unverified]

Adds `kv-cache-utilization-scorer` so requests avoid pods under KV-cache pressure.
Weights are illustrative — tune to your workload.
Use for: workloads where KV-cache occupancy varies across pods.

```yaml
apiVersion: heimdall.moreh.io/v1alpha1
kind: SchedulingProfile
metadata:
  name: e2e-kv
spec:
  profileHandler: e2e
  plugins:
    - type: inflight-requests-scorer
    - type: kv-cache-utilization-scorer
    - type: max-score-picker
  profiles:
    default:
      pluginRefs:
        - name: inflight-requests-scorer
          weight: 1
        - name: kv-cache-utilization-scorer
          weight: 1
        - name: max-score-picker
```

---

## Recipe 4: End-to-end, prefix-cache-aware [unverified]

Adds `prefix-cache-scorer` for prompts that share prefixes (system prompts,
few-shot templates). The block size comes from the pod's `InferenceWorker`
(`modelCard.kvCacheBlockSize`) — it is not set on the scorer.
Weights are illustrative — tune to your workload.
Use for: shared-prefix / templated-prompt workloads.

```yaml
apiVersion: heimdall.moreh.io/v1alpha1
kind: SchedulingProfile
metadata:
  name: e2e-prefix
spec:
  profileHandler: e2e
  plugins:
    - type: inflight-requests-scorer
    - type: prefix-cache-scorer
      config:
        normalization: longestPrefix
        transform: logistic
        k: 14.0
        x0: 0.7
    - type: max-score-picker
  profiles:
    default:
      pluginRefs:
        - name: inflight-requests-scorer
          weight: 1
        - name: prefix-cache-scorer
          weight: 3
        - name: max-score-picker
```

---

## Binding a profile from an AIGateway

A `SchedulingProfile` takes effect only when an `AIGateway` binds a model to it
(`default` is the gateway-wide fallback):

```yaml
apiVersion: heimdall.moreh.io/v1alpha1
kind: AIGateway
metadata:
  name: mif
spec:
  replicas: 1
  schedulingProfiles:
    - model: default
      profile: e2e-basic          # the SchedulingProfile name
    # - { model: <modelName>, profile: <otherProfile> }   # optional per-model override
```

Then bind inference pods to the gateway with the `mif.moreh.io/aigateway: mif`
label on the `InferenceService` (add `mif.moreh.io/role: prefill|decode` in pd mode).

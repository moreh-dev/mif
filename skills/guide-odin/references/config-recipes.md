# Odin InferenceService Configuration Recipes

Complete `InferenceService` and `InferenceServiceTemplate` manifest examples.
Copy a recipe, replace `<...>` placeholders, and apply with `kubectl apply -f`.

**Verification status:**
- **[verified]** — Directly sourced from docs, test configs, or chart defaults
- **[unverified]** — Constructed from API specs; functionally valid but not tested in production

---

## Recipe 1: Simple aggregate with preset [verified]

Source: `website/docs/getting-started/quickstart.mdx`

Deploys 2 replicas using a preset. Creates a Deployment workload (no LWS).
Requires: HuggingFace token with model license accepted.

```yaml
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: vllm-llama3-1b-instruct-tp2
  labels:
    mif.moreh.io/aigateway: mif
spec:
  replicas: 2
  templateRefs:
    - name: vllm
    - name: vllm-meta-llama-llama-3.2-1b-instruct-amd-mi250-tp2
  parallelism:
    tensor: 2
  template:
    spec:
      containers:
        - name: main
          env:
            - name: HF_TOKEN
              value: <huggingFaceToken>
```

---

## Recipe 2: Data-parallel decode with runtime-base [verified]

Source: `website/docs/features/preset.mdx`

Uses `vllm-decode-dp` runtime-base. Creates a LeaderWorkerSet workload (`data > 1`).
Note: Uses `workerTemplate`, not `template`.

```yaml
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: my-custom-model
  labels:
    mif.moreh.io/aigateway: mif
spec:
  replicas: 1
  templateRefs:
    - name: vllm-decode-dp
  model:
    name: <modelName>
  parallelism:
    data: 2
    tensor: 1
  workerTemplate:
    spec:
      containers:
        - name: main
          env:
            - name: ISVC_EXTRA_ARGS
              value: >-
                --disable-uvicorn-access-log
                --no-enable-log-requests
                --quantization None
                --max-model-len 4096
            - name: HF_TOKEN
              value: <huggingFaceToken>
          resources:
            limits:
              amd.com/gpu: 1
            requests:
              amd.com/gpu: 1
      nodeSelector:
        moai.moreh.io/accelerator.vendor: amd
        moai.moreh.io/accelerator.model: <acceleratorModel>
      tolerations:
        - key: amd.com/gpu
          operator: Exists
          effect: NoSchedule
```

---

## Recipe 3: PD-disaggregated (prefill + decode) [verified]

Source: `test/e2e/performance/config/inference-service.yaml.tmpl`

Separate InferenceServices for prefill and decode phases. Both bind to the same
AIGateway via the `mif.moreh.io/aigateway` label. Requires an AIGateway with a `pd`
SchedulingProfile (`spec.profileHandler: pd`).

```yaml
# Prefill InferenceService
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: <name>-prefill
  labels:
    mif.moreh.io/aigateway: <gatewayName>
spec:
  replicas: <prefillReplicas>
  templateRefs:
    - name: vllm-prefill-dp        # or vllm-prefill for non-DP
    - name: <prefillPreset>
  parallelism:
    data: <dataSize>
    dataLocal: <dataLocalSize>
    tensor: <tensorSize>
    expert: true                    # for MoE models
  workerTemplate:
    spec:
      containers:
        - name: main
          env:
            - name: HF_TOKEN
              value: <huggingFaceToken>
---
# Decode InferenceService
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: <name>-decode
  labels:
    mif.moreh.io/aigateway: <gatewayName>
spec:
  replicas: <decodeReplicas>
  templateRefs:
    - name: vllm-decode-dp          # or vllm-decode for non-DP
    - name: <decodePreset>
  parallelism:
    data: <dataSize>
    dataLocal: <dataLocalSize>
    tensor: <tensorSize>
    expert: true
  workerTemplate:
    spec:
      containers:
        - name: main
          env:
            - name: HF_TOKEN
              value: <huggingFaceToken>
```

---

## Recipe 4: Offline model from PersistentVolume [verified]

Source: `website/docs/operations/hf-model-management-with-pv.mdx`

Pre-download models to a RWX PVC, serve offline (no HF Hub access).
Requires: PVC `models` with downloaded model weights.

```yaml
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: vllm-offline
  labels:
    mif.moreh.io/aigateway: mif
spec:
  replicas: 2
  templateRefs:
    - name: vllm
    - name: <preset>
  template:
    spec:
      containers:
        - name: main
          env:
            - name: HF_HOME
              value: /mnt/models
            - name: HF_HUB_OFFLINE
              value: "1"
          volumeMounts:
            - name: models
              mountPath: /mnt/models
      volumes:
        - name: models
          persistentVolumeClaim:
            claimName: models
```

---

## Recipe 5: Custom reusable preset [verified]

Source: `website/docs/features/preset.mdx`

Create a reusable `InferenceServiceTemplate` from a custom configuration.
Apply to namespace, then reference from `InferenceService`.

```yaml
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceServiceTemplate
metadata:
  name: custom-prefill-dp16ep
spec:
  model:
    name: <modelName>
  parallelism:
    data: 16
    dataLocal: 8
    expert: true
  workerTemplate:
    spec:
      containers:
        - name: main
          env:
            - name: ISVC_EXTRA_ARGS
              value: >-
                --disable-uvicorn-access-log
                --no-enable-log-requests
          resources:
            limits:
              amd.com/gpu: "8"
            requests:
              amd.com/gpu: "8"
      nodeSelector:
        moai.moreh.io/accelerator.vendor: amd
        moai.moreh.io/accelerator.model: <acceleratorModel>
      tolerations:
        - key: amd.com/gpu
          operator: Exists
          effect: NoSchedule
---
# Usage: reference alongside the runtime-base
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: my-prefill
  labels:
    mif.moreh.io/aigateway: mif
spec:
  replicas: 1
  templateRefs:
    - name: vllm-prefill-dp
    - name: custom-prefill-dp16ep
  workerTemplate:
    spec:
      containers:
        - name: main
          env:
            - name: HF_TOKEN
              value: <huggingFaceToken>
```

---

## Recipe 6: Pipeline parallel deployment [unverified]

Constructed from API specs. Uses pipeline parallelism to split the model across
multiple pods (stages). Requires `vllm-pp` or `vllm-decode-pp` runtime-base.
Each pipeline group has `pipeline` workers.

```yaml
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: vllm-pipeline
  labels:
    mif.moreh.io/aigateway: mif
spec:
  replicas: 1
  templateRefs:
    - name: vllm-pp
  model:
    name: <modelName>
  parallelism:
    pipeline: 4
    tensor: 2
  workerTemplate:
    spec:
      containers:
        - name: main
          env:
            - name: HF_TOKEN
              value: <huggingFaceToken>
          resources:
            limits:
              amd.com/gpu: 2
            requests:
              amd.com/gpu: 2
      nodeSelector:
        moai.moreh.io/accelerator.vendor: amd
        moai.moreh.io/accelerator.model: <acceleratorModel>
      tolerations:
        - key: amd.com/gpu
          operator: Exists
          effect: NoSchedule
```

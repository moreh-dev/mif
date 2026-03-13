# Helm Charts — Agent Rules

Rules specific to the `deploy/helm/` directory. General contribution guidelines are in the root [`AGENTS.md`](/AGENTS.md).

## Design Principles

### Minimum Necessary Complexity

- **Do not add configuration options, fields, or abstractions for hypothetical future use cases.** Only add what the current task concretely requires.
- Before introducing a new value field, ask: "Is there a real, current use case that cannot be handled without it?" If the answer is no, omit the field and handle the edge case through documentation instead.
- Example: when considering whether to add a `minio.externalHost` field to support cross-namespace MinIO, the right answer was to document that users can point `loki.storage.s3.endpoint` to the external host directly — no new field needed.

### Documentation over Code for Edge Cases

- When a behavior difference only arises in a non-default, edge-case configuration, prefer documenting the workaround over adding a dedicated code path or configuration key.
- Reserve code changes for cases where the default path is broken or the workaround is genuinely error-prone.

### Reject Designs Before They Are Built

- If an initial design is heading in the wrong direction (e.g., standalone prerequisites instead of sub-chart dependencies, `enabled: false` defaults, nested config instead of top-level sections), raise the issue and redesign before writing code. Retrofitting a wrong structure is always more costly.

## Helm Chart Development

### Sub-chart Integration

- **All infrastructure components belong as sub-chart dependencies** of `moai-inference-framework`. Do not design them as standalone prerequisites that users install separately.
- **Enablement convention**: Every sub-chart dependency must have both a `condition:` entry in `Chart.yaml` AND `enabled: true` in the default `values.yaml`. Setting `enabled: false` as the default breaks the "install everything in one chart" philosophy. Follow the same pattern as existing components (`keda`, `lws`, `odin`, etc.).

  ```yaml
  # Chart.yaml — always add condition: and use the official repository
  - name: vector
    version: 0.39.0
    repository: https://helm.vector.dev
    condition: vector.enabled

  # values.yaml — always default to true
  vector:
    enabled: true
  ```

- **Official repositories**: Always use the chart's official upstream repository, not a mirror.
  - loki: `https://grafana.github.io/helm-charts`
  - vector: `https://helm.vector.dev`
  - minio: `https://charts.min.io`

### Dynamic Service Name References

- **Do not use `fullnameOverride`** to fix service names. Instead, build references using `.Release.Name` so that names are always consistent with whatever release name the user chooses.

  ```yaml
  # templates/grafana/datasource-loki.yaml
  url: http://{{ .Release.Name }}-loki-gateway.{{ include "common.names.namespace" . }}.svc.cluster.local

  # templates/loki/credentials.yaml
  BUCKET_HOST: {{ printf "%s-minio" .Release.Name | quote }}
  ```

- In sub-chart `customConfig` values rendered through `tpl`, use `{{ .Release.Name }}` directly — it is evaluated by the sub-chart's `tpl` call and resolves to the parent release name.

  ```yaml
  # values.yaml (vector customConfig) — .Release.Name evaluated by tpl
  endpoint: "http://{{ .Release.Name }}-loki-gateway"
  ```

### Separation of Concerns in values.yaml

- **Large infrastructure components must be top-level sections**, not nested under their consumers. For example, MinIO configuration belongs at `minio:`, not at `loki.minio:`. This allows MinIO to be independently enabled/disabled and reused by other components in the future.

### MinIO Provisioning Pattern

- Use the `minio/minio` chart (`https://charts.min.io`), not the bitnami chart.
- Create buckets, users, and policies directly via the chart's top-level `buckets`, `users`, and `policies` fields (not under a `provisioning` key).
- Create a **dedicated user per consuming service** with a policy scoped to only its bucket — do not use root credentials for service-to-service access.

  ```yaml
  minio:
    policies:
      - name: loki
        statements:
          - resources: ["arn:aws:s3:::loki/*"]
            effect: Allow
            actions: ["s3:*"]
    users:
      - accessKey: loki
        secretKey: "loki123!"
        policy: loki
    buckets:
      - name: loki
  ```

- Templates that read MinIO credentials must reference the `users` array directly:

  ```yaml
  # credentials.yaml
  stringData:
    AWS_ACCESS_KEY_ID:     {{ (index .Values.minio.users 0).accessKey | quote }}
    AWS_SECRET_ACCESS_KEY: {{ (index .Values.minio.users 0).secretKey | quote }}
  ```

- The MinIO subchart's post-install hooks for bucket/user/policy creation cause a **deadlock** when a consumer (e.g., Loki) depends on the bucket at startup: the consumer waits for the bucket, Helm waits for the consumer, and the hook never fires. To break this, provision buckets, users, and policies via a **regular Job** (not a Helm hook) that runs as soon as MinIO becomes reachable. Define the resources in `values.yaml` under `minio.buckets`, `minio.users`, and `minio.policies` as usual, but the init Job — not the subchart hooks — is responsible for actually creating them.

### Helm `tpl` Passthrough — Vector Label Syntax

- The vector chart renders `customConfig` through Helm's `tpl` function (`{{ tpl (toYaml .Values.customConfig) . | indent 4 }}`). This means any `{{ }}` expression in `customConfig` is evaluated as a Go template at render time.
- To pass **Vector's own field-template syntax** (`{{ field }}`) through `tpl` without evaluation, use Go raw string literals:

  ```yaml
  # values.yaml — correct
  labels:
    namespace: "{{`{{ namespace }}`}}"

  # values.yaml — WRONG: tpl evaluates {{ namespace }} as a Go template function
  labels:
    namespace: "{{ namespace }}"
  ```

- **Before using `customConfig` with any sub-chart, always verify whether the chart applies `tpl` to it** by running `helm pull <chart> --version <ver> --untar` and inspecting the ConfigMap template.

### YAML Anchors

- **Do not use YAML anchors at the root level of `values.yaml`** (e.g., `_defaults: &defaults`). Helm treats unknown root-level keys as invalid and may emit warnings or errors. Instead, duplicate shared configuration explicitly for each component.

## Odin Presets (`moai-inference-preset`)

An Odin preset is a pair of Odin `InferenceServiceTemplate` resources — a **base template** (runtime base) and a **preset-specific template** — that together define how to deploy a Moreh vLLM pod. The base template defines how vLLM servers are launched and is shared across presets. The preset-specific template adds model-specific arguments, environment variables, resource requests, and disaggregation settings.

### Preset naming convention

Preset names follow the pattern:
`{image_tag}-{org_name}-{model_name}[-mtp][-prefill][-decode]-{accelerator_vendor}-{accelerator_model}-{parallelism}[-moe-{moe_parallelism}]`

- `{org_name}` and `{model_name}` follow Hugging Face Hub names in kebab-case (e.g., `meta-llama/Llama-3.3-70B-Instruct` → `meta-llama-llama-3.3-70b-instruct`).
- `-mtp` is appended after `{model_name}` if multi-token prediction is used.
- `-prefill` or `-decode` is appended for disaggregation modes, placed after `{model_name}` (or `-mtp`) and before `{accelerator_vendor}`.
- `{parallelism}` examples: `1`, `tp4`, `tp8`, `dp8`. Canonical order for combined strategies: `dp` → `pp` → `tp` → `cp`.
- For MoE models, `-moe-{moe_parallelism}` is appended (e.g., `-moe-ep8`, `-moe-tp8`).

### Reserved labels

Odin presets use `mif.moreh.io/*` labels:

| Label key                         | Description                  | Example values                          |
| :-------------------------------- | :--------------------------- | :-------------------------------------- |
| `mif.moreh.io/template.type`      | Template type                | `runtime-base`, `preset`                |
| `mif.moreh.io/model.org`          | HF org name (kebab-case)     | `meta-llama`, `deepseek-ai`             |
| `mif.moreh.io/model.name`         | HF model name (kebab-case)   | `llama-3.3-70b-instruct`, `deepseek-r1` |
| `mif.moreh.io/model.mtp`          | Multi-token prediction       | `"true"` or unset                       |
| `mif.moreh.io/role`               | Disaggregation mode          | `e2e`, `prefill`, `decode`              |
| `mif.moreh.io/accelerator.vendor` | GPU vendor                   | `amd`                                   |
| `mif.moreh.io/accelerator.model`  | GPU model                    | `mi250`, `mi300x`, `mi308x`             |
| `mif.moreh.io/parallelism`        | Parallelism mode             | `tp4`, `dp8-moe-ep8`                    |

### Responsibility boundaries

**Presets define** (model/GPU-specific, not user-configurable):
- vLLM arguments for parallelism within a single rank (`--tensor-parallel-size`, `--enable-expert-parallel`, etc.)
- Model-specific vLLM arguments (`--trust-remote-code`, `--max-model-len`, `--max-num-seqs`, `--kv-cache-type`, `--quantization`, `--gpu-memory-utilization`, etc.)
- Logging arguments (`--disable-uvicorn-access-log`, `--no-enable-log-requests`) — presets must include these because `ISVC_EXTRA_ARGS` in a preset fully overrides the runtime base's value during Odin strategic merge patch (env vars merge by `name` key)
- Model-specific environment variables (`VLLM_ROCM_USE_AITER`, `VLLM_MOE_DP_CHUNK_SIZE`, `UCX_*`, `NCCL_*`, etc.)
- Resources (GPU count, RDMA NICs), tolerations, and nodeSelector

**Runtime bases define** (shared across presets):
- `spec.framework` (e.g., `vllm`)
- Execution command(s) and launch logic (for-loop for DP, cleanup traps)
- Cross-rank parallelism arguments (`--data-parallel-rank`, `--data-parallel-address`, `--data-parallel-rpc-port`)
- Disaggregation-specific environment variables (`VLLM_NIXL_SIDE_CHANNEL_HOST`, `VLLM_IS_DECODE_WORKER`)
- Shared memory settings, readiness probes
- Proxy sidecar configuration (for PD disaggregation)

**Users configure** (not defined by presets or runtime bases):
- Image repository and tag (with default provided)
- Volume mounts and model loading method (HF download vs. PV)
- Hugging Face token
- Number of replicas
- `--no-enable-prefix-caching`

**Product team templates configure** (must NOT be set in presets):
- `--prefix-caching-hash-algo`, `--kv-events-config`, `--block-size`

### PD decode proxy response headers

- `heimdall-proxy --response-header` is a debug flag that adds `X-Decoder-Host-Port` and `X-Prefiller-Host-Port` to responses.
- **Sim decode utils** (`sim-decode`, `sim-decode-dp`) include `--response-header` by default because they are debug-only templates.
- **Production runtime-bases** (`vllm-decode`, `vllm-decode-dp`, `vllm-decode-pp`) do **not** set `--response-header` — users opt in via the decode `InferenceService` by overriding the proxy's `ISVC_EXTRA_ARGS`.
- When `--response-header` is used, Heimdall's `response-header-handler` plugin is redundant.

### MIF Pod Label Keys

When filtering or labeling logs, metrics, or other signals by MIF-specific pod attributes, use these label keys:

| Concept           | Label key                    | Example value       |
| :---------------- | :--------------------------- | :------------------ |
| Pool              | `mif.moreh.io/pool`          | `heimdall`          |
| Role              | `mif.moreh.io/role`          | `prefill`, `decode` |
| App name          | `app.kubernetes.io/name`     | `vllm`              |
| Inference service | `app.kubernetes.io/instance` | `llama-3-2-1b`      |

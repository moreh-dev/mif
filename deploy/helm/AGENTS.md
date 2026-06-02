# Helm Charts — Agent Rules

Rules specific to `deploy/helm/`. General contribution guidelines are in the root [`AGENTS.md`](/AGENTS.md).

## Principles

- **YAGNI** — Add no value field or abstraction without a current, concrete use case. Prefer documenting workarounds over new code paths for non-default edge cases.
- **Reject wrong designs early** — Standalone prerequisites, `enabled: false` defaults, deeply-nested config instead of top-level sections — redesign before writing code, never retrofit.

## Verification

After any chart change, run the narrowest sufficient check: `make helm-lint`, `helm lint <chart>`, or `helm template <chart>` with representative values; `make helm-dependency` when `Chart.yaml` deps change; `make helm-docs` when values/docs templates change. Don't claim a change complete without at least one render- or lint-level step; if skipped, state which and why.

## Sub-chart integration

- Every infrastructure component is a sub-chart of `moai-inference-framework`, never a standalone prerequisite. Default to `enabled: true` with a `condition:` entry in `Chart.yaml` — `enabled: false` defaults break the one-chart philosophy.
- Use official upstream repos: loki `https://grafana.github.io/helm-charts`, vector `https://helm.vector.dev`, minio `https://charts.min.io`.

## Naming and references

- No `fullnameOverride`. Build service refs from `{{ .Release.Name }}-<svc>.{{ include "common.names.namespace" . }}.svc.cluster.local`. Inside a sub-chart's `customConfig` rendered through `tpl`, `{{ .Release.Name }}` resolves to the **parent** release name.
- Large infra components are top-level keys (`minio:`, not `loki.minio:`) so they can be reused.
- No YAML anchors at the root of `values.yaml` — Helm rejects unknown root keys.

## Sub-chart `tpl` passthrough

Before using a sub-chart's `customConfig`, verify whether the chart wraps it with `tpl` (`helm pull <chart> --version <ver> --untar`, inspect the ConfigMap template). If yes, escape the target tool's own `{{ }}` syntax with Go raw string literals to prevent Helm from evaluating it — e.g. Vector labels: `"{{`{{ namespace }}`}}"`.

## MinIO

Use `minio/minio`. Declare buckets in the top-level `minio.buckets`. Do **not** set the sub-chart's `minio.users`/`minio.policies`: those provision through the sub-chart's post-install Helm hook, which deadlocks when a consumer (e.g. Loki, Tempo) needs its bucket and credentials at startup — the consumer waits for the bucket, Helm waits for the consumer, and the hook never fires.

Instead, the regular `templates/minio/init-job.yaml` Job (not a hook) provisions storage as soon as MinIO is reachable. For each enabled consumer it creates the bucket, a least-privilege policy scoped to that bucket, and a dedicated user (never the root user), via `mc admin`. Per-consumer credentials live in a top-level `<consumer>Bucket` section (e.g. `lokiBucket`, `tempoBucket`) and are surfaced through a `<consumer>-bucket` Secret + ConfigMap (`templates/<consumer>/credentials.yaml`); both the init Job and the consumer's pods read them via `extraEnvFrom` + `-config.expand-env=true`.

## Alert provisioning

The chart provisions Grafana Unified Alerting through ConfigMaps labelled `grafana_alert=1`, mounted by the `grafana-sc-alerts` sidecar into `/etc/grafana/provisioning/alerting/`. Two groups:

- **Rules / templates / policies** — one ConfigMap per file under `files/alerts/*.yaml`, emitted verbatim by `alert-configmap.yaml`. Do **not** wrap with `tpl` (alert YAML embeds Grafana's own `{{ }}` syntax). Reference Grafana URLs via Grafana's `{{ externalURL }}` template, not a chart-side placeholder.
- **Heimdall Slack contact point** — `heimdall-slack-configmap.yaml`. URL resolves from `alerts.heimdall.slack.existingSecret` + `slack.secretKeys.webhookUrlKey` (Bitnami secret-reference convention) via Helm `lookup`, falling back to inline `slack.webhookUrl`. `helm template` / `--dry-run` cannot read cluster state, so `existingSecret` renders empty under them — verify against a real cluster.

Operators must set `prometheus-stack.grafana.grafana.ini.server.root_url` for Slack links to work; otherwise Grafana falls back to `http://localhost:3000`.

## Odin presets (`moai-inference-preset`)

A preset is a pair of `InferenceServiceTemplate` resources — a **runtime base** (shared launcher) and a **preset-specific template** (model/GPU args).

**Naming**: `{image_tag}-{org}-{model}[-mtp][-prefill|-decode]-{vendor}-{accel}-{parallelism}[-moe-{moe_par}]`. Org/model in HF kebab-case. Combined parallelism order: `dp` → `pp` → `tp` → `cp`. MoE adds `-moe-{ep|tp}N`.

**Responsibility split**:

- Runtime bases — `spec.framework`, launch command and parallelism flag assembly (`--tensor-parallel-size` etc.), disaggregation env (`VLLM_NIXL_SIDE_CHANNEL_HOST`, `VLLM_IS_DECODE_WORKER`), shm/readiness, PD proxy sidecar.
- Presets — `spec.parallelism` values, model-specific vLLM args (`--max-model-len`, `--gpu-memory-utilization`, …), logging args (must repeat — `ISVC_EXTRA_ARGS` is fully overridden, not merged), model-specific env, resources/tolerations/nodeSelector.
- Utils (`*-hf-hub-offline` templates) — offline HF cache env (`HF_HOME`, `HF_HUB_OFFLINE`, `HF_MODULES_CACHE`), shared by runtime bases and presets.
- Users — image tag, replicas, volumes / model loading method, HF token, `--no-enable-prefix-caching`.

**Reserved `mif.moreh.io/*` labels**:

| Key | Example |
| --- | --- |
| `template.type` | `runtime-base`, `preset` |
| `model.org` / `model.name` | `meta-llama` / `llama-3.3-70b-instruct` |
| `model.mtp` | `"true"` |
| `role` | `e2e`, `prefill`, `decode` |
| `accelerator.vendor` / `accelerator.model` | `amd` / `mi300x` |
| `parallelism` | `tp4`, `dp8-moe-ep8` |
| `pool` | `heimdall` |

Logs/metrics filtering also uses stock Kubernetes labels `app.kubernetes.io/name` (e.g. `vllm`) and `app.kubernetes.io/instance` (inference service name).

**PD decode proxy** — `heimdall-proxy --response-header` is a debug flag. Sim decode utils (`sim-decode*`) default it on; production runtime-bases (`vllm-decode*`) leave it off and users opt in via the decode `InferenceService`. When the flag is set, Heimdall's `response-header-handler` plugin is redundant.

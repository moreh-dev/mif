# Helm Charts — Agent Rules

Rules specific to `deploy/helm/`. General contribution guidelines are in the root [`AGENTS.md`](/AGENTS.md).

## Principles

- **YAGNI** — Add no value field or abstraction without a current, concrete use case. Prefer documenting workarounds over new code paths for non-default edge cases.
- **Reject wrong designs early** — Standalone prerequisites, `enabled: false` defaults, deeply-nested config instead of top-level sections — redesign before writing code, never retrofit.

## Scope of this directory

`moai-inference-framework` is the only chart in this repository. Odin presets and runtime-bases (`InferenceServiceTemplate` resources) are **not** packaged here — they are installed into the `mif` namespace by the cluster administrator, either manually or by Odin from S3. Do not reintroduce a preset chart, and do not add preset or runtime-base template paths to docs or skills. When documentation needs to show a preset, direct readers to inspect the templates installed in their cluster:

```shell
kubectl get inferenceservicetemplate -n mif -l mif.moreh.io/template.type=preset
```

## Verification

After any chart change, run the narrowest sufficient check: `make helm-lint`, `helm lint <chart>`, or `helm template <chart>` with representative values; `make helm-dependency` when `Chart.yaml` deps change; `make helm-docs` when values/docs templates change. Don't claim a change complete without at least one render- or lint-level step; if skipped, state which and why.

## Sub-chart integration

- Every infrastructure component is a sub-chart of `moai-inference-framework`, never a standalone prerequisite. Default to `enabled: true` with a `condition:` entry in `Chart.yaml` — `enabled: false` defaults break the one-chart philosophy.
- Use official upstream repos: loki `https://grafana.github.io/helm-charts`, vector `https://helm.vector.dev`, minio `https://charts.min.io`, tempo `https://grafana-community.github.io/helm-charts` (Grafana's community-maintained charts repo, which carries current `tempo-distributed`).
- Keep `prometheus-stack.defaultRules.create: true`. The bundled "Kubernetes / Compute Resources" Grafana dashboards (panels *and* their `workload`/`type` template variables) query kubernetes-mixin recording rules such as `namespace_workload_pod:kube_pod_owner:relabel`, so disabling default rules blanks those dashboards even though raw metrics are still scraped.

## Naming and references

- No `fullnameOverride`. Build service refs from `{{ .Release.Name }}-<svc>.{{ include "common.names.namespace" . }}.svc.cluster.local`. Inside a sub-chart's `customConfig` rendered through `tpl`, `{{ .Release.Name }}` resolves to the **parent** release name.
- Large infra components are top-level keys (`minio:`, not `loki.minio:`) so they can be reused.
- No YAML anchors at the root of `values.yaml` — Helm rejects unknown root keys.

## Sub-chart `tpl` passthrough

Before using a sub-chart's `customConfig`, verify whether the chart wraps it with `tpl` (`helm pull <chart> --version <ver> --untar`, inspect the ConfigMap template). If yes, escape the target tool's own `{{ }}` syntax with Go raw string literals to prevent Helm from evaluating it — e.g. Vector labels: `"{{`{{ namespace }}`}}"`.

## MinIO

Use `minio/minio`. Do **not** set the sub-chart's `minio.users`/`minio.policies`: those provision through the sub-chart's post-install Helm hook, which deadlocks when a consumer (e.g. Loki, Tempo) needs its bucket and credentials at startup — the consumer waits for the bucket, Helm waits for the consumer, and the hook never fires.

Instead, the regular `templates/minio/init-job.yaml` Job (not a hook) provisions storage as soon as MinIO is reachable. For each enabled consumer it creates the bucket itself, a least-privilege policy scoped to that bucket, and a dedicated user (never the root user), via `mc admin`. The top-level `minio.buckets` only optionally pre-creates buckets via the sub-chart; it is not required and does not drive this scoped per-consumer provisioning. Per-consumer credentials live in a top-level `<consumer>Bucket` section (e.g. `lokiBucket`, `tempoBucket`) and are surfaced through a `<consumer>-bucket` Secret + ConfigMap (`templates/<consumer>/credentials.yaml`); both the init Job and the consumer's pods read them via `extraEnvFrom` + `-config.expand-env=true`.

## Grafana dashboards

- AIGateway dashboard coverage audits must include first-party metrics from gateway request metrics, gateway exposition, and sidecar metrics. Use `rate` for counters, guard derived ratios against zero denominators, and aggregate replica-local series deliberately before grouping by model or endpoint.
- AIGateway metrics carry no `role` label; endpoint-role filtering joins Kubernetes metadata instead. `kube_pod_labels` exposes `label_mif_moreh_io_role` only because `prometheus-stack.kube-state-metrics.metricLabelsAllowlist` allowlists `pods=[mif.moreh.io/role]` — keep that allowlist and the dashboards in sync. Once a resource has any allowlist entry, KSM emits `kube_<resource>_labels` for **every** object of that resource (allowlisted labels appear only where set), so `=~".*"` keeps label-less objects while `=~".+"` or a specific value drops them.
- A Grafana template variable built from a PromQL join (anything that is not a plain series selector) must use `query_result(<expr>)` plus a `regex` such as `/endpoint="([^"]+)"/`; `label_values(<expr>, <label>)` resolves through the series/label-values API and only accepts selectors.

## Alert provisioning

The chart provisions Grafana Unified Alerting through ConfigMaps labelled `grafana_alert=1`, mounted by the `grafana-sc-alerts` sidecar into `/etc/grafana/provisioning/alerting/`. Two groups:

- **Rules / templates / policies** — one ConfigMap per file under `files/alerts/*.yaml`, emitted verbatim by `alert-configmap.yaml`. Do **not** wrap with `tpl` (alert YAML embeds Grafana's own `{{ }}` syntax). Reference Grafana URLs via Grafana's `{{ externalURL }}` template, not a chart-side placeholder.
- **Heimdall Slack contact point** — `heimdall-slack-configmap.yaml`. URL resolves from `alerts.heimdall.slack.existingSecret` + `slack.secretKeys.webhookUrlKey` (Bitnami secret-reference convention) via Helm `lookup`, falling back to inline `slack.webhookUrl`. `helm template` / `--dry-run` cannot read cluster state, so `existingSecret` renders empty under them — verify against a real cluster.

Operators must set `prometheus-stack.grafana.grafana.ini.server.root_url` for Slack links to work; otherwise Grafana falls back to `http://localhost:3000`.

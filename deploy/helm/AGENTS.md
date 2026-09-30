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

- The chart bundles only `common` and `kubernetes-replicator`. Observability (Prometheus, Loki, Tempo, Vector, MinIO, Grafana dashboards and alerts) and NFD are deployed separately by GitOps, not by this chart. Do not add them back.
- Add a sub-chart only with a `condition:` entry in `Chart.yaml` and official upstream repos.

## Naming and references

- No `fullnameOverride`. Build service refs from `{{ .Release.Name }}-<svc>.{{ include "common.names.namespace" . }}.svc.cluster.local`. Inside a sub-chart's `customConfig` rendered through `tpl`, `{{ .Release.Name }}` resolves to the **parent** release name.
- Large infra components are top-level keys (`minio:`, not `loki.minio:`) so they can be reused.
- No YAML anchors at the root of `values.yaml` — Helm rejects unknown root keys.

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

### MIF Pod Label Keys

When filtering or labeling logs, metrics, or other signals by MIF-specific pod attributes, use these label keys:

| Concept           | Label key                    | Example value       |
| :---------------- | :--------------------------- | :------------------ |
| Pool              | `mif.moreh.io/pool`          | `heimdall`          |
| Role              | `mif.moreh.io/role`          | `prefill`, `decode` |
| App name          | `app.kubernetes.io/name`     | `vllm`              |
| Inference service | `app.kubernetes.io/instance` | `llama-3-2-1b`      |

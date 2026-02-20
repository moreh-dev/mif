# Contribution Guide

This guide serves as the unified source of truth for all contributors, both human engineers and automation agents.

## Git Commit Guidelines

> [!NOTE]
> References:
>
> - [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/)

The Conventional Commits specification is a lightweight convention on top of commit messages. It provides an easy set of rules for creating an explicit commit history; which makes it easier to write automated tools on top of. This convention dovetails with SemVer, by describing the features, fixes, and breaking changes made in commit messages.

The commit message should be structured as follows:

```shell
<type>[(<scope>)][!]: <description>

[<body>]

[<footer>]
```

### Type

- `<type>(<scope>)!`: Breaking change. (major version bump except for `v0.x.x` where it bumps minor version)
- `feat`: New feature or enhancement. (minor version bump)
- `fix`: Bug fix. (patch version bump)
- `refactor`: Code change that neither fixes a bug nor adds a feature.
- `style`: Code style changes (whitespace, formatting, etc.) that do not affect functionality.
- `test`: Only test-related changes.
- `docs`: Only documentation changes.
- `chore`: Changes without a direct impact on the codebase (build process, dependencies, etc.).

### Scope

- `workflow`: Changes related to CI/CD workflows.
- `deploy`: Changes related to deployment (Helm charts, container files, etc.)
- `config`: Changes related to files hard to manage within helm charts.
- `preset`: Changes related to preset files.
- `website`: Changes related to website.
- `e2e`: Changes related to end-to-end tests.

## Code Style Guidelines

### Comments

- **Language**: All comments must be in English.
- **Goals**: Focus on **why** something is done, not what or how.
  - Explain non-obvious business logic, edge cases, or complex algorithms.
  - Avoid restating the obvious or detailing implementation steps if the code is clear.
- **API Documentation**: Exported functions should have concise doc comments explaining their purpose.

### Go Templates

- **Indentation**: Go template syntax (e.g., `{{- if ... }}`) must be indented to match the surrounding code context for better readability.

### E2E Test

- **Version scope**:
  - E2E tests cover only `vX.Y.Z` (release) and `vX.Y.Z-rc.N` (release candidate) version formats.
  - Other version formats (e.g. dev builds, custom tags) are out of scope and should not be tested in E2E.

- **Do not test resource specifications**:
  - Do not validate individual fields of the YAML file declaring the resource (resource spec).
  - Instead, create the resource and verify that its status reaches the expected state.

- **Assume fully controlled cluster**:
  - Do not check if components are already installed.
  - Assume the cluster is fully controlled by the test and installed components are safe to overwrite or delete.

- **Test suite layout**:
  - Split tests by purpose under `test/e2e`, for example `test/e2e/performance` and `test/e2e/quality`.
  - In each directory, define shared Ginkgo configuration (labels, timeouts, common hooks) in `suite_test.go`, and keep scenarios in separate `*_test.go` files.
  - Shared configuration values must come from the `test/utils/settings` package instead of hard-coded constants in test files.

- **Environment variable management**:
  - Manage all E2E environment variables centrally in `test/e2e/envs/env_vars.go`.
  - When a new environment variable is required:
    - Add it to the `envVars` slice with default value, description, category, and type.
    - Expose it via public variables (for example `TestModel`, `HFToken`) and access it only through those variables.
    - Do not call `os.Getenv` directly in test code.
  - Keep the documentation consistent: changes must pass the `validateEnvVars()` check.

- **Resource templates and settings**:
  - Manage Kubernetes resource specifications for Gateway, InferenceService, Jobs, and similar resources as Go templates (`.yaml.tmpl`) under `test/config/**`.
  - Tests must read template paths and default values from constants in `test/utils/settings/constants.go`.
  - When adding a new benchmark or performance test Job:
    - Add the template file under an appropriate `test/config/<domain>` subdirectory.
    - Define the corresponding path and default parameters in the `settings` package.

- **Utility reuse**:
  - Implement all cluster manipulation logic (namespace creation, Gateway create/delete, Heimdall install/uninstall, InferenceService(Template) create/delete, etc.) in the `test/utils` package and call only those helpers from tests.
  - Follow this pattern for scenario flow:
    - `BeforeAll`: create namespace → install Gateway → install Heimdall → create InferenceServiceTemplates → create InferenceServices → wait until they are Ready.
    - `AfterAll`: if `envs.SkipCleanup` is `false`, clean up the above resources in reverse order.
    - `It(...)`: render the Job template → create the Job with `kubectl create -f -` → wait for completion with `kubectl wait` → collect logs and perform domain-specific assertions.

- **Makefile and workflow integration**:
  - Provide separate Make targets per test purpose (for example `e2e-performance`, `e2e-quality`) so that CI can run them independently.
  - GitHub Actions and other workflows should invoke these targets directly, and new test categories should follow the same pattern when adding additional targets and workflows.

## Agent Self-Improvement

After completing any non-trivial task, evaluate whether the work involved:
- A recurring pattern that will likely appear again in future tasks, or
- A mistake that was corrected through user feedback, or
- A design decision that required deliberate reasoning to reach the right answer.

If any of the above applies, **record it in this file** (`AGENTS.md`) before closing the task. Entries should be concise, actionable, and placed under the most relevant existing section. If no section fits, create one.

The goal is to make every repeated task faster and every repeated mistake impossible.

## Design Principles

### Minimum Necessary Complexity

- **Do not add configuration options, fields, or abstractions for hypothetical future use cases.** Only add what the current task concretely requires.
- Before introducing a new value field, ask: "Is there a real, current use case that cannot be handled without it?" If the answer is no, omit the field and handle the edge case through documentation instead.
- Example: when considering whether to add a `minio.externalHost` field to support cross-namespace MinIO, the right answer was to document that users can set `minio.fullnameOverride` to a FQDN when `minio.enabled: false` — no new field needed.

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
  # Chart.yaml — always add condition:
  - name: vector
    version: 0.39.0
    repository: https://moreh-dev.github.io/helm-charts
    condition: vector.enabled

  # values.yaml — always default to true
  vector:
    enabled: true
  ```

### Predictable Service Names

- Set `fullnameOverride` on every sub-chart so that service names are deterministic regardless of the Helm release name. Without this, service names vary with the release name and break cross-component references.

  ```yaml
  minio:
    fullnameOverride: minio # service is always "minio"
  loki:
    fullnameOverride: loki # gateway is always "loki-gateway"
  ```

### Separation of Concerns in values.yaml

- **Large infrastructure components must be top-level sections**, not nested under their consumers. For example, MinIO configuration belongs at `minio:`, not at `loki.minio:`. This allows MinIO to be independently enabled/disabled and reused by other components in the future.

### MinIO Provisioning Pattern

- Use `provisioning` (not `defaultBuckets`) to create buckets, users, and policies. This allows fine-grained access control.
- Create a **dedicated user per consuming service** with a policy scoped to only its bucket — do not use root credentials for service-to-service access.

  ```yaml
  minio:
    provisioning:
      enabled: true
      policies:
        - name: loki
          statements:
            - resources: ["arn:aws:s3:::loki/*"]
              effect: Allow
              actions: ["s3:*"]
      users:
        - username: loki
          password: ""
          policies: [loki]
          setPolicies: true
      buckets:
        - name: loki
  ```

- Templates that read MinIO credentials must reference the **provisioned user**, not root:

  ```yaml
  # credentials.yaml
  stringData:
    AWS_ACCESS_KEY_ID:
      { { (index .Values.minio.provisioning.users 0).username | quote } }
    AWS_SECRET_ACCESS_KEY:
      { { (index .Values.minio.provisioning.users 0).password | quote } }
  ```

### Helm `tpl` Passthrough — Vector Label Syntax

- The moreh/vector chart renders `customConfig` through Helm's `tpl` function (`{{ tpl (toYaml .Values.customConfig) . | indent 4 }}`). This means any `{{ }}` expression in `customConfig` is evaluated as a Go template at render time.
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

### Avoiding Unnecessary Environment Variables

- Prefer static values or short DNS names over dynamically-injected environment variables when components are co-located in the same namespace.
- For example, a Vector sink targeting a Loki gateway in the same namespace should use `http://loki-gateway` (short name), not `http://loki-gateway.${VECTOR_NAMESPACE}.svc.cluster.local` with an injected namespace variable.

# Test — Agent Rules

Rules specific to the `test/` directory. General contribution guidelines are in the root [`AGENTS.md`](/AGENTS.md).

## E2E Test

- **Version scope**:
  - E2E tests cover only `vX.Y.Z` (release) and `vX.Y.Z-rc.N` (release candidate) version formats.
  - Other version formats (e.g. dev builds, custom tags) are out of scope and should not be tested in E2E.

- **Do not test resource specifications**:
  - Do not validate individual fields of the YAML file declaring the resource (resource spec).
  - Instead, create the resource and verify that its status reaches the expected state.

- **Cluster lifecycle is outside test code**:
  - Test code must never create or delete Kubernetes clusters.
  - Tests assume a valid kubeconfig already exists. The Makefile handles Kind cluster lifecycle (`setup-test-e2e`, `cleanup-test-e2e`).
  - Do not reference Kind directly in test code; use environment variables to control behavior differences between environments.

- **Hardcoded configuration, not environment-variable-driven**:
  - Fixed values (model names, template refs, S3 region/bucket, namespaces, gateway class) are hardcoded in `test/utils/settings/constants.go` or directly in each test file.
  - Only execution settings (`SKIP_*`), credentials (`AWS_*`, `S3_*`, `HF_*`), and environment-specific values (`WORKLOAD_NAMESPACE`, `ISTIO_REV`) remain as environment variables.
  - Each test category (smoke, performance, quality) hardcodes its own template refs and GPU/PVC settings. Smoke uses simulation images; performance and quality use real GPU images.
  - Do not use infrastructure-awareness flags like `IsKind` or `SimulationMode`.

- **Test suite layout**:
  - Split tests by purpose under `test/e2e`: `smoke`, `performance`, `quality`.
  - In each directory, define shared Ginkgo configuration (labels, timeouts, common hooks) in `suite_test.go`, and keep scenarios in separate `*_test.go` files.
  - Shared configuration values must come from the `test/utils/settings` package instead of hard-coded constants in test files.
  - Common suite setup/teardown logic (prerequisite installation) lives in `test/utils/setup/prerequisites.go`.

- **Environment variable management**:
  - Environment variables are only for: execution settings (`SKIP_PREREQUISITE`, `SKIP_CLEANUP`, `SKIP_SCORE_VALIDATION`), credentials (`AWS_*`, `S3_*`, `HF_*`), and environment-specific values (`WORKLOAD_NAMESPACE`, `ISTIO_REV`).
  - Manage all E2E environment variables centrally in `test/e2e/envs/env_vars.go`.
  - Do not add new environment variables for fixed configuration values. Instead, hardcode them in `test/utils/settings/constants.go` or in the test file that uses them.
  - Do not call `os.Getenv` directly in test code.

- **Resource templates and settings**:
  - Manage Kubernetes resource specifications for Gateway, InferenceService, Jobs, and similar resources as Go templates (`.yaml.tmpl`) under `test/config/**`.
  - Tests must read template paths and default values from constants in `test/utils/settings/constants.go`.
  - Template conditionals should be feature-driven (e.g. `{{ if .GPUResourcesEnabled }}`, `{{ if .ModelPVCName }}`, `{{ if .S3AccessKeyID }}`), not infrastructure-driven.
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
  - Provide separate Make targets per test purpose (`test-e2e-smoke`, `test-e2e-performance`, `test-e2e-quality`) so that CI can run them independently.
  - `test-e2e-kind` creates a Kind cluster, runs smoke tests (simulation images, no GPU), and cleans up.
  - GitHub Actions and other workflows should invoke these targets directly, and new test categories should follow the same pattern when adding additional targets and workflows.

# Test — Agent Rules

Rules specific to the `test/` directory. General contribution guidelines are in the root [`AGENTS.md`](/AGENTS.md).

## E2E Test

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

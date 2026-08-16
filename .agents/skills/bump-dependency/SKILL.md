---
name: bump-dependency
description: Guide for updating MIF dependency versions. Use this skill when asked to bump, update, or upgrade a dependency such as Odin, Odin-CRD, LWS, Heimdall, or moai-inference-framework. Covers which files to modify, how to run updates, and how to verify the result.
---

# Dependency Version Update Guide

## Overview

MIF depends on several components whose versions are tracked across Helm charts, container image references, and website documentation. This guide ensures all references are updated consistently when bumping a dependency. After every version bump, you **must** also update the version table in `website/docs/operations/latest-release.mdx`.

## Components

| Component                | Type              | Version Locations                                                                                              |
| :----------------------- | :---------------- | :------------------------------------------------------------------------------------------------------------- |
| Odin + Odin-CRD          | Helm sub-chart    | `deploy/helm/moai-inference-framework/Chart.yaml`                                                              |
| LWS                      | Helm sub-chart    | `deploy/helm/moai-inference-framework/Chart.yaml`                                                              |
| Heimdall                 | External Helm chart | `website/docs/getting-started/quickstart.mdx`                                                                  |
| moai-inference-framework | MIF chart release | `website/docs/getting-started/prerequisites.mdx`, `website/docs/getting-started/quickstart.mdx`                |

## Procedures

### 1. Helm Sub-chart Bump (Odin, Odin-CRD, LWS)

**Required information:** target version.

**Steps:**

1. Edit `deploy/helm/moai-inference-framework/Chart.yaml`:
   - Find the `dependencies` entry matching the component name.
   - Update the `version` field to the target version.
   - For **Odin**: always update both `odin` and `odin-crd` entries to the same version.
2. Run `make helm-dependency` to regenerate `Chart.lock` and download updated `.tgz` archives.
3. Run `make helm-docs` to regenerate chart README documentation.
4. Run `make helm-lint` to verify the chart is valid.
5. Search `website/docs/` for references to the updated component and update if needed (see [Website Updates for Spec Changes](#4-website-updates-for-spec-changes)).

**Adding a new sub-chart dependency:**

If a component is not yet listed in `Chart.yaml`, add a new entry following the existing pattern:

```yaml
- name: <component>
  version: <target-version>
  repository: <chart-repo-url> # ask user
  condition: <component>.enabled
```

Then add the corresponding enablement default in `values.yaml`:

```yaml
<component>:
  enabled: true
```

### 2. Heimdall Chart Bump

Heimdall (the AIGateway operator) is deployed as a separate Helm chart (`moreh/heimdall` from `https://moreh-dev.github.io/helm-charts`), not as a sub-chart of moai-inference-framework.

**Required information:** target version. If the new version includes config or API changes, also ask for the Heimdall source repository URL to review the changes.

**Steps:**

1. Update the Heimdall `--version` flags in `website/docs/getting-started/prerequisites.mdx`. That page pins several Heimdall charts separately, so update every one that changes for the target release — not just the operator: the operator chart (`helm upgrade -i heimdall moreh/heimdall`), the CRD chart (`helm upgrade -i heimdall-crd moreh/heimdall-crd`), and the AIGateway CRD chart (`helm upgrade -i heimdall-aigateway-crd moreh/heimdall-aigateway-crd`). Leaving a CRD chart version stale while bumping only the operator is the failure this step guards against.
2. Search `website/docs/` for other Heimdall version references and update them.
3. Clone the Heimdall source repo with `--recurse-submodules` and review what changed between the old and new version tags. Heimdall uses Git submodules for its core components, so check both the main repo diff (`git diff <old-tag>..<new-tag>`) and the submodule commit ranges for plugin or API changes. Use `git ls-tree <tag> third_party/` to get the submodule commit SHAs at each tag, then diff within each submodule.
4. **Verify plugin documentation against source structs.** Do not rely solely on the diff — diffs can miss pre-existing documentation gaps. For every plugin listed in `plugins.mdx`:
   - Locate the Go config struct in the source and compare **every field** against the documented parameters.
   - Check the **nesting depth**: fields belong to whichever struct declares them. If a field is in a nested struct (e.g., `kvblock.IndexConfig`), it must be documented under the corresponding nested header, not the parent.
   - Watch for **inline/embedded structs** (`json:",inline"`). Go's inline tag flattens fields from composed structs into the parent JSON, so these fields must appear in the parent's parameter table.
   - Confirm all **new plugins** registered in `register.go` (both the main repo and submodule repos) are documented, and any unregistered plugins are not.
5. Update the reference docs based on the changes found:
   - `website/docs/reference/heimdall/usage.mdx` — AIGateway, SchedulingProfile, and InferenceWorker CRD fields
   - `website/docs/reference/heimdall/plugins.mdx` — plugin parameters, new plugins, removed plugins
   - `website/docs/getting-started/prerequisites.mdx` — the `heimdall` install command/values if config structure changed
   - Other docs that reference Heimdall scheduling, routing, or load balancing behavior

### 3. MIF Chart Release Update (moai-inference-framework)

**Required information:** new chart release version (e.g., `v0.4.0`).

**Steps:**

1. Update version references in `website/docs/getting-started/prerequisites.mdx`:
   - `helm upgrade` commands that specify `--version`.
2. Update version references in `website/docs/getting-started/quickstart.mdx`:
   - Version badges / prerequisite version list.
3. Search `website/docs/` for any other references to the old version string.
4. Do **NOT** modify `website/versioned_docs/` — these are frozen snapshots of past versions.
5. Do **NOT** modify `Chart.yaml` `version` or `appVersion` fields — these are set by CI/CD during release.

### 4. Website Updates for Spec Changes

When a dependency introduces API, CRD, or configuration changes (not just a version number bump):

1. Ask the user what changed: new fields, removed fields, renamed APIs, behavior changes.
2. Search `website/docs/` for references to affected CRD kinds, field names, or config options:
   ```
   grep -r "<old-field-or-kind>" website/docs/
   ```
3. Update affected documentation: YAML examples, API reference pages, feature descriptions.
4. Common search targets by component:
   - **Odin CRD**: `InferenceService`, `InferenceServiceTemplate`, `templateRefs`, `inferencePoolRefs`
   - **Heimdall**: `InferencePool`, `EndpointPickerConfig`, plugin names/parameters, scheduling profiles, routing, load balancing
   - **LWS**: `LeaderWorkerSet`, worker configuration
   - **Presets**: model deployment guides, `website/docs/features/preset.mdx` (YAML examples of `InferenceService` / `InferenceServiceTemplate`), `mif.moreh.io/*` labels

## Pre-conditions Checklist

Before starting any version bump, confirm:

- [ ] The target version exists (chart repo, container registry, or release page).
- [ ] Any breaking changes or migration steps are understood.
- [ ] If the bump requires coordinated changes across multiple components, all components are being updated together.

## Verification Checklist

After completing the version bump:

- [ ] `make helm-dependency` succeeds without errors.
- [ ] `make helm-lint` passes for all charts.
- [ ] `make helm-docs` regenerates cleanly (no unexpected diff).
- [ ] Grep for the **old** version string confirms no stale references remain.
- [ ] Website documentation reflects the new version and any spec changes.

## Commit Convention

Follow the project's [Git Commit Guidelines](../../CLAUDE.md):

```
chore(deploy): bump <component> version(s)

- <component1>: <old-version> -> <new-version>
- <component2>: <old-version> -> <new-version>
```

The `<issue-id>:` prefix (e.g., `MAF-19235:` or `NO-ISSUE:`) is automatically added by the pre-commit hook based on the branch name. Do not include it manually.

If website documentation is also updated in the same commit, use a broader scope or split into separate commits:

```
chore(deploy): bump odin to v0.8.0
docs(website): update docs for odin v0.8.0 spec changes
```

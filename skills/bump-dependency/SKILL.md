---
name: bump-dependency
description: Guide for updating MIF dependency versions. Use this skill when asked to bump, update, or upgrade a dependency such as Odin, Odin-CRD, LWS, Heimdall, heimdall-proxy, moreh-vLLM presets, moai-inference-framework, or moai-inference-preset. Covers which files to modify, how to run updates, and how to verify the result.
---

# Dependency Version Update Guide

## Overview

MIF depends on several components whose versions are tracked across Helm charts, container image references, preset directories, and website documentation. This guide ensures all references are updated consistently when bumping a dependency.

## Components

| Component                | Type              | Version Locations                                                                                                                        |
| :----------------------- | :---------------- | :--------------------------------------------------------------------------------------------------------------------------------------- |
| Odin + Odin-CRD          | Helm sub-chart    | `deploy/helm/moai-inference-framework/Chart.yaml`                                                                                        |
| LWS                      | Helm sub-chart    | `deploy/helm/moai-inference-framework/Chart.yaml`                                                                                        |
| Heimdall                 | External Helm chart | Not a moai-inference-framework sub-chart by default. Ask user for chart repo and deployment details.                                    |
| heimdall-proxy           | Container image   | `deploy/helm/moai-inference-preset/templates/runtime-bases/*.helm.yaml`, `deploy/helm/moai-inference-preset/templates/utils/*.helm.yaml` |
| moreh-vLLM preset        | Preset directory  | `deploy/helm/moai-inference-preset/templates/presets/moreh-vllm/`                                                                        |
| moai-inference-framework | MIF chart release | `website/docs/getting-started/prerequisites.mdx`, `website/docs/getting-started/quickstart.mdx`                                          |
| moai-inference-preset    | MIF chart release | `website/docs/getting-started/prerequisites.mdx`, `website/docs/getting-started/quickstart.mdx`                                          |

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
5. Search `website/docs/` for references to the updated component and update if needed (see [Website Updates for Spec Changes](#6-website-updates-for-spec-changes)).

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

Heimdall is deployed as a separate Helm chart, not as a sub-chart of moai-inference-framework.

**Required information:** target version and chart repository URL (ask the user — the repo is private).

**Steps:**

1. Ask the user for the Heimdall chart repository URL and target version.
2. Identify where Heimdall version references exist (ask user if unsure — may include website docs or deployment scripts).
3. Update all identified references.
4. Search `website/docs/` for Heimdall-related documentation and update if needed (see [Website Updates for Spec Changes](#6-website-updates-for-spec-changes)).

### 3. heimdall-proxy Image Bump

**Required information:** target image tag (e.g., `v0.7.0-rc.5`).

**Steps:**

1. Grep for the current image tag across the preset chart templates:
   ```
   grep -r "heimdall-proxy:" deploy/helm/moai-inference-preset/templates/
   ```
2. Replace the old tag with the new tag in **every** matching file:
   - `deploy/helm/moai-inference-preset/templates/runtime-bases/*.helm.yaml`
   - `deploy/helm/moai-inference-preset/templates/utils/*.helm.yaml`
3. Verify no stale references remain by grepping for the old tag.

### 4. moreh-vLLM Preset Update

**Required information:** new version string, which models/configs are affected, any changes to vLLM arguments or resource requirements.

**Steps:**

1. Ask the user for the scope of changes: new models, updated configs, or full version bump.
2. Update or create preset directories under:
   ```
   deploy/helm/moai-inference-preset/templates/presets/moreh-vllm/<version>/
   ```
3. Follow the [preset naming convention](../../deploy/helm/AGENTS.md) defined in `deploy/helm/AGENTS.md`.
4. Update runtime-base templates if the new version changes launch logic, proxy configuration, or disaggregation behavior.

### 5. MIF Chart Release Update (moai-inference-framework, moai-inference-preset)

**Required information:** new chart release version (e.g., `v0.4.0`).

**Steps:**

1. Update version references in `website/docs/getting-started/prerequisites.mdx`:
   - `helm upgrade` commands that specify `--version`.
2. Update version references in `website/docs/getting-started/quickstart.mdx`:
   - Version badges / prerequisite version list.
3. Search `website/docs/` for any other references to the old version string.
4. Do **NOT** modify `website/versioned_docs/` — these are frozen snapshots of past versions.
5. Do **NOT** modify `Chart.yaml` `version` or `appVersion` fields — these are set by CI/CD during release.

### 6. Website Updates for Spec Changes

When a dependency introduces API, CRD, or configuration changes (not just a version number bump):

1. Ask the user what changed: new fields, removed fields, renamed APIs, behavior changes.
2. Search `website/docs/` for references to affected CRD kinds, field names, or config options:
   ```
   grep -r "<old-field-or-kind>" website/docs/
   ```
3. Update affected documentation: YAML examples, API reference pages, feature descriptions.
4. Common search targets by component:
   - **Odin CRD**: `InferenceService`, `InferenceServiceTemplate`, `InferencePool`, `templateRefs`
   - **Heimdall**: scheduling, routing, load balancing, metric collection
   - **LWS**: `LeaderWorkerSet`, worker configuration
   - **Presets**: model deployment guides, preset feature docs, `mif.moreh.io/*` labels

## Pre-conditions Checklist

Before starting any version bump, confirm:

- [ ] The target version exists (chart repo, container registry, or release page).
- [ ] Any breaking changes or migration steps are understood.
- [ ] If the bump requires coordinated changes across multiple components (e.g., Odin CRD change that affects presets), all components are being updated together.

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

---
name: release
description: >
  Cut a MIF release — tag a release candidate or a stable version, and for stable
  versions write the GitHub release notes. Use this skill when the user asks to
  "release", "cut a release", "create a release", "tag <version>", "write release
  notes", or says "/release". Also trigger when the user mentions "release notes",
  "changelog", or an rc version in the context of shipping MIF.
---

# MIF Release

Tag MIF releases and, for stable versions, publish GitHub release notes that highlight
dependency version changes and key improvements.

Publishing is driven entirely by **pushing a tag** — the workflows in `.github/workflows/`
trigger on the tag pattern. Nothing publishes until a tag reaches the remote.

## Version Rules

MIF follows [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) and semver.
Since we're pre-1.0 (`v0.x.y`):

| Commit type | Bump |
|---|---|
| `fix`, `refactor`, `style`, `chore`, `docs`, `test` | patch |
| `feat` | minor |
| `!` (breaking change) | minor (not major, because v0.x) |

Release candidates are numbered `vX.Y.Z-rc.N` starting at `rc.1`. Multiple rcs per minor
version are normal — v0.4.0 went through `rc.1` to `rc.5` before the stable tag.

## Release types

The tag pattern selects the workflow, and the workflows differ in where they publish:

| Tag | Workflow | Publishes to | GitHub release | Versioned docs |
|---|---|---|---|---|
| `vX.Y.Z-rc.N` | `cd-mif-stage.yaml` | artifact-keeper ChartMuseum **only** | no | no |
| `vX.Y.Z` | `cd-mif-prod.yaml` | ChartMuseum **and** the public `moreh-dev/helm-charts` | yes | yes |

**An rc never reaches the public Helm repository.** `release-chart/action.yaml` defaults
`publish_helm_charts` to `"false"`, and only the prod workflow sets it to `"true"`.
Do not expect an rc to fix version pins in `website/docs/` — those resolve at the stable
release. (`v0.5.0-rc.1` appears in the public index only because it was tagged before
`ad11141` split publishing per environment.)

### Release candidate flow

Only the version decision and the tag apply. Skip versioned docs, release notes, and the
GitHub release — no rc has ever had any of them.

```bash
# 1. Refuse to tag a dirty tree, or a commit that is not the tip of origin/main.
#    Each line exits non-zero rather than just printing state.
git fetch origin main
test -z "$(git status --porcelain)" || { echo "working tree is dirty"; exit 1; }
test "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)" || { echo "HEAD is not origin/main"; exit 1; }

# 2. Annotated tag, message summarizing the delta since the previous rc
git tag -a <version> -m "<summary>"

# 3. Push the tag — this is what triggers cd-mif-stage.yaml
git push origin <version>

# 4. Verify the workflow fired and succeeded
gh run list --limit 5
```

Include any upgrade hazard in the tag message; it is the only release artifact an rc
produces. Then jump to [Verify the publish](#7-verify-the-publish).

## Stable release flow

### 1. Determine version

```bash
# Find the latest stable tag
git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -1

# Find the previous stable tag for comparison
git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -2

# List commits since last stable tag
git log <latest-stable-tag>..HEAD --oneline --no-merges
```

If the tag already exists, use it. Otherwise, analyze commit types to compute the recommended
version bump. Present and wait for user confirmation.

### 2. Ensure versioned docs exist

The version comparison tables in the release notes rely on `website/versioned_docs/version-v<X.Y.Z>/operations/latest-release.mdx`.
Before investigating changes, verify that versioned docs exist for **both** the release version and the previous version.

```bash
# Check if versioned docs exist for the release version
ls website/versioned_docs/version-<releaseVersion>/operations/latest-release.mdx

# Check if versioned docs exist for the previous version
ls website/versioned_docs/version-<prevVersion>/operations/latest-release.mdx
```

If the release version's versioned docs do not exist, create them:

```bash
cd website
npm run docs:version <releaseVersion>
```

This snapshots the current `website/docs/` into `website/versioned_docs/version-<releaseVersion>/`.
Commit the generated `versioned_docs`, `versioned_sidebars`, and `versions.json` before proceeding.

**Important:** `website/docs/operations/latest-release.mdx` must already reflect the release version's
dependency versions **before** running `docs:version`. If it doesn't, update it first, then create
the versioned docs.

### 3. Investigate changes

Thoroughly research all changes between the previous stable tag and the release tag.

#### 3a. Commit and PR analysis

```bash
# Commits between releases
git log <prevVersion>..<releaseVersion> --oneline --no-merges

# Overall change stats
git diff <prevVersion>..<releaseVersion> --stat | tail -5
```

Group commits by scope to understand the breadth of changes:
- `deploy` — Helm chart, infrastructure, dependency bumps
- `website` — documentation
- `skills` — agent skills

#### 3b. Dependency version changes

Read the version tables from the versioned docs created in step 2:

```bash
# Release versions
cat website/versioned_docs/version-<releaseVersion>/operations/latest-release.mdx

# Previous release versions (for comparison)
cat website/versioned_docs/version-<prevVersion>/operations/latest-release.mdx
```

If versioned docs don't exist for the previous release, extract versions from the Helm chart at that tag:

```bash
git show <prevVersion>:deploy/helm/moai-inference-framework/Chart.yaml
```

#### 3c. Area-specific diffs

Check each major area for changes:

```bash
# Helm chart changes
git diff <prevVersion>..<releaseVersion> --stat -- deploy/helm/moai-inference-framework/

# Website changes
git diff <prevVersion>..<releaseVersion> --stat -- website/

# Skills changes
git diff <prevVersion>..<releaseVersion> --stat -- skills/ .agents/skills/
```

### 4. Write release notes

Structure the release notes as follows:

```markdown
## Dependency Version Changes

### MIF Helm Charts

| Component | <prevVersion> | <releaseVersion> |
|-----------|--------|--------|
| moai-inference-framework | vX.Y.Z | **vA.B.C** |

### Core Components

Installed as separate charts, each pinned independently in
`website/docs/getting-started/prerequisites.mdx`. They drift apart easily — check every row
against `operations/latest-release.mdx` rather than assuming the CRD charts track their
operator:

| Component | <prevVersion> | <releaseVersion> |
|-----------|--------|--------|
| Odin | vX.Y.Z | **vA.B.C** |
| Odin CRD | vX.Y.Z | **vA.B.C** |
| Heimdall | vX.Y.Z | **vA.B.C** |
| Heimdall CRD | vX.Y.Z | **vA.B.C** |
| Heimdall AIGateway CRD | vX.Y.Z | **vA.B.C** |
| LWS | X.Y.Z | **A.B.C** |

### Infrastructure Dependencies

Bundled as sub-charts in `moai-inference-framework`. Take the list from that chart's
`Chart.yaml` at the release tag rather than from this template — sub-charts are added and
removed between releases (`odin`, `odin-crd`, and `keda` were unbundled in `c58d8bb`):

| Component | <prevVersion> | <releaseVersion> |
|-----------|--------|--------|
| kube-prometheus-stack | X.Y.Z | X.Y.Z |
| ... | ... | ... |

> Use `—` for components that didn't exist in the previous release.
> **Bold** changed versions. Leave unchanged versions unbolded.

## Highlights

### <Area Name>
- Description of change (#PR)

### <Area Name>
- Description of change (#PR)

## What's Changed

**Full Changelog**: https://github.com/moreh-dev/mif/compare/<prevVersion>...<releaseVersion>
```

Guidelines for the **Highlights** section:
- Group by functional area, not by commit type. Common areas:
  - Observability Stack
  - Hardware Support
  - Documentation (Website)
  - Agent Skills
- Write from the **user's perspective** — focus on what they gain, not internal implementation.
- Reference PR numbers with `#N` format (auto-linked by GitHub).
- Group related PRs into a single entry when they form one logical change.
- Omit purely internal changes (CI tweaks, minor doc typos) unless they affect users.

### 5. Present for review

Show the full release note to the user and ask for approval or edits. Do not proceed until
the user explicitly approves.

Pay special attention to dependency versions — the agent may not have full visibility into
versions managed outside this repo (e.g., Heimdall and its CRD charts). Cross-check them
against the published index, and explicitly ask the user to verify any you are uncertain
about:

```bash
curl -s https://moreh-dev.github.io/helm-charts/index.yaml \
  | grep -E '^[[:space:]]+(name|version):'
```

### 6. Create GitHub Release

`gh release create` creates the tag at the target commit if it does not already exist, and
that tag push is what triggers `cd-mif-prod.yaml`. Confirm the working tree is clean and
`HEAD` matches `origin/main` first.

```bash
gh release create <version> --title "<version>" --notes "$(cat <<'EOF'
<release-note-content>
EOF
)"
```

After creation, print the release URL.

If edits are needed after creation, update with:

```bash
gh release edit <version> --notes "$(cat <<'EOF'
<updated-release-note-content>
EOF
)"
```

### 7. Verify the publish

The tag only starts the job — confirm it finished and landed where expected.

```bash
# The deploy workflow should appear for this tag and succeed
gh run list --limit 5

# Stable only: the chart should appear in the public index
curl -s https://moreh-dev.github.io/helm-charts/index.yaml \
  | grep -A2 'moai-inference-framework'
```

A successful stage run publishes to the ChartMuseum only, so the public index will **not**
change for an rc. Report which registry actually received the chart rather than assuming
from the workflow name.

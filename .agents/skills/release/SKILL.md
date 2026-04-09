---
name: release
description: >
  Create a new MIF GitHub release. Use this skill when the user asks to "release",
  "cut a release", "create a release", "write release notes", or says "/release".
  Also trigger when the user mentions "release notes" or "changelog" in the context
  of shipping a new MIF version.
---

# MIF Release

Create GitHub releases for the MIF project with curated release notes that highlight
dependency version changes and key improvements.

## Version Rules

MIF follows [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) and semver.
Since we're pre-1.0 (`v0.x.y`):

| Commit type | Bump |
|---|---|
| `fix`, `refactor`, `style`, `chore`, `docs`, `test` | patch |
| `feat` | minor |
| `!` (breaking change) | minor (not major, because v0.x) |

## Release Flow

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

### 2. Investigate changes

Thoroughly research all changes between the previous stable tag and the release tag.

#### 2a. Commit and PR analysis

```bash
# Commits between releases
git log <prev-tag>..<release-tag> --oneline --no-merges

# Overall change stats
git diff <prev-tag>..<release-tag> --stat | tail -5
```

Group commits by scope to understand the breadth of changes:
- `deploy` — Helm chart, infrastructure, dependency bumps
- `preset` — inference service templates, quickstart presets
- `website` — documentation
- `e2e` / `test` — testing framework
- `skills` — agent skills

#### 2b. Dependency version changes

Read the version table from the **current** `website/docs/operations/latest-release.mdx`.
This is the source of truth for v0.4.0+ component versions.

Also read the **previous release's** version of the same file to build the comparison:

```bash
# Current versions
cat website/versioned_docs/version-<release-version>/operations/latest-release.mdx

# Previous versions (for comparison)
cat website/versioned_docs/version-<prev-version>/operations/latest-release.mdx
```

If versioned docs don't exist for the previous release, extract versions from the Helm chart at that tag:

```bash
git show <prev-tag>:deploy/helm/moai-inference-framework/Chart.yaml
```

#### 2c. Area-specific diffs

Check each major area for changes:

```bash
# Helm chart changes
git diff <prev-tag>..<release-tag> --stat -- deploy/helm/moai-inference-framework/

# Preset changes
git diff <prev-tag>..<release-tag> --stat -- deploy/helm/moai-inference-preset/

# Website changes
git diff <prev-tag>..<release-tag> --stat -- website/

# Test changes
git diff <prev-tag>..<release-tag> --stat -- test/

# Skills changes
git diff <prev-tag>..<release-tag> --stat -- skills/ .agents/skills/
```

### 3. Write release notes

Structure the release notes as follows:

```markdown
## Dependency Version Changes

### MIF Helm Charts

| Component | <prev-version> | <release-version> |
|-----------|--------|--------|
| moai-inference-framework | vX.Y.Z | **vA.B.C** |
| moai-inference-preset | vX.Y.Z | **vA.B.C** |

### Core Components

| Component | <prev-version> | <release-version> |
|-----------|--------|--------|
| Odin | vX.Y.Z | **vA.B.C** |
| Odin CRD | vX.Y.Z | **vA.B.C** |
| Heimdall | vX.Y.Z | **vA.B.C** |
| heimdall-proxy | vX.Y.Z | **vA.B.C** |
| LWS | X.Y.Z | **A.B.C** |
| moreh-vLLM preset | X.Y.Z | **A.B.C** |
| Istio | X.Y.Z | **A.B.C** |

### Infrastructure Dependencies

Bundled as sub-charts in `moai-inference-framework`:

| Component | <prev-version> | <release-version> |
|-----------|--------|--------|
| kube-prometheus-stack | X.Y.Z | X.Y.Z |
| KEDA | X.Y.Z | X.Y.Z |
| ... | ... | ... |

> Use `—` for components that didn't exist in the previous release.
> **Bold** changed versions. Leave unchanged versions unbolded.

## Highlights

### <Area Name>
- Description of change (#PR)

### <Area Name>
- Description of change (#PR)

## What's Changed

**Full Changelog**: https://github.com/moreh-dev/mif/compare/<prev-tag>...<release-tag>
```

Guidelines for the **Highlights** section:
- Group by functional area, not by commit type. Common areas:
  - Observability Stack
  - Hardware Support
  - Preset Expansion
  - Documentation (Website)
  - E2E Testing
  - Agent Skills
- Write from the **user's perspective** — focus on what they gain, not internal implementation.
- Reference PR numbers with `#N` format (auto-linked by GitHub).
- Group related PRs into a single entry when they form one logical change.
- Omit purely internal changes (CI tweaks, minor doc typos) unless they affect users.

### 4. Present for review

Show the full release note to the user and ask for approval or edits. Do not proceed until
the user explicitly approves.

Pay special attention to dependency versions — the agent may not have full visibility into
versions managed outside this repo (e.g., Heimdall, Istio). Explicitly ask the user to
verify any versions you are uncertain about.

### 5. Create GitHub Release

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

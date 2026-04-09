# Contribution Guide

This guide serves as the unified source of truth for all contributors, both human engineers and automation agents.

## How to navigate this repo

- For E2E test rules, see [`test/AGENTS.md`](test/AGENTS.md).
- For Helm chart rules, see [`deploy/helm/AGENTS.md`](deploy/helm/AGENTS.md).
- For website and documentation rules, see [`website/AGENTS.md`](website/AGENTS.md).
- For agent workflow guides tied to specific domains, see [`skills/README.md`](skills/README.md) and the relevant `SKILL.md`.

## General Rules

### Git Commit Guidelines

> [!NOTE]
> References:
>
> - [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/)

The commit message should be structured as follows:

```shell
<type>[(<scope>)][!]: <description>

[<body>]

[<footer>]
```

#### Type

- `<type>(<scope>)!`: Breaking change. (major version bump except for `v0.x.x` where it bumps minor version)
- `feat`: New feature or enhancement. (minor version bump)
- `fix`: Bug fix. (patch version bump)
- `refactor`: Code change that neither fixes a bug nor adds a feature.
- `style`: Code style changes (whitespace, formatting, etc.) that do not affect functionality.
- `test`: Only test-related changes.
- `docs`: Only documentation changes.
- `chore`: Changes without a direct impact on the codebase (build process, dependencies, etc.).

#### Scope

- `workflow`: Changes related to CI/CD workflows.
- `deploy`: Changes related to deployment (Helm charts, container files, etc.)
- `config`: Changes related to files hard to manage within helm charts.
- `preset`: Changes related to preset files.
- `website`: Changes related to website.
- `e2e`: Changes related to end-to-end tests.
- `skills`: Changes related to agent skills.

### Code Style Guidelines

#### Comments

- **Language**: All comments must be in English.

#### Diagrams

- **Format**: Use [Mermaid](https://mermaid.js.org/) fenced code blocks (` ```mermaid `) for diagrams. Do not use plain-text arrow diagrams in fenced code blocks; directory trees and inline prose are exempt.

#### Go Templates

- **Indentation**: Go template syntax (e.g., `{{- if ... }}`) must be indented to match the surrounding code context for better readability.

## Maintenance

### Agent Self-Improvement

After completing any non-trivial task, evaluate whether the work involved:
- A recurring pattern that will likely appear again in future tasks, or
- A mistake that was corrected through user feedback, or
- A design decision that required deliberate reasoning to reach the right answer.

If any of the above applies, **record it in the most relevant maintenance document before closing the task**:
- Prefer the nearest domain-specific `AGENTS.md`.
- Use this file only for repo-wide patterns.
- Update a `SKILL.md` only when the knowledge belongs in an agent skill rather than an `AGENTS.md` rule.

Entries should be concise, actionable, and placed under the most relevant existing section. If no section fits, create one.

The goal is to make every repeated task faster and every repeated mistake impossible.

### Creating Sub-directory AGENTS.md Files

When a directory accumulates enough domain-specific rules to warrant separation, create a dedicated `AGENTS.md` in that directory. Follow this checklist:

1. **Create `AGENTS.md`** in the target directory with a header that links back to this root file:
   ```markdown
   # <Domain> — Agent Rules

   Rules specific to the `<dir>/` directory. General contribution guidelines are in the root [`AGENTS.md`](/AGENTS.md).
   ```

2. **Create a `CLAUDE.md` symlink** pointing to `AGENTS.md` in the same directory. Cursor reads `CLAUDE.md` as context; the symlink ensures both tools see the same content:
   ```shell
   cd <dir> && ln -s AGENTS.md CLAUDE.md
   ```

3. **Move the relevant sections** from the root `AGENTS.md` (or parent `AGENTS.md`) into the new file. Replace the moved content in the parent with a one-line reference:
   ```markdown
   ### E2E Test

   See [`test/AGENTS.md`](test/AGENTS.md).
   ```

4. **Update the Agent Self-Improvement section** in the parent to mention the new file as a recording target.

## Domain References

### E2E Test

See [`test/AGENTS.md`](test/AGENTS.md).

### Helm Charts

See [`deploy/helm/AGENTS.md`](deploy/helm/AGENTS.md) for design principles and chart development rules.

### Website

See [`website/AGENTS.md`](website/AGENTS.md) for Docusaurus site and documentation rules.

### Agent Skills

Domain-specific expert guides for AI agents are in [`skills/`](skills/). See [`skills/README.md`](skills/README.md) for installation and available skills.

The `skills/` directory intentionally does not have its own `AGENTS.md`. Skills are distributed as a Claude Code plugin, and the directory structure follows the plugin specification rather than the sub-directory `AGENTS.md` convention.

When updating knowledge in `skills/`, modify the relevant `SKILL.md` or its supporting files directly. Do not create `AGENTS.md` inside `skills/`.

When working on a specific MIF component, consult the relevant skill:

- **Dependency version updates**: [`.agents/skills/bump-dependency/SKILL.md`](.agents/skills/bump-dependency/SKILL.md)
- **GitHub release**: [`.agents/skills/release/SKILL.md`](.agents/skills/release/SKILL.md)
- **Heimdall scheduler**: [`skills/guide-heimdall/SKILL.md`](skills/guide-heimdall/SKILL.md)
- **Odin inference operator**: [`skills/guide-odin/SKILL.md`](skills/guide-odin/SKILL.md)

### Offline Hugging Face cache

- For air-gapped `trust_remote_code` deployments, pre-download both the model snapshot and the dynamic module sources. `hf download` alone may leave `HF_MODULES_CACHE` incomplete; if the warm-up pod lacks `torch` or other model-side dependencies, populate `HF_MODULES_CACHE` from the local HF snapshot cache rather than relying on `transformers` to import the remote module.

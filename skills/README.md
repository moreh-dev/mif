# MIF Agent Skills

Agent Skills package domain-specific expert knowledge for AI coding assistants. Each skill is a self-contained directory with a `SKILL.md` file containing actionable instructions for configuring, deploying, and troubleshooting specific MIF components.

Skills follow the [Agent Skills open standard](https://agentskills.io/specification) and are compatible with Gemini CLI, Claude Code, and Cursor.

## Available Skills

| Skill | Description |
| ----- | ----------- |
| [bump-dependency](./bump-dependency/) | Dependency version update procedures for Helm charts, images, presets, and docs |
| [guide-heimdall](./guide-heimdall/) | Heimdall scheduler configuration, plugin selection, and deployment |
| [guide-odin](./guide-odin/) | Odin inference operator, InferenceService, templates, and parallelism |

## Installation

### Gemini CLI

Install skills directly from the remote repository:

```shell
gemini skills install git@github.com:moreh-dev/mif.git --path skills
```

Or, if you already have the MIF repository cloned, link the local skills directory:

```shell
gemini skills link ./skills --scope workspace
```

Verify installed skills:

```shell
gemini skills list
```

**Compatibility notes:**

- Both skills use the required `name` and `description` YAML frontmatter fields.
- Directory names match the `name` field in each `SKILL.md` (e.g., `guide-heimdall/SKILL.md` has `name: guide-heimdall`).
- The `references/` subdirectory in each skill contains supplementary material (config recipes). Gemini CLI loads all resources at skill activation time rather than on-demand.

### Claude Code

There are two ways to use MIF skills in Claude Code:

#### Option A: Automatic discovery (in-repo)

When working inside the MIF repository, skills are automatically discoverable via the root `AGENTS.md` / `CLAUDE.md` reference. No additional installation is needed. When working on a specific component, Claude will reference the relevant skill file.

#### Option B: Plugin installation (from any project)

For using MIF skills from **outside** the MIF repository, install as a Claude Code plugin:

```shell
# Step 1: Add the MIF marketplace
claude plugin marketplace add moreh-dev/mif

# Step 2: Install the skills plugin
claude plugin install mif-skills@mif
```

The same commands are available interactively inside Claude Code:

```
/plugin marketplace add moreh-dev/mif
/plugin install mif-skills@mif
```

**Scope options:**

```shell
# Install for yourself across all projects (default)
claude plugin install mif-skills@mif --scope user

# Install for your team via .claude/settings.json
claude plugin install mif-skills@mif --scope project
```

**Management commands (CLI):**

```shell
claude plugin list                         # List installed plugins
claude plugin disable mif-skills@mif       # Disable without uninstalling
claude plugin uninstall mif-skills@mif     # Remove completely
claude plugin marketplace update mif       # Update marketplace to pick up new skills
```

**Management commands (interactive):**

```
/plugin                           # List installed plugins
/plugin disable mif-skills@mif    # Disable without uninstalling
/plugin uninstall mif-skills@mif  # Remove completely
/plugin marketplace update mif    # Update marketplace to pick up new skills
```

**Local testing (development):**

```shell
# Test the plugin directly from a local checkout
claude --plugin-dir ./skills
```

#### How the plugin works

The plugin is defined by two files:

| File | Purpose |
| ---- | ------- |
| `.claude-plugin/marketplace.json` (repo root) | Registers the MIF repository as a Claude Code marketplace |
| `skills/.claude-plugin/plugin.json` | Plugin manifest that exposes the skills directory |

The `plugin.json` uses `"skills": "./"` to tell Claude Code that skill directories (`guide-heimdall/`, `guide-odin/`) are located in the plugin root itself, rather than in a nested `skills/` subdirectory.

### Cursor

Skills are automatically discoverable via the `CLAUDE.md` symlink, which references the skills directory. No additional installation is needed.

## Skill Format

Each skill follows the [Agent Skills specification](https://agentskills.io/specification):

```
guide-<component>/
├── SKILL.md                # Required: YAML frontmatter + markdown instructions
└── references/             # Optional: supplementary material
    └── config-recipes.md   # Ready-to-use configuration examples
```

### Required YAML frontmatter

```yaml
---
name: guide-<component> # Must match the directory name
description: >- # Describes what the skill does and when to use it
  Expert guide for ...
---
```

### Optional frontmatter fields

| Field | Description |
| ----- | ----------- |
| `license` | License name or reference to bundled file |
| `compatibility` | Environment requirements (e.g., `Requires kubectl and Helm`) |
| `metadata` | Arbitrary key-value map (e.g., `author`, `version`) |

## Creating a New Skill

1. Create a directory under `skills/` matching the skill name (e.g., `skills/guide-<component>/`).
2. Add a `SKILL.md` with YAML frontmatter (`name`, `description`) and markdown instructions.
3. Optionally add `references/`, `scripts/`, or `assets/` subdirectories for supplementary material.
4. Update the **Available Skills** table above.
5. Add a reference in the root `AGENTS.md` under the **Agent Skills** section.
6. Commit with scope `skills`: `feat(skills): add guide-<name> skill`.

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


## Creating a New Skill

Use the `skill-creator` Claude Code plugin to create new skills. It handles the skill format, frontmatter, and directory structure automatically.

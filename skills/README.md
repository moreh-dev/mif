# MIF Agent Skills

Agent Skills package domain-specific expert knowledge for AI coding assistants. Each skill is a self-contained directory with a `SKILL.md` file containing actionable instructions for configuring, deploying, and troubleshooting specific MIF components.

## Available Skills

| Skill | Description |
| ----- | ----------- |
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

### Claude Code

Skills are automatically discoverable via the root `AGENTS.md` / `CLAUDE.md` reference. No additional installation is needed. When working on a specific component, Claude will reference the relevant skill file.

### Cursor

Skills are automatically discoverable via the `CLAUDE.md` symlink, which references the skills directory. No additional installation is needed.

## Creating a New Skill

1. Create a directory under `skills/` matching the skill name (e.g., `skills/guide-<component>/`).
2. Add a `SKILL.md` with YAML frontmatter (`name`, `description`) and markdown instructions.
3. Optionally add `references/`, `scripts/`, or `assets/` subdirectories for supplementary material.
4. Update the **Available Skills** table above.
5. Add a reference in the root `AGENTS.md` under the **Agent Skills** section.
6. Commit with scope `skills`: `feat(skills): add guide-<name> skill`.

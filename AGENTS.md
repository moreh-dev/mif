# Contribution Guide

This guide serves as the unified source of truth for all contributors, both human engineers and automation agents.

## Git Commit Guidelines

> [!NOTE]
> References:
>
> - [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/)

The Conventional Commits specification is a lightweight convention on top of commit messages. It provides an easy set of rules for creating an explicit commit history; which makes it easier to write automated tools on top of. This convention dovetails with SemVer, by describing the features, fixes, and breaking changes made in commit messages.

The commit message should be structured as follows:

```shell
<type>[(<scope>)][!]: <description>

[<body>]

[<footer>]
```

### Type

- `<type>(<scope>)!`: Breaking change. (major version bump except for `v0.x.x` where it bumps minor version)
- `feat`: New feature or enhancement. (minor version bump)
- `fix`: Bug fix. (patch version bump)
- `refactor`: Code change that neither fixes a bug nor adds a feature.
- `style`: Code style changes (whitespace, formatting, etc.) that do not affect functionality.
- `test`: Only test-related changes.
- `docs`: Only documentation changes.
- `chore`: Changes without a direct impact on the codebase (build process, dependencies, etc.).

### Scope

- `workflow`: Changes related to CI/CD workflows.
- `deploy`: Changes related to deployment (Helm charts, container files, etc.)
- `config`: Changes related to files hard to manage within helm charts.
- `preset`: Changes related to preset files.

## Code Style Guidelines

### Comments

- **All comments in code must be written in English.**
- This applies to:
  - Shell scripts (`.sh` files)
  - Makefiles
  - Configuration files (YAML, JSON, TOML, INI, etc.)
  - Source code comments

#### Comment Guidelines

- **Remove verbose or redundant comments**: If the code is self-explanatory, remove the comment.
- **Remove obvious comments**: Comments that simply restate what the code does should be removed.
  - ❌ Bad: `// Check if secret already exists`
  - ✅ Good: Code is clear without comment
- **Remove implementation details**: Comments explaining step-by-step implementation should be removed if the code flow is clear.
  - ❌ Bad: `// 1. Create hosts.toml file for the registry (containerd v1.5+ standard)`
  - ✅ Good: Code structure makes the steps clear
- **Keep essential comments**: Only keep comments that explain:
  - **Why** something is done (not what or how)
  - Non-obvious business logic or edge cases
  - Complex algorithms or workarounds
  - Public API documentation (for exported functions)
- **Function documentation**: For exported functions, use concise doc comments that explain purpose, not implementation details.
  - ❌ Bad: `// createMIFValuesFile creates a temporary values file for moai-inference-framework that configures ecrTokenRefresher with the given AWS credentials. These credentials are sourced from environment variables...`
  - ✅ Good: `// createMIFValuesFile creates a values file for moai-inference-framework with ECR token refresher configuration.`
- **Inline comments**: Use sparingly and only for non-obvious logic.
  - ❌ Bad: `// Update namespace in metadata`
  - ✅ Good: Code is self-explanatory

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
- `e2e`: Changes related to end-to-end tests.

## Code Style Guidelines

### Comments

- **Language**: All comments must be in English.
- **Goals**: Focus on **why** something is done, not what or how.
  - Explain non-obvious business logic, edge cases, or complex algorithms.
  - Avoid restating the obvious or detailing implementation steps if the code is clear.
- **API Documentation**: Exported functions should have concise doc comments explaining their purpose.

### Go Templates

- **Indentation**: Go template syntax (e.g., `{{- if ... }}`) must be indented to match the surrounding code context for better readability.

### E2E Test

- **Do not test resource specifications**:
  - Do not validate individual fields of the YAML file declaring the resource (resource spec).
  - Instead, create the resource and verify that its status reaches the expected state.
- **Assume fully controlled cluster**:
  - Do not check if components are already installed.
  - Assume the cluster is fully controlled by the test and installed components are safe to overwrite or delete.

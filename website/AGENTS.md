# Website (Docusaurus) — Agent Rules

This file defines rules for contributors and automation agents working in `website/`.

## 1. Structure & Metadata

- **Frontmatter**: All `.mdx` files must start with the following (wrap titles in single quotes):
  ```mdx
  ---
  title: <title>
  sidebar_label: <label>
  sidebar_position: <position>
  ---
  ```
- **Categories**: Every subfolder under `docs/` containing docs must have a `_category_.yaml` with `label` and `position`.
- **Placement**: Place new docs in appropriate subfolders (e.g., `getting-started`, `reference`). Create a new folder with `_category_.yaml` only if necessary.
- **Artifacts**: Never edit `versioned_docs/` or `versioned_sidebars/`. Only edit source files in `docs/`.

## 2. Formatting & Syntax

- **Standard Markdown**:
  - Use `-` for lists.
  - Use Sentence case for headings.
  - Add a horizontal rule (`---`) above H2 headings (except the top one).

### Tabs

https://docusaurus.io/docs/markdown-features/tabs

```mdx
import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs groupId="fruits" queryString>
  <TabItem value="apple" label="Apple" default>
    This is Apple
  </TabItem>
  <TabItem value="orange" label="Orange">
    This is Orange
  </TabItem>
</Tabs>
```

### Code blocks

https://docusaurus.io/docs/markdown-features/code-blocks

````mdx
```yaml
image:
  repository: ubuntu
  tag: "22.04"
```
````

- title: ` ```<language> title="<title>"`
- highlight: ` ```<language> {1,4-6}`
- **Variables**:
  - Format as `<variableName>` (camelCase, no quotes).
  - Highlight lines containing variables in code blocks (e.g., ` ```yaml {2} `).

### Admonitions

https://docusaurus.io/docs/markdown-features/admonitions

```mdx
::::info

This is a tip

:::warning
This is a warning
:::

::::
```

- `:::info`, `:::warning`

## 3. Tone & Voice

- **Audience**: Technical engineers (familiar with K8s, Helm, LLMs).
- **Perspective**:
  - **Procedures**: Second person ("You"). Example: "Replace `<hash>` with..."
  - **Concepts**: Third person. Example: "The controller manages..."
  - **Avoid**: First person ("We", "I").
- **Style**:
  - **Imperative**: Use command style for steps (e.g., "Create a file...").
  - **Concise**: Avoid marketing fluff. Link to prerequisites instead of repeating them.

# Website (Docusaurus) — Agent Rules

Rules specific to the `website/` directory. General contribution guidelines are in the root [`AGENTS.md`](/AGENTS.md).

## 1. Structure & Metadata

- **File Extension**: All documentation files must use the `.mdx` extension.
- **Frontmatter**: All documentation files must start with the following:
  ```mdx
  ---
  title: <title>
  sidebar_label: <label>
  sidebar_position: <position>
  ---
  ```
- **Categories**: Every subfolder under `docs/` containing docs must have a `_category_.yaml` with `label`, `position`, and `collapsed: false`.
- **Placement**: Place new docs in appropriate subfolders (e.g., `getting-started`, `reference`). Create a new folder with `_category_.yaml` only if necessary.
- **Artifacts**: Never edit `versioned_docs/` or `versioned_sidebars/`. Only edit source files in `docs/`.
- **Images**: If a documentation file contains images, convert the file to a directory of the same name containing an `index.mdx` file. Place images directly within that directory (not in a subdirectory) and use simple filenames (e.g., `my-doc/image.png`).

## 2. Formatting & Syntax

- **Standard Markdown**:
  - Use `-` for lists.
  - Use Sentence case for headings.
  - Add a horizontal rule (`---`) above H2 headings (except the top one).

### Tabs

https://docusaurus.io/docs/markdown-features/tabs

```mdx
import Tabs from "@theme/Tabs";
import TabItem from "@theme/TabItem";

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
- **Expected output**: Blocks showing command output must always specify both a language type and a title on the same opening fence. Use `shell` for terminal output:
  ````mdx
  ```shell title="Expected output (one pod per node, all `Running`)"
  NAME           READY   STATUS    RESTARTS   AGE
  vector-xxxxx   1/1     Running   0          2m
  ```
  ````
- **Variables**:
  - Format as `<variableName>` (camelCase, no quotes).
  - Highlight lines containing variables in code blocks (e.g., ` ```yaml {2} `).
- **No line number references in text**:
  - Do not refer to specific line numbers in the descriptive text (e.g., avoid "Replace the value on line 4").
  - Instead, refer to the content or field names (e.g., "Replace the `tags` value").

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

## 4. Content Guidelines

- **Inference Deployment**: When documenting deployment of inference services (e.g., vLLM, SGLang), instructions MUST use the `InferenceService` resource with a preset.

- **No duplicate installation steps**: Operation or feature docs must not repeat the values file example or `helm upgrade` command that already appears in `getting-started/prerequisites.mdx`. Instead, link directly to the relevant section:
  ```mdx
  See [Prerequisites](../getting-started/prerequisites.mdx#moai-inference-framework) for the required values and install command.
  ```
  Duplication causes the two pages to diverge whenever the chart version or values change.

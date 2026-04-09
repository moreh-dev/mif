# Website (Docusaurus) — Agent Rules

Rules specific to the `website/` directory. General contribution guidelines are in the root [`AGENTS.md`](/AGENTS.md).

These rules cover both:
- documentation content under `docs/`
- Docusaurus site code and configuration in this directory (e.g., `src/`, `static/`, `blog/`, `docusaurus.config.ts`, `sidebars.ts`)

## Production Build

```shell
cd website
npm run build
```

## Versioned Docs

To create a new documentation version aligned with a MIF release tag:

```shell
cd website
npm run docs:version X.Y.Z
```

Commit the generated `versioned_docs`, `versioned_sidebars`, and `versions.json` before creating the corresponding `vX.Y.Z` Git tag.

## Verification

- For documentation, config, or UI changes in `website/`, run:
  ```shell
  cd website
  npm run build
  ```
- Do not claim website work is complete if the production build fails.
- If the task changes only prose and the build cannot be run in the current environment, state that explicitly in the final report.

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
- **Artifacts**: Never edit `versioned_docs/` or `versioned_sidebars/` directly.
- **Documentation source**: For documentation content, edit source files under `docs/` only.
- **Website code/config**: When the task involves Docusaurus configuration, UI components, assets, or blog content, edit the relevant source files under `website/` (for example `src/`, `static/`, `blog/`, `docusaurus.config.ts`, `sidebars.ts`) instead of generated artifacts.
- **Images**: If a documentation file contains images, convert the file to a directory of the same name containing an `index.mdx` file. Place images directly within that directory (not in a subdirectory) and use simple filenames (e.g., `my-doc/image.png`).

## 2. Formatting & Syntax

- **Standard Markdown**:
  - Use `-` for lists.
  - Use Sentence case for headings.
  - Do not use H1 (`#`) in the body. The frontmatter `title` renders as H1 automatically.
  - Add a horizontal rule (`---`) above H2 headings (except the top one).

### Tabs

See [Docusaurus Tabs](https://docusaurus.io/docs/markdown-features/tabs). Always use `groupId` and `queryString` props.

### Code blocks

See [Docusaurus Code Blocks](https://docusaurus.io/docs/markdown-features/code-blocks).

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

See [Docusaurus Admonitions](https://docusaurus.io/docs/markdown-features/admonitions). Use `:::info` and `:::warning`.

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

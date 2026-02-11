# Website (Docusaurus) — Agent Rules

This file defines rules for contributors and automation agents working in the `website/` directory.

## Do not write code under `src/`

- **Do not add or modify any code under `website/src/`.** This project does not use custom theme code, swizzled components, or other files in `src/`.
- All behavior and UI must be achieved via:
  - **Configuration**: `docusaurus.config.ts`, `sidebars.ts`, plugin options.
  - **Content**: Markdown/MDX under `docs/`, front matter, and `_category_.json`.
  - **Styling**: `css/custom.css` and `static/` assets.
- If a feature would require adding or editing files under `src/`, do not implement it that way; suggest an alternative using config, docs, or CSS only, or state that the feature is out of scope for this setup.

This rule applies permanently so that the website remains maintainable without custom Docusaurus theme code.

## Documentation Standards

- Write all documentation in clear and concise English.
- Use Sentence case for all headings.
- Add a horizontal rule (`---`) and a new line above the second and subsequent H2 headings.
- Use `-` for lists.
- Use `<variable>` for variables in YAML to trigger an error if not replaced. Do not wrap it in double quotes ("").
- Use camelCase for variable names (e.g., `<huggingFaceToken>`).
- Highlight variables in code blocks using Docusaurus line highlight syntax with curly braces (for example, ` ```yaml {23,34} ` or ` ```yaml myfile.yaml {23,34} `) so that the lines containing `<variable>` placeholders are visually emphasized. Do **not** add highlights when they would cover almost the entire snippet; use them only when they help focus attention on a small subset of important lines (typically the lines that contain `<variable>` placeholders or other values the reader must change).
- **In front matter, wrap all `title` values in single quotes** (e.g. `title: 'My document title'`) to avoid YAML parsing issues (e.g. colons in the title).
- Use Docusaurus admonitions: `:::info`, `:::tip`, `:::warning`, `:::note`, `:::danger`. Optional title in brackets, e.g. `:::warning[Warning]`. Close with `:::`.
- For tabs, use Docusaurus `<Tabs>` and `<TabItem>` (import `@theme/Tabs` and `@theme/TabItem` at the top of the file when needed). Do not use MkDocs-style `==-` / `===`.
- Use standard GFM tables only; do not add MkDocs-style attributes such as `{.compact}`.
- **Every subfolder under `docs/` that contains docs (Markdown/MDX) must have a `_category_.json` file.** Use the same structure as existing categories (e.g. `docs/getting_started/_category_.json`): include `label`, `key` (kebab-case), `collapsible`, `collapsed`, and optionally `position` and `link` (e.g. generated-index). When adding a new docs folder, add its `_category_.json` at the same time. **When adding or updating documents within a category, check whether the category's description (e.g. `link.description` in `_category_.json`) still accurately reflects the contents; update the description if it no longer fits.**
- **Every new document must belong to a subcategory.** Place new docs in an existing subfolder under `docs/` that fits the topic (e.g. getting*started, features, reference). If no existing category fits the context, create a new subfolder, add the document(s) there, and add a `\_category*.json` for that folder so the new category appears in the sidebar.
- **Do not manually edit or validate generated documentation output.** The `versioned_docs/` and `versioned_sidebars/` directories are build artifacts produced by `docusaurus docs:version` / `docusaurus build`. Do not run manual style/content review, linters, or formatters on these folders, and do not hand-edit files under them; instead, edit the source files under `docs/` and regenerate.

## Tone and voice

The documentation is written for operators and engineers who deploy, configure, and run the MoAI Inference Framework (e.g. on Kubernetes). Keep the following in mind so new and updated content matches the existing tone.

- **Audience:** Technical readers with experience in Kubernetes, Helm, and (for some topics) GPU clusters or LLM inference. Assume familiarity with concepts such as namespaces, CRDs, and YAML; link to external references (e.g. Gateway API, cert-manager) where it helps, but avoid over-explaining basics.
- **Prerequisites as baseline:** All documents except the prerequisites document itself should be written **assuming that the prerequisites are already satisfied.** The reader is assumed to have the environment described in the prerequisites (e.g. Kubernetes, Helm, cert-manager, MoAI Inference Framework dependencies, GPU/network setup) in place. Do not re-explain or repeat prerequisite steps in other docs; link to the prerequisites document when readers must have it done first.
- **Voice:** Use **second person (“you”)** for procedures and requirements that address the reader directly (e.g. “You need to replace…”, “If you encounter difficulties…”). Use **neutral or third person** for conceptual or descriptive text (e.g. “The framework supports…”, “This component is responsible for…”). Do not use first person (“we”) in the body of the docs.
- **Style:** Be **clear, concise, and factual.** Prefer short sentences and concrete instructions. Avoid marketing language, filler phrases, or casual tone. Prefer active voice; use passive only when it improves clarity (e.g. when the actor is obvious or unimportant).
- **Procedures and steps:** Use **imperative** for commands and steps (e.g. “Deploy the chart using the following command.”, “Create a `values.yaml` file.”). Introduce code blocks with a single sentence that states what the reader is doing or what the snippet is for.
- **Placeholders and prerequisites:** State clearly when the reader must substitute values (e.g. “Replace `<huggingFaceToken>` with your token.”). Use admonitions (`:::info`, `:::warning`, `:::tip`) for prerequisites, caveats, and optional hints so they are visible without cluttering the main flow.
- **Consistency:** Match the tone of existing pages: direct, professional, and instructional. New documents should feel like a natural part of the same guide.

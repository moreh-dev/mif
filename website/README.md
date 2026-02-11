# MIF Documentation (Docusaurus)

This directory contains the Docusaurus-based documentation site for the MoAI Inference Framework (MIF).

## Local development

```bash
cd website
npm install
npm run start
```

## Production build

```bash
cd website
npm run build
```

## Versioned docs

To create a new documentation version aligned with a MIF release tag:

```bash
cd website
npx docusaurus docs:version X.Y.Z
```

Commit the generated `versioned_docs`, `versioned_sidebars`, and `versions.json` before creating the corresponding `vX.Y.Z` Git tag.

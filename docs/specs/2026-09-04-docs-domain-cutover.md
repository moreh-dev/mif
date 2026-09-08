# MIF documentation domain cutover — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Serve the MIF Docusaurus documentation at `docs.moreh.io`, retire the Retype site that occupies that domain, and remove `test-docs.moreh.io`.

**Architecture:** `docs.moreh.io` is already bound as the GitHub Pages custom domain of `moreh-dev/docs.moreh.io-published`, serving its `publish-build` branch with an approved certificate. Instead of moving that domain onto the repository MIF deploys into today (`moreh-dev/test-docs.moreh.io`), this plan retargets the MIF deployment workflow to publish onto `publish-build` itself. Neither the custom-domain binding nor the Pages configuration is touched, so no certificate is reissued, no HSTS gap opens, and the cutover needs no repository-admin permission.

**Tech Stack:** Docusaurus 3.10, `@docusaurus/plugin-client-redirects`, GitHub Actions, GitHub Pages (legacy branch build), Cloudflare DNS.

---

## Observed state

Captured 2026-09-04.

| | `test-docs.moreh.io` | `docs.moreh.io` |
| --- | --- | --- |
| DNS | `CNAME -> moreh-dev.github.io.`, TTL 300, unproxied | `CNAME -> moreh-dev.github.io.`, TTL 300, unproxied |
| Serving repository | `moreh-dev/test-docs.moreh.io` (public) | `moreh-dev/docs.moreh.io-published` (private, Pages public) |
| Pages source branch | `main` | `publish-build` |
| Certificate expiry | 2026-11-09 | 2026-10-12 |
| Domain verification | — | `protected_domain_state: verified` |
| Generator | Docusaurus 3.10 | Retype 4.4.0 |
| Last publish | 2026-09-01 | 2026-03-25 |

Facts that shape the plan:

- The repository named `moreh-dev/docs.moreh.io` does **not** serve `docs.moreh.io`. Its Pages record carries `cname: null` and serves an internal preview at `improved-adventure-l15le77.pages.github.io`. The public site is served by `moreh-dev/docs.moreh.io-published`.
- `moreh-dev/docs.moreh.io` publishes to the live site by building the `publish` branch and force-pushing the result to `publish-build` in `docs.moreh.io-published` (`.github/workflows/retype-action-publish.yml`).
- `gitgod-bot`, the identity used by the MIF docs deployment, already holds `admin` on `docs.moreh.io-published`.
- The organization is on GitHub Enterprise Cloud, which is what allows a private repository to serve a public Pages site.
- `moreh.io` links to `https://docs.moreh.io` from its header and footer navigation.
- `moreh-dev/offline_installer` (`README.md`, `tools/moai-deploy.sh`) and `moreh-dev/moai-engine` (`.agents/skills/vllm-rocm-build/SKILL.md`) link to `test-docs.moreh.io`.

---

## Why this approach

| | Option A: move the domain to `test-docs.moreh.io` | Option B: retarget the deploy at `docs.moreh.io-published` |
| --- | --- | --- |
| Custom-domain binding | Released from one repository, claimed by another | Untouched |
| Certificate | Reissued; site unreachable until approved | Untouched |
| HSTS exposure | `max-age=31556952` is already set on the domain, so visitors get an unbypassable error during the certificate gap | None |
| Removing the old site | Separate step | Implicit — the same repository starts serving the new content |
| DNS work | Delete one record | Delete one record |
| Rollback | Re-bind domain, wait for another certificate | Restore the previous branch content |

Option B is chosen. The decisive difference is the certificate gap: because `docs.moreh.io` already sends HSTS with a one-year max-age, a browser that has seen the site cannot click through a certificate error, so Option A carries a hard-outage window whose length GitHub does not guarantee.

Within Option B there are two placements for the build, and the deciding factor is permission rather than mechanism.

| | B1: deploy to `main`, repoint Pages at `main` | B2: deploy onto `publish-build`, the branch Pages already serves |
| --- | --- | --- |
| GitHub settings change | `PUT /repos/.../pages`, which requires **admin** on `docs.moreh.io-published` | None |
| Who can perform the cutover | Only a repository admin | Anyone with write, which the docs maintainers already hold |
| Who can perform the rollback | Only a repository admin | Anyone with write |
| Previous site preserved | Yes, `publish-build` is left untouched | Only if it is archived to another branch first |
| Stray Retype publish run | Harmless, it writes to a branch Pages no longer serves | **Overwrites the live site**, so the Retype workflow must be disabled first |
| Cutover moment | A deliberate settings change after the deploy is verified | The merge itself |

B2 is chosen. The docs maintainers hold `push`, not `admin`, on `docs.moreh.io-published`, so B1 makes both the cutover and every future rollback depend on reaching one of the four repository admins. A recovery path that requires someone else to be available is worse than the checkpoint B1 buys.

B2's two costs are paid up front in Task 6: the previous site is archived to a branch so rollback stays a one-command push, and the Retype publish workflow is disabled so it cannot overwrite the live site.

## Out of scope

- Migrating documentation that exists on the Retype site but not on the Docusaurus site. No task in this plan creates, moves, or evaluates that content.
- Renaming any repository. `docs.moreh.io-published` keeps its name.
- The internal Retype preview (`moreh-dev/docs.moreh.io` `main` -> `main-build`), which is unaffected by this cutover.
- Updating `moreh-dev/offline_installer` and `moreh-dev/moai-engine`, which are owned elsewhere. Task 9 records the handoff.

## File structure

| Path | Responsibility | Change |
| --- | --- | --- |
| `docs/specs/2026-09-04-docs-domain-cutover.md` | This plan | Create |
| `website/static/CNAME` | The custom domain Pages reads from the published branch | Modify |
| `website/docusaurus.config.ts` | Site URL, and the redirect map for retired Retype paths | Modify |
| `website/package.json`, `website/package-lock.json` | Adds `@docusaurus/plugin-client-redirects` | Modify |
| `.github/workflows/cd-docs.yaml` | Deployment target repository, manual trigger | Modify |
| `skills/guide-heimdall/SKILL.md` | Documentation URLs cited by the skill | Modify |
| `skills/guide-odin/SKILL.md` | Documentation URLs cited by the skill | Modify |

---

## Task 1: Confirm the deployment credential reaches the new target

`MIF_DOCS_TOKEN` is not a repository secret of `moreh-dev/mif`, so it resolves from an organization secret whose value is not readable. `gitgod-bot` holds `admin` on `docs.moreh.io-published`, but a fine-grained token can still carry a repository allowlist that excludes it. Two of the three checks are observable from an ordinary member account; the third is not, and Step 3 says what to do about that.

**Files:** none.

- [x] **Step 1: Confirm the bot's standing access to the new target**

```bash
gh api repos/moreh-dev/docs.moreh.io-published/collaborators/gitgod-bot/permission --jq '.permission'
gh api repos/moreh-dev/test-docs.moreh.io/collaborators/gitgod-bot/permission --jq '.permission'
```

Expected: `admin` from both, meaning the account's effective permission on the new target matches the target it already deploys to.

Observed 2026-09-04: `admin` and `admin`. The grant is not a direct collaboration — `repos/moreh-dev/docs.moreh.io-published/collaborators?affiliation=direct` does not list `gitgod-bot` — so it arrives through an organization or team grant, which the permission endpoint resolves authoritatively.

- [x] **Step 2: Confirm where the secret is defined**

```bash
gh api repos/moreh-dev/mif/actions/secrets --jq '.secrets[].name'
gh api repos/moreh-dev/mif/actions/organization-secrets --jq '.secrets[].name'
```

Expected: `MIF_DOCS_TOKEN` absent from the first list and present in the second, confirming it resolves from the organization.

Observed 2026-09-04: the repository holds `HF_ENDPOINT`, `HF_TOKEN`, `KUBECONFIG_BASE64`; the organization exposes `MIF_DOCS_TOKEN` to this repository. An organization-wide code search returns exactly one consumer, `moreh-dev/mif/.github/workflows/cd-docs.yaml`, so no other workflow's success can serve as evidence about how broadly the token is scoped.

- [ ] **Step 3: Resolve the token's repository scope, or let Task 6 resolve it**

The token's type and repository allowlist are visible only to an organization secrets administrator; `orgs/moreh-dev/actions/secrets` returns `403` for an ordinary member. Either of the following closes this.

**(a) Ask.** Ask whoever administers `moreh-dev` organization secrets whether the token behind `MIF_DOCS_TOKEN` is a classic PAT carrying `repo` scope, or a fine-grained PAT whose repository allowlist includes `moreh-dev/docs.moreh.io-published`. If it is fine-grained and scoped to `moreh-dev/test-docs.moreh.io` alone, the allowlist has to be extended.

**(b) Let the deployment answer it.** The cutover in Task 7 *is* the deployment. If the token cannot write to `docs.moreh.io-published` the push is rejected, the workflow goes red, and `publish-build` still holds the Retype build, so `docs.moreh.io` is unchanged.

There is therefore no gate to hold the merge behind. The failure mode is a red workflow over an unchanged site, not a broken one — a property Task 6 secures by archiving `publish-build` and disabling the Retype publish workflow before anything writes to that branch. Resolve the scope question whenever convenient.

---

## Task 2: Add client-side redirects for the retired Retype paths

The Retype site uses `snake_case` paths with no `/docs/` prefix; the Docusaurus site uses `kebab-case` under `/docs/`. Every existing `docs.moreh.io` deep link would otherwise 404. GitHub Pages cannot serve server-side redirects, so the redirects are emitted as static HTML pages at build time.

**Files:**
- Modify: `website/package.json`
- Modify: `website/package-lock.json`
- Modify: `website/docusaurus.config.ts`

- [x] **Step 1: Install the plugin at the exact version the tree already runs**

Read the core version and install that exact version. Passing a range makes npm resolve to the newest patch while the lockfile holds the rest of the tree at the older one, and npm then installs a second copy of `@docusaurus/core` and its siblings nested under the plugin. A Docusaurus plugin has to share the host's core instance.

```bash
cd website
CORE=$(node -p "require('@docusaurus/core/package.json').version")
echo "$CORE"
npm install --save "@docusaurus/plugin-client-redirects@$CORE"
```

Expected: `$CORE` prints `3.10.0`, and `package.json` gains `"@docusaurus/plugin-client-redirects": "^3.10.0"`. npm writes the caret by default, which matches how the sibling `@docusaurus/*` dependencies are declared, while the lockfile pins the exact version that `npm ci` installs in CI.

- [x] **Step 1b: Confirm the tree holds exactly one core**

```bash
cd website
npm ls @docusaurus/core @docusaurus/plugin-client-redirects
git diff --stat package.json package-lock.json
```

Expected: every `@docusaurus/core` line reads the same version, each occurrence after the first marked `deduped`, and a lockfile diff of roughly 25 added lines. A diff in the hundreds means a nested duplicate was installed — recover with `git checkout -- package.json package-lock.json && npm ci`, then redo Step 1.

Observed 2026-09-04: installing `@^3.10.0` resolved to 3.10.2 and added 403 lockfile lines carrying nested `@docusaurus/core`, `bundler`, `mdx-loader` and `utils` copies. Reinstalling at `3.10.0` gave a single deduped core and a 25-line lockfile diff.

- [x] **Step 2: Declare the redirect map**

In `website/docusaurus.config.ts`, insert this constant immediately after the `getLastStableVersion()` function and before `const config: Config = {`:

```ts
// Paths served by the Retype site that previously occupied docs.moreh.io.
// Emitted as static redirect pages because GitHub Pages has no server-side
// redirect facility.
const retypeRedirects = [
  {from: "/getting_started/overview", to: "/"},
  {
    from: "/getting_started/prerequisites",
    to: "/docs/getting-started/prerequisites",
  },
  {from: "/getting_started/quickstart", to: "/docs/getting-started/quickstart"},
  {
    from: "/getting_started/supported_devices",
    to: "/docs/reference/supported-devices",
  },
  {from: "/getting_started/logs", to: "/docs/operations/monitoring/logs"},
  {from: "/getting_started/monitoring", to: "/docs/operations/monitoring/metrics"},
  {from: "/features/preset", to: "/docs/features/preset"},
  {
    from: "/features/prefill_decode_disaggregation",
    to: "/docs/features/prefill-decode-disaggregation",
  },
  {
    from: "/features/prefix_cache_aware_routing",
    to: "/docs/features/prefix-cache-aware-routing",
  },
  {
    from: "/best_practices/container_image_caching_with_harbor",
    to: "/docs/operations/container-image-caching-with-harbor",
  },
  {
    from: "/best_practices/hf_model_management_with_pv",
    to: "/docs/operations/hf-model-management-with-pv",
  },
  {
    from: "/benchmarking/deepseek_r1_671b_on_amd_mi300x_gpus_maximum_throughput",
    to: "/blog/2025/11/11/deepseek-r1-671b-on-amd-mi300x-gpus-maximum-throughput",
  },
  {from: "/reference/heimdall_scheduler", to: "/docs/reference/heimdall/usage"},
  {
    from: "/reference/odin_inference_service",
    to: "/docs/reference/odin/api-reference",
  },
  {
    from: "/reference/odin_inference_service_template",
    to: "/docs/reference/odin/api-reference",
  },
];
```

Six Retype paths get no entry because the Docusaurus site has no corresponding page: `/features/auto_scaling`, `/features/context_length_aware_routing`, `/features/expert_parallelism`, `/features/load_aware_routing`, `/best_practices/resource_allocation`, and `/benchmarking/more_benchmarking_for_deepseek_r1_671b_on_amd_mi300x_gpus/performance_with_prefix_cache_and_load_aware_routing`. They will return 404.

- [x] **Step 3: Register the plugin**

In the same file, add this entry to the `plugins` array, after the `docusaurus-plugin-image-zoom` entry:

```ts
    [
      "@docusaurus/plugin-client-redirects",
      {
        redirects: retypeRedirects,
      },
    ],
```

- [x] **Step 4: Build and verify the redirect pages are emitted**

```bash
cd website
npm run build
```

Expected: the build succeeds. `onBrokenLinks` is `"throw"`, so a `to` value that does not resolve to a route fails the build here rather than in production.

```bash
cd website
ls build/getting_started/quickstart/index.html build/reference/heimdall_scheduler/index.html
grep -o 'http-equiv="refresh"' build/getting_started/quickstart/index.html
```

Expected: both files exist, and the `grep` prints `http-equiv="refresh"`.

- [x] **Step 5: Verify each redirect points at the destination the map names**

An emitted page proves only that the plugin ran. Read the target each page carries, compare it against the map, and confirm the target is itself built.

```bash
cd website
python3 - <<'PY'
import pathlib, re, sys
cfg = pathlib.Path("docusaurus.config.ts").read_text()
start = cfg.index("const retypeRedirects")
block = cfg[start:cfg.index("];", start)]
pairs = re.findall(r'from:\s*"([^"]+)",\s*\n?\s*to:\s*"([^"]+)"', block)
fail = 0
for frm, to in pairs:
    html = pathlib.Path("build" + frm + "/index.html").read_text()
    m = re.search(r'http-equiv="refresh"\s+content="0;\s*url=([^"]+)"', html)
    target = m.group(1) if m else None
    expected = to if to.endswith("/") else to + "/"
    dest = pathlib.Path("build" + expected.rstrip("/") + "/index.html")
    ok = target == expected and dest.exists()
    fail += 0 if ok else 1
    if not ok:
        print(f"FAIL {frm} -> {target}")
print(f"entries {len(pairs)}, mismatches {fail}")
sys.exit(1 if fail or len(pairs) != 15 else 0)
PY
```

Expected: `entries 15, mismatches 0` and exit status 0. A `FAIL` line names the entry whose emitted target diverges from the map, or whose destination was never built. The entry count is asserted too, so an entry silently dropped from the map fails here.

Observed 2026-09-04: `entries 15, mismatches 0`.

- [x] **Step 5b: Confirm the redirect pages stay out of the sitemap**

```bash
cd website
grep -c '/getting_started/quickstart/<' build/sitemap.xml
grep -o '<loc>' build/sitemap.xml | wc -l
```

Expected: `0` from the first command — redirect stubs must not be advertised as content — and `50` from the second, matching the page count of the site itself.

Observed 2026-09-04: `0` and `50`.

- [x] **Step 6: Commit**

```bash
git add website/package.json website/package-lock.json website/docusaurus.config.ts
git commit -m "MAF-20888: feat(website): redirect retired Retype paths to their Docusaurus routes"
```

---

## Task 3: Point the site at `docs.moreh.io`

`website/static/CNAME` is copied into the build output and is what GitHub Pages reads from the published branch to keep the custom domain bound. `url` in the Docusaurus config drives canonical links, Open Graph tags, and the sitemap.

**Files:**
- Modify: `website/static/CNAME`
- Modify: `website/docusaurus.config.ts:17`

- [x] **Step 1: Update the CNAME file**

```bash
echo 'docs.moreh.io' > website/static/CNAME
cat website/static/CNAME
```

Expected: `docs.moreh.io`

- [x] **Step 2: Update the site URL**

In `website/docusaurus.config.ts`, change:

```ts
  url: "https://test-docs.moreh.io/",
```

to:

```ts
  url: "https://docs.moreh.io/",
```

- [x] **Step 3: Build and verify the emitted canonical host**

`sitemap.xml` is emitted as a single line, so `grep -c` would report `1` no matter how many entries carry the host. Count occurrences instead, and sweep the whole build output rather than the sitemap alone.

```bash
cd website
npm run build
cat build/CNAME
grep -o 'https://docs\.moreh\.io' build/sitemap.xml | wc -l
grep -o 'https://test-docs\.moreh\.io' build/sitemap.xml | wc -l
grep -rl 'test-docs\.moreh\.io' build/
```

Expected: `build/CNAME` reads `docs.moreh.io`; the first count is `50`, matching the sitemap entry count; the second is `0`; and the final `grep -rl` prints nothing, so no page, feed, or search index still carries the old host.

Observed 2026-09-04: `docs.moreh.io`, `50`, `0`, and no files. Canonical links across the build resolve to `https://docs.moreh.io` only.

- [x] **Step 4: Commit**

```bash
git add website/static/CNAME website/docusaurus.config.ts
git commit -m "MAF-20888: feat(website): serve the documentation site from docs.moreh.io"
```

---

## Task 4: Retarget the deployment workflow

`docusaurus deploy` pushes the build output to `https://<GIT_USER>:<GIT_PASS>@github.com/<ORGANIZATION_NAME>/<PROJECT_NAME>.git` on `<DEPLOYMENT_BRANCH>`. Both change: the repository becomes the one that owns the `docs.moreh.io` domain, and the branch becomes the one its Pages already serves.

A `workflow_dispatch` trigger is added because the workflow currently fires only on pushes touching `website/**`. The cutover needs to be run and, if necessary, re-run on demand without an unrelated content commit.

**Files:**
- Modify: `.github/workflows/cd-docs.yaml`

- [x] **Step 1: Add the manual trigger**

In `.github/workflows/cd-docs.yaml`, change:

```yaml
on:
  push:
    branches:
      - main
    paths:
      - "website/**"
```

to:

```yaml
on:
  workflow_dispatch:
  push:
    branches:
      - main
    paths:
      - "website/**"
```

- [x] **Step 2: Change the deployment target**

In the same file, change:

```yaml
      PROJECT_NAME: "test-docs.moreh.io"
      DEPLOYMENT_BRANCH: "main"
```

to:

```yaml
      PROJECT_NAME: "docs.moreh.io-published"
      DEPLOYMENT_BRANCH: "publish-build"
```

`PROJECT_NAME` names the repository `docusaurus deploy` pushes to and `DEPLOYMENT_BRANCH` names the branch. Together they place the build on exactly the branch GitHub Pages already serves for `docs.moreh.io`, which is what removes the need for any settings change.

- [x] **Step 3: Verify the workflow still parses and carries the intended target**

```bash
grep -n 'test-docs' .github/workflows/cd-docs.yaml
python3 -c "
import yaml, json
d = yaml.safe_load(open('.github/workflows/cd-docs.yaml'))
print('triggers:', list(d[True].keys()))
print(json.dumps(d['jobs']['build']['env'], indent=2))
"
```

Expected: no output from `grep`; `triggers: ['workflow_dispatch', 'push']`; and an env block whose `PROJECT_NAME` is `docs.moreh.io-published` and `DEPLOYMENT_BRANCH` is `publish-build`. The YAML load is what catches an indentation slip in the `on:` block, which `grep` cannot see.

Observed 2026-09-04: no `grep` output, both triggers present, `PROJECT_NAME` set to `docs.moreh.io-published` and `DEPLOYMENT_BRANCH` to `publish-build`.

Note for Task 7: `workflow_dispatch` becomes selectable only once this file is on the default branch. The cutover deployment therefore comes from the push trigger on merge; the manual trigger exists for re-runs after that.

- [x] **Step 4: Commit**

```bash
git add .github/workflows/cd-docs.yaml
git commit -m "MAF-20888: chore(workflow): deploy the docs site to docs.moreh.io-published"
```

---

## Task 5: Update the documentation URLs cited by the skills

**Files:**
- Modify: `skills/guide-heimdall/SKILL.md:33-36`
- Modify: `skills/guide-odin/SKILL.md:47-51`

- [x] **Step 1: Rewrite the URLs**

```bash
sed -i 's|https://test-docs\.moreh\.io/|https://docs.moreh.io/|g' \
  skills/guide-heimdall/SKILL.md skills/guide-odin/SKILL.md
```

- [x] **Step 1b: Realign any fixed-width table the substitution touched**

The new host is five characters shorter. `skills/guide-odin/SKILL.md` pads its reference table to a fixed column width, so the substitution leaves every data row short of the header rule. `skills/guide-heimdall/SKILL.md` uses an unpadded table and needs nothing.

```bash
sed -n '/^| Topic  /,/^$/p' skills/guide-odin/SKILL.md \
  | awk 'NF {n=""; for (i = 1; i <= length($0); i++) if (substr($0, i, 1) == "|") n = n" "i; print n}' \
  | sort -u
```

Expected: exactly one line, meaning every row places its pipes at the same columns. More than one line means the table needs re-padding.

Observed 2026-09-04: the substitution left the data rows at 162 characters against a 167-character header. Re-padded to a uniform 162; the check now returns the single line `1 34 94 162`.

- [x] **Step 2: Verify the repository is clean of the old host**

```bash
git grep -n 'test-docs\.moreh\.io' -- . ':!website/node_modules' ':!website/.docusaurus' ':!docs/specs'
```

Expected: no output. If `website/versioned_docs/` reports a hit, fix it there too — versioned snapshots are served as live pages.

`docs/specs` is excluded because this plan names the retired host throughout as its subject; rewriting those mentions would make the document describe a cutover away from itself. Every other path must come back empty.

Observed 2026-09-04: no output. Every remaining mention in the repository is inside this document.

- [x] **Step 3: Commit**

```bash
git add skills/guide-heimdall/SKILL.md skills/guide-odin/SKILL.md
git commit -m "MAF-20888: docs(skills): cite the docs.moreh.io host"
```

---

## Task 6: Prepare the target before the cutover

Deploying onto `publish-build` overwrites the live site in place, which costs two protections that placing the build elsewhere would have given for free. Both are bought back here, and both must be done **before** the branch merges.

**Files:** none. This task runs against GitHub and needs only `push` permission.

- [x] **Step 1: Archive the site currently being served**

`publish-build` holds the Retype build that `docs.moreh.io` serves. `docusaurus deploy` force-pushes, so without a second ref pointing at that commit it becomes unreachable.

```bash
R=moreh-dev/docs.moreh.io-published
SHA=$(gh api repos/$R/branches/publish-build --jq '.commit.sha')
echo "$SHA"
gh api -X POST repos/$R/git/refs --input - <<JSON
{"ref": "refs/heads/retype-archive", "sha": "$SHA"}
JSON
```

Expected: the API returns the new ref with the same SHA the branch reported.

Observed 2026-09-04: `retype-archive` created at `837780c855f6b2a680db5523f15332b3f7ca532d`.

- [x] **Step 2: Confirm the archive really holds the site**

A ref pointing at the right commit is not proof the content is there. Read the tree.

```bash
gh api "repos/moreh-dev/docs.moreh.io-published/contents?ref=retype-archive" --jq '[.[].name] | join(", ")'
gh api repos/moreh-dev/docs.moreh.io-published/contents/CNAME?ref=retype-archive --jq '.content' | base64 -d
```

Expected: `index.html`, `CNAME` and the Retype section directories (`benchmarking`, `best_practices`, `features`, `getting_started`, `reference`), and a CNAME reading `docs.moreh.io`. Without the CNAME the archive would restore the content but drop the custom domain.

Observed 2026-09-04: `.nojekyll, 404.html, CNAME, benchmarking, best_practices, features, getting_started, index.html, reference, resources, robots.txt, sitemap.xml.gz, static`, and `docs.moreh.io`.

- [x] **Step 3: Disable the Retype publish workflow**

`moreh-dev/docs.moreh.io` force-pushes to `publish-build` whenever its `publish` branch is updated. Once the MIF site lives on that branch, such a push silently replaces the live site with the Retype build. Disabling the workflow is what makes the cutover durable, not housekeeping.

The internal preview workflow (`retype-action-main.yml`, which builds `main` into `main-build`) stays enabled. The internal Retype site is not part of this cutover.

```bash
gh workflow disable "Publish the final version to GitHub Pages" --repo moreh-dev/docs.moreh.io
gh api repos/moreh-dev/docs.moreh.io/actions/workflows --jq '.workflows[] | "\(.state)\t\(.name)"'
```

Expected: `Publish the final version to GitHub Pages` reads `disabled_manually`, and `Publish the working version to GitHub Pages` stays `active`.

Observed 2026-09-04: exactly that.

- [ ] **Step 4: Record the unpublished internal commits in the pull request**

```bash
gh api repos/moreh-dev/docs.moreh.io/compare/publish...main --jq '{ahead_by, behind_by}'
```

Expected: `{"ahead_by": 6, "behind_by": 1}`, unchanged from 2026-09-04. The 6 ahead are internal Retype edits that were never published to `docs.moreh.io`, so the cutover loses nothing that was live; the 1 behind is `Publish 2026/03/25 (#130)`, which exists only on `publish`. A larger `ahead_by` means more internal edits have accumulated since — still unpublished and still unaffected, but say so in the pull request.

---

## Task 7: Cut over by merging

With Task 6 done, the cutover is the merge. Merging touches `website/**`, which fires `docs-production-deploy`; the workflow builds and force-pushes onto `publish-build`; GitHub Pages rebuilds and `docs.moreh.io` serves the new site. There is no settings change and no separate switch.

This also settles Task 1 Step 3. If the token cannot write to `docs.moreh.io-published` the push fails, the workflow goes red, and `docs.moreh.io` keeps serving the Retype build untouched.

**Files:** none. This task runs against GitHub.

- [ ] **Step 1: Merge the branch and watch the deployment**

```bash
gh run list --repo moreh-dev/mif --workflow docs-production-deploy --limit 3
```

Expected: a run for the merge commit reaching `completed` with `success`. A failure on the deploy step points at the token scope — see Task 1 Step 3 — and leaves the live site unchanged.

- [ ] **Step 2: Confirm the build landed on the served branch**

```bash
R=moreh-dev/docs.moreh.io-published
gh api repos/$R/contents/CNAME?ref=publish-build --jq '.content' | base64 -d
gh api "repos/$R/contents?ref=publish-build" --jq '[.[].name] | join(", ")'
```

Expected: CNAME still reads `docs.moreh.io`, and the listing now shows the Docusaurus output — `assets`, `blog`, `docs`, `index.html`, `sitemap.xml` — rather than the Retype directories. A CNAME that came back empty or different means the deploy dropped it; restore from `retype-archive` immediately, because the custom domain depends on that file.

- [ ] **Step 3: Confirm the Pages configuration was never touched**

```bash
gh api repos/moreh-dev/docs.moreh.io-published/pages \
  --jq '{cname, source, https_enforced, cert: .https_certificate.state, expires: .https_certificate.expires_at}'
```

Expected: `cname` is `docs.moreh.io`, `source.branch` is `publish-build`, `https_enforced` is `true`, `cert` is `approved`, and `expires` is `2026-10-12` — the same certificate as before the cutover. A different expiry would mean the domain was released and re-bound somewhere, which this approach is built to avoid.

- [ ] **Step 4: Verify the live site**

Wait for Pages `status` to read `built`, then:

```bash
for p in / /docs/getting-started/quickstart/ /docs/reference/heimdall/usage/ /docs/features/preset/ /blog/; do
  printf '%-40s ' "$p"
  curl -sS -o /dev/null -w 'http=%{http_code}\n' "https://docs.moreh.io$p"
done
```

Expected: `http=200` on every line.

- [ ] **Step 5: Verify the redirects from the retired Retype paths**

```bash
for p in /getting_started/quickstart/ /features/preset/ /reference/heimdall_scheduler/; do
  printf '%-40s ' "$p"
  curl -sS "https://docs.moreh.io$p" | grep -c 'http-equiv="refresh"'
done
```

Expected: `1` on every line.

---

## Task 8: Retire `test-docs.moreh.io`

Run only after Task 7 verification passes. Until the DNS record is removed, `test-docs.moreh.io` keeps serving the last build pushed to `moreh-dev/test-docs.moreh.io`, which is a working fallback.

**Files:** none.

- [ ] **Step 1: Delete the Cloudflare DNS record**

In the Cloudflare dashboard for the `moreh.io` zone (nameservers `dion.ns.cloudflare.com`, `sky.ns.cloudflare.com`), delete the `CNAME` record `test-docs` pointing at `moreh-dev.github.io`. Requires Cloudflare access, which is outside this repository — hand off to whoever administers the zone if needed.

- [ ] **Step 2: Verify the record no longer resolves**

```bash
curl -sS -H 'accept: application/dns-json' \
  'https://dns.google/resolve?name=test-docs.moreh.io&type=CNAME' \
  | python3 -c 'import sys,json;d=json.load(sys.stdin);print("Status",d["Status"],"Answer",d.get("Answer"))'
```

Expected: `Status 3` (NXDOMAIN) and `Answer None`. The record's TTL was 300, so allow five minutes.

- [ ] **Step 3: Disable Pages on the old deployment repository**

```bash
gh api -X DELETE repos/moreh-dev/test-docs.moreh.io/pages
gh api repos/moreh-dev/test-docs.moreh.io/pages
```

Expected: the second command returns `404` — Pages is off.

- [ ] **Step 4: Archive the old deployment repository**

```bash
gh repo archive moreh-dev/test-docs.moreh.io --yes
gh repo view moreh-dev/test-docs.moreh.io --json isArchived --jq '.isArchived'
```

Expected: `true`.

---

## Task 9: Hand off the external references

Two repositories outside `moreh-dev/mif` link to `test-docs.moreh.io` and will break once Task 8 completes. They are owned elsewhere, so this task files the work rather than doing it.

**Files:** none.

- [ ] **Step 1: Open a follow-up ticket**

File a ticket listing the references to update:

| Repository | Location |
| --- | --- |
| `moreh-dev/offline_installer` | `README.md:442`, `tools/moai-deploy.sh:693` |
| `moreh-dev/moai-engine` | `.agents/skills/vllm-rocm-build/SKILL.md:109,111` |

Both need `https://test-docs.moreh.io/` rewritten to `https://docs.moreh.io/`. The path segments are unchanged, so a host-only substitution is sufficient.

- [ ] **Step 2: Link the ticket from MAF-20888**

---

## Rollback

Every row is executable with `push` permission. None of them touches the Pages configuration, so none of them waits on a certificate.

| Situation | Action | Recovery time |
| --- | --- | --- |
| The new site is broken after the cutover | `git push --force` the `retype-archive` content back onto `publish-build`, or `gh api -X PATCH repos/moreh-dev/docs.moreh.io-published/git/refs/heads/publish-build -f sha=837780c8... -F force=true` | One Pages build; domain and certificate untouched |
| The deployment workflow cannot push to the new target | Nothing to undo. The push failed, so `publish-build` still holds the Retype build and `docs.moreh.io` is unchanged. Fix the token scope and re-run | Immediate |
| A future Retype publish overwrites the live site | Re-run the MIF deployment via `workflow_dispatch`, then confirm the Retype publish workflow is still disabled | One workflow run |
| `test-docs.moreh.io` is needed again after Task 8 | Re-create the Cloudflare `CNAME` record and re-enable Pages on the archived repository | Five minutes for DNS, plus certificate reissue for the restored domain |

The first row is why Task 6 archives `publish-build` before anything writes to it. Without `retype-archive`, the previous site would survive only as an unreferenced commit.

## Completion criteria

Mapped from the ticket.

- [ ] `docs.moreh.io` serves the MIF Docusaurus site — verified by Task 7 Step 4.
- [ ] The previous Retype site is no longer served — its build is replaced on `publish-build` (Task 7 Step 2) and its publish pipeline is disabled (Task 6 Step 3).
- [ ] `test-docs.moreh.io` no longer resolves — verified by Task 8 Step 2.
- [ ] HTTPS and the main documentation paths respond correctly after propagation — verified by Task 7 Steps 3 through 5, including that the certificate is the same one as before the cutover.

# MIF documentation domain cutover — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Serve the MIF Docusaurus documentation at `docs.moreh.io`, retire the Retype site that occupies that domain, and remove `test-docs.moreh.io`.

**Architecture:** `docs.moreh.io` is already bound as the GitHub Pages custom domain of `moreh-dev/docs.moreh.io-published`, with an approved certificate. Instead of moving that domain onto the repository MIF deploys into today (`moreh-dev/test-docs.moreh.io`), this plan retargets the MIF deployment workflow at `docs.moreh.io-published` and switches that repository's Pages source branch to the one MIF publishes. The custom-domain binding is never released, so no certificate is reissued and no HSTS gap opens.

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
| Rollback | Re-bind domain, wait for another certificate | Point the Pages source branch back at `publish-build` |

Option B is chosen. The decisive difference is the certificate gap: because `docs.moreh.io` already sends HSTS with a one-year max-age, a browser that has seen the site cannot click through a certificate error, so Option A carries a hard-outage window whose length GitHub does not guarantee.

Within Option B, MIF deploys to the `main` branch of `docs.moreh.io-published` and the Pages source moves from `publish-build` to `main`, rather than overwriting `publish-build`. This keeps `publish-build` intact as a rollback snapshot, and it makes a stray run of the Retype publish workflow harmless because that workflow writes only to `publish-build`.

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

**(b) Let the deployment answer it.** Task 6 Steps 1 through 3 push to `docs.moreh.io-published` while that repository's Pages source still points at `publish-build`. A rejected push fails the workflow and leaves the live site serving the Retype build, so the question is answered at no cost.

The gate is therefore **Task 6 Step 4**, not Task 6 as a whole: do not switch the Pages source branch until Task 6 Step 2 has confirmed the build output is on `main`. Switching first would point Pages at a branch holding only the initial commit, which both breaks the site and drops the CNAME file that keeps the custom domain bound.

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

`docusaurus deploy` pushes the build output to `https://<GIT_USER>:<GIT_PASS>@github.com/<ORGANIZATION_NAME>/<PROJECT_NAME>.git` on `<DEPLOYMENT_BRANCH>`. Only `PROJECT_NAME` changes; `DEPLOYMENT_BRANCH` is already `main`, which is the branch this plan makes the new Pages source.

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
```

to:

```yaml
      PROJECT_NAME: "docs.moreh.io-published"
```

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

Expected: no output from `grep`; `triggers: ['workflow_dispatch', 'push']`; and an env block whose `PROJECT_NAME` is `docs.moreh.io-published` with `DEPLOYMENT_BRANCH` still `main`. The YAML load is what catches an indentation slip in the `on:` block, which `grep` cannot see.

Observed 2026-09-04: no `grep` output, both triggers present, `PROJECT_NAME` retargeted.

Note for Task 6: `workflow_dispatch` becomes selectable only once this file is on the default branch. Until the branch merges, the first deployment has to come from the push trigger.

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

## Task 6: Deploy to the new target, then switch the Pages source

Order matters. GitHub Pages reads the CNAME file from whichever branch is the source. If the source branch is switched to `main` while `main` still holds only the initial commit, Pages finds no CNAME file and can drop the custom domain — which is exactly the certificate reissue this plan exists to avoid. Deploy first so `main` already carries `CNAME` and `index.html`, then switch.

**Files:** none. This task runs against GitHub.

- [ ] **Step 1: Merge the branch to `main` and let the workflow run**

Open a pull request for the branch, merge it, and confirm the `docs-production-deploy` workflow starts. The merge touches `website/**`, so the push trigger fires.

```bash
gh run list --repo moreh-dev/mif --workflow docs-production-deploy --limit 3
```

Expected: the newest run is `in_progress` or `completed` with `success`. If it failed on the push step, return to Task 1 Step 3 — the token scope is the likely cause.

- [ ] **Step 2: Confirm the build output landed on the new target**

```bash
gh api repos/moreh-dev/docs.moreh.io-published/contents/CNAME?ref=main --jq '.content' | base64 -d
gh api repos/moreh-dev/docs.moreh.io-published/contents?ref=main --jq '.[].name' | head -20
```

Expected: the first command prints `docs.moreh.io`; the second lists `404.html`, `assets`, `blog`, `docs`, `index.html`, `sitemap.xml` among others.

Note: `docusaurus deploy` force-pushes, so the `README.md` currently on `main` is replaced. That file is not referenced anywhere.

- [ ] **Step 3: Record the rollback pointer**

```bash
gh api repos/moreh-dev/docs.moreh.io-published/branches/publish-build --jq '.commit.sha'
```

Expected: a commit SHA. Write it into the pull request description. `publish-build` is left untouched and is the rollback target.

- [ ] **Step 4: Switch the Pages source branch**

```bash
gh api -X PUT repos/moreh-dev/docs.moreh.io-published/pages --input - <<'JSON'
{"source": {"branch": "main", "path": "/"}}
JSON
```

- [ ] **Step 5: Verify the Pages record**

```bash
gh api repos/moreh-dev/docs.moreh.io-published/pages \
  --jq '{cname, source, status, https_enforced, cert: .https_certificate.state}'
```

Expected: `cname` is `docs.moreh.io`, `source.branch` is `main`, `https_enforced` is `true`, and `cert` is `approved`. If `cname` came back `null`, immediately re-set it — the domain must not be left unbound:

```bash
gh api -X PUT repos/moreh-dev/docs.moreh.io-published/pages --input - <<'JSON'
{"cname": "docs.moreh.io", "https_enforced": true}
JSON
```

- [ ] **Step 6: Verify the live site**

Wait for `status` to read `built`, then:

```bash
for p in / /docs/getting-started/quickstart/ /docs/reference/heimdall/usage/ /docs/features/preset/ /blog/; do
  printf '%-40s ' "$p"
  curl -sS -o /dev/null -w 'http=%{http_code}\n' "https://docs.moreh.io$p"
done
```

Expected: `http=200` on every line.

- [ ] **Step 7: Verify the redirects**

```bash
for p in /getting_started/quickstart/ /features/preset/ /reference/heimdall_scheduler/; do
  printf '%-40s ' "$p"
  curl -sS "https://docs.moreh.io$p" | grep -c 'http-equiv="refresh"'
done
```

Expected: `1` on every line.

- [ ] **Step 8: Verify the certificate was never reissued**

```bash
gh api repos/moreh-dev/docs.moreh.io-published/pages --jq '.https_certificate.expires_at'
```

Expected: `2026-10-12`, unchanged from the pre-cutover observation. A different date means the domain was released and re-bound at some point; the site still works, but note it in the pull request so the certificate window is tracked.

---

## Task 7: Stop the Retype publish pipeline

`moreh-dev/docs.moreh.io` force-pushes to `publish-build` whenever its `publish` branch is updated. With the Pages source now on `main`, such a push cannot reach the live site, but leaving the workflow enabled invites a future engineer to believe they are publishing the public site when they are not.

The internal preview workflow (`retype-action-main.yml`, which builds `main` into `main-build`) is left enabled — the internal Retype site is not part of this cutover.

**Files:** none in this repository.

- [ ] **Step 1: Disable the publish workflow**

```bash
gh workflow disable "Publish the final version to GitHub Pages" --repo moreh-dev/docs.moreh.io
```

- [ ] **Step 2: Verify it is disabled**

```bash
gh workflow list --repo moreh-dev/docs.moreh.io --all
```

Expected: `Publish the final version to GitHub Pages` shows state `disabled_manually`.

- [ ] **Step 3: Note the unpublished commits in the pull request**

`moreh-dev/docs.moreh.io` `main` is 6 commits ahead of `publish` and 1 behind it, the latter being `Publish 2026/03/25 (#130)`, which exists only on `publish`. The 6 ahead were never published to `docs.moreh.io`, so this cutover does not lose anything that was live. Record the divergence in the pull request description.

```bash
gh api repos/moreh-dev/docs.moreh.io/compare/publish...main --jq '{ahead_by, behind_by}'
```

Expected: `{"ahead_by": 6, "behind_by": 1}`, unchanged from 2026-09-04. A larger `ahead_by` means someone has written internal Retype docs since; that content is still unpublished and unaffected, but say so in the pull request.

---

## Task 8: Retire `test-docs.moreh.io`

Run only after Task 6 verification passes. Until the DNS record is removed, `test-docs.moreh.io` keeps serving the last build pushed to `moreh-dev/test-docs.moreh.io`, which is a working fallback.

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

| Situation | Action | Recovery time |
| --- | --- | --- |
| New site is broken after the source switch | `gh api -X PUT repos/moreh-dev/docs.moreh.io-published/pages --input -` with `{"source": {"branch": "publish-build", "path": "/"}}` | One Pages build; domain and certificate untouched |
| Deployment workflow cannot push to the new target | Revert the `PROJECT_NAME` change and re-run; `docs.moreh.io` keeps serving the Retype build because the Pages source has not moved yet | Immediate, provided Task 6 Step 4 has not run |
| `test-docs.moreh.io` needed again after Task 8 | Re-create the Cloudflare `CNAME` record and re-enable Pages on the archived repository | Five minutes for DNS, plus certificate reissue for the restored domain |

The rollback in the first row is why Task 6 switches the Pages source instead of overwriting `publish-build`: the previous site remains byte-for-byte intact on its own branch.

## Completion criteria

Mapped from the ticket.

- [ ] `docs.moreh.io` serves the MIF Docusaurus site — verified by Task 6 Step 6.
- [ ] The previous Retype site is no longer served — the Pages source no longer points at `publish-build` (Task 6 Step 5) and its publish pipeline is disabled (Task 7).
- [ ] `test-docs.moreh.io` no longer resolves — verified by Task 8 Step 2.
- [ ] HTTPS and the main documentation paths respond correctly after propagation — verified by Task 6 Steps 5 through 8.

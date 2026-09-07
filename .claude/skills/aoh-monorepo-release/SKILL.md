---
name: aoh-monorepo-release
description: >
  Judgment-call helper for cutting stable releases in AOH monorepos that
  use the marker-driven Changesets flow on a GitFlow base (develop →
  optional RC → main). The mechanical bumping is done by the repo's
  `pnpm release:prepare`; the publish + tag + GitHub-Release + sync-back
  is done by `.github/workflows/publish-on-main.yml` once the release
  commit lands on main. This skill owns what the maintainer must decide:
  whether the release needs an RC freeze branch or can go direct, which
  pending changesets to queue, whether their declared bump levels still
  match the actual changes, and what to put in the PR body. Trigger
  keywords: "release ui", "release the gis module", "cut a release",
  "publish stable", "promote alpha to stable", "ship X to main", "queue a
  release", "stable release", "RC branch", "release candidate", "create
  release PR".
license: Proprietary
metadata:
  owner: AOH Platform Team
  status: v3
---

# AOH Monorepo Release (marker-driven)

Help a maintainer cut a **selective per-package stable release**. Only
the packages you explicitly queue get released; everything else on
`develop` keeps flowing through continuous alpha untouched.

## TL;DR — the happy path

```bash
git checkout develop && git pull --ff-only
git checkout -b release/<pkg>-$(date +%Y%m%d)     # <pkg> = short slug (e.g. ui), not a path

ls .changeset/*.md | grep -v README               # 1. pending — read each .md's frontmatter for pkg + bump
touch .release-markers/<changeset-basename>        # 2. queue each changeset to ship
git add .release-markers/ && git commit -m "queue: <pkg> release"

pnpm release:prepare                               # 3. bump + changelog the queued ones only
git add -A && git commit -m "chore(release): @mssfoobar/<pkg>@vX.Y.Z"
git push -u origin release/<pkg>-$(date +%Y%m%d)

gh pr create --base main \                          # 4. PR develop-line -> main
  --title "chore(release): @mssfoobar/<pkg>@vX.Y.Z" --body '...'
# merge with a MERGE COMMIT (not squash) so the subject survives
```

On merge, `publish-on-main.yml` publishes, tags, creates GitHub Releases,
and opens the sync-back PR (`main → develop`) automatically. **Merge that
sync-back PR to finish** — it does not auto-merge. The rest of this skill
is the *judgment* around the above and the few things that still need a
human.

## The model — read this once

`changeset version` has two properties that make a plain run unusable for
selective releases:

1. **It's global.** There is no `--only <pkg>`. It acts on *every*
   `.changeset/*.md` and bumps *every* package they touch.
2. **It's destructive.** For each affected package it bumps
   `package.json`, appends a `CHANGELOG.md` section, and **deletes** the
   consumed `.md` files.

So a plain `changeset version` would drag every package with a pending
changeset into the release, and worse — a package you *didn't* mean to
ship gets bumped "early," its changesets consumed, so when you finally do
release it, its version has already climbed through phantom intermediate
versions and its changelog is fragmented into slivers instead of one
clump.

**The marker system fixes this with one trick** (`scripts/release/prepare.mjs`):

> Hide every changeset you're *not* releasing → run `changeset version`
> (it now only sees the ones you *are*) → put the hidden ones back.

A **marker** is just a queue entry: a zero-byte file in `.release-markers/`
whose name matches a changeset basename. `pnpm release:prepare` reads the
markers, stashes all changesets aside, hoists only the marked ones,
versions, then restores the rest. The un-queued changesets are never
touched — they stay pending for the next alpha publish and the next
release.

### What markers do NOT do

Markers govern **package versioning + the npm/tag release** of the
packages in the workflow's `release_pkgs` list. They do **not** control
where a service is *deployed*. Selective *deployment* — e.g. "ship the
new IAMS image to QA but hold GIS at its current image" — is the
`gitops/` version-vector's job (per-env overlay image pins), not markers.
Don't reach for a marker to deploy something; reach for the gitops
overlay.

### Status — this is transitional

The marker system is the **current, supported** selective-release
mechanism. It is also a deliberate workaround for a Changesets gap. The
`gitops/` version-vector (AOH-7579) is expected to move deploy
selectivity to per-env overlay pins; once *deployment* is the selective
lever, *publishing* no longer needs to be selective and Changesets can go
vanilla, retiring markers. Until that lands, **markers are the flow** —
don't invest in swapping in another release tool (release-please, nx
release) to solve a problem the version-vector is already slated to
dissolve.

## Prerequisites (this repo is already set up)

ops-hub has all three; these are just the sanity checks:

```bash
[ -d .release-markers ]                       || echo "no marker queue"
[ -f scripts/release/prepare.mjs ]            || echo "no prepare script"
[ -f .github/workflows/publish-on-main.yml ]  || echo "no publish workflow"
```

> **Why the queue lives at the repo root, not in `.changeset/`:**
> `@changesets/read` walks every *subdirectory* of `.changeset/` as a
> legacy V1 changeset and tries to read a `changes.json` inside it — even
> a dot-prefixed one — which crashes the alpha snapshot publish with
> ENOENT. Keeping markers at `.release-markers/` dodges that scan. (A
> sibling repo, `aoh-web-lib`, predates this finding and still uses
> `.changeset/_release/`; **in ops-hub it is always `.release-markers/`**.)

## Two flows: direct vs RC

The marker + `release:prepare` + merge-to-main mechanics are identical in
both. The only difference is *where the release branch lives*.

| | **Direct** (the common case) | **RC tier** (coordinated / wants a freeze) |
|---|---|---|
| When | One package, low-risk, no freeze needed | Multi-package, or wants a QA freeze window |
| Branch | `release/<pkg>-<date>` off develop | `YYYYMMDD/rc` off develop |
| Release PR | `release/<pkg>-… → main` | `YYYYMMDD/rc → main` |
| Freeze/QA | none | the RC branch is the freeze point; fixes land on the RC, not develop |
| Then | merge → `publish-on-main.yml` → sync-back PR | same |

`release/<pkg>-<date>` matches the branch ruleset. The `YYYYMMDD/rc`
shape deliberately does **not** (it starts with digits) — it's a
ruleset-excepted freeze-branch convention, not something to "fix" by
renaming.

## What to gather before queuing

1. **Which package(s).** If unclear, list pending changesets first and
   ask. Packages: `auth-sdk`, `graphql`, `logger`, `sse-client`, `ui`,
   `packages/cli`, and the modules under `modules/<name>/`
   (`gis-web-sdk`/`gis-service`/`msr-web-sdk`/`msr-service`/…), plus the
   `agents` skills subtree.
2. **Direct or RC** — infer from scope (multi-package → RC), or ask.
3. **Why now** — informs the PR body.
4. **Do the declared bumps still match the code?** Read each changeset; a
   `feat` declaring `patch`, or a `fix` with a breaking signature, is
   worth surfacing before it ships.

## The judgment calls (what this skill is for)

- **Self-containment.** Are the chosen changesets a coherent unit? A
  feature and its follow-up fix → ship together; a feature that depends
  on a peer package's unreleased change → hold.
- **Cross-package co-bumps.** If package A workspace-depends on B and you
  bump B, Changesets co-bumps A's dependency range. Such consumers get a
  version bump but are usually neither published nor tagged (see *What
  gets tagged*). This is normal; flag it so it isn't a surprise in the
  diff.
- **Private packages.** `"private": true` packages (e.g. `graphql`,
  `gis-client`, `gis-types`, the Go services, and `agents`) are **skipped
  by `changeset publish`** — nothing goes to the registry. They still get
  a version bump; whether they also get a git tag + GitHub Release depends
  only on whether they're in `release_pkgs` (below). A Go service is
  *also* shipped as a container image by its own `deploy-<svc>.yml`, keyed
  on the same `vX.Y.Z` tag. The `agents` subtree is content: its tag +
  GitHub Release **is** the release (no artifact), and the Release carries
  the `aia skills add` install snippet.
- **A service with no feature changeset** (e.g. a Go service whose work
  was tracked as code, not a `@mssfoobar/*` changeset)? Hand-author one so
  `release:prepare` bumps it. **The changeset key must be the package's
  exact `name`** — for a Go service that's the unscoped name (e.g.
  `"gis-service"`, not `@mssfoobar/...`). A mismatched key is a silent
  no-op.

  ```bash
  cat > .changeset/gis-service-release.md <<'EOF'
  ---
  "gis-service": minor
  ---

  Release gis-service.
  EOF
  touch .release-markers/gis-service-release
  ```

## What actually gets tagged

`publish-on-main.yml` cuts a git tag + GitHub Release **only** for
packages in its hard-coded `release_pkgs` list (each entry is a workspace
dir, resolved to that package's `package.json` `name` for the tag
`<name>@v<version>`):

```
packages/auth-sdk packages/graphql packages/logger packages/sse-client
packages/ui modules/gis/web modules/gis/service modules/msr/web
modules/msr/service agents packages/cli
```

Consequences worth knowing:

- `gis-client`/`gis-types` are **not** in the list → version bump only, no
  tag, no Release, no publish (they're private).
- `msr-client`/`msr-types` are also not in the list (no tag/Release) but
  **do** publish to the registry — unlike their GIS counterparts they are
  not private, because `msr-web-sdk` depends on them at runtime.
- `apps/*` and `tools/*` are deliberately omitted.
- If you add a new release-tracked package, **add its workspace dir to
  `release_pkgs`** or it will be silently never-tagged.
- **Consumable Go modules** (today `packages/aoh-golib`) live in a separate
  `go_module_pkgs` list in the same step and get a **Go-format** tag
  `<dir>/vX.Y.Z` (e.g. `packages/aoh-golib/v0.2.0`) + a GitHub Release with
  a `go get` snippet — the exact tag name Go requires to resolve a
  subdirectory module. They are deliberately NOT in `release_pkgs` (an
  npm-style `@mssfoobar/aoh-golib@v*` tag would be redundant noise). Version
  them via changesets + markers like any package; a v2+ major changes the
  module path itself (`…/aoh-golib/v2`) — read
  `openspec/changes/migrate-aoh-golib/design.md` D6 before cutting one
  (AOH-8091).

## Review the prepare diff

After `pnpm release:prepare`, before committing:

- Each version bumped `X.Y.Z` → the expected next.
- CHANGELOGs read coherently (no mangled markdown, no placeholders).
- No *unrelated* package bumped (workspace co-bumps are expected;
  unrelated bumps are not).
- Un-queued changesets are still present in `.changeset/`.

## Open the release PR → main

```bash
gh pr create --base main --title "chore(release): @mssfoobar/<pkg>@vX.Y.Z" --body '<body>'
```

The merge **commit subject** must start with `chore(release):` —
`publish-on-main.yml` reads `github.event.head_commit.message`. **Merge
with "Create a merge commit"** (not squash) so the subject is preserved
verbatim. For a multi-package release, name the lead package in the title
and list all in the body; the workflow tags every bumped package
regardless of the title.

Suggested PR body shape:

```markdown
## Releasing
- @mssfoobar/<pkg> vX.Y.Z → vX'.Y'.Z' (major|minor|patch)
- <co-bumped consumer> (dependency co-bump only — not published)

## Why
[Maintainer's framing.]

## Changesets consumed
- <changeset-basename> (bump)
- ...

## Held back
- <changeset-basename> — [why it's not in this release]

## After merge
- publish-on-main.yml publishes the npm packages, tags each bumped
  package <name>@vX.Y.Z, and creates GitHub Releases.
- <deploy-<svc>.yml builds the versioned image, if a service shipped>
- An auto-PR `main → develop` ("chore: sync release to develop") appears —
  merge it to sync develop back to parity.
```

## After merge — what's automatic, what to check

On a `chore(release):` merge, `publish-on-main.yml`:

1. Builds (`turbo run build --filter='!./modules/*/service'` — Go services
   excluded; they have no Node build).
2. `changeset publish` — publishes non-private packages whose registry
   version moved. Registry-idempotent: already-published versions are
   skipped.
3. **Tags** each `release_pkgs` package `<name>@v<version>` **iff that tag
   doesn't already exist** — idempotent and self-healing, independent of
   commit topology (AOH-7377). A missing tag from a prior partial run is
   filled in on the next release.
4. Creates a GitHub Release per freshly-tagged package (notes from its
   CHANGELOG; the `agents` Release gets the `aia skills add` snippet).
5. Opens the sync-back PR `main → develop`, using an app token so
   `changeset-check` fires and **auto-passes** via its `release-head-ref:
   main` exemption (AOH-7378).

A versioned service image (e.g. `gis-service:vX.Y.Z`) is built by that
service's `deploy-<svc>.yml` on the same release commit, keyed on the tag.

Then **merge the sync-back PR** to bring develop to parity. Verify:

```bash
git fetch origin --tags
git tag --list | grep -E "@v[0-9]|service@v"   # expected tags present
gh release list --limit 10                       # GitHub Releases present
git show origin/develop:<pkg>/package.json | grep version   # develop synced
```

## If something goes wrong

The pipeline is now self-healing for the failure classes that used to
need manual recovery (idempotent tags, app-token sync-back, Go-build
exclusion, and actions bumped off the deprecated Node 20 runtime all
landed). The few situations that still need a human:

- **A re-run of a failed publish uses the OLD workflow file.** GitHub
  re-runs execute the workflow file from the *original* commit, not
  current main — so "Re-run failed jobs" won't pick up a fix you landed on
  main. Instead, **re-trigger with a fresh `chore(release):` commit** (an
  empty one is fine — versions are already bumped and `changeset publish`
  is registry-idempotent):
  ```bash
  git checkout -b chore/retrigger-publish origin/main
  git commit --allow-empty -m "chore(release): re-trigger publish (@mssfoobar/<pkg>@vX.Y.Z)"
  git push -u origin chore/retrigger-publish
  gh pr create --base main --title "chore(release): re-trigger publish (@mssfoobar/<pkg>@vX.Y.Z)" --body '...'
  # merge with a merge commit; the tag step self-heals any tags the failed run missed
  ```
- **Wrong changeset queued, before merge.** `git rm .release-markers/<name>`
  and re-run `pnpm release:prepare`. After merge it's already consumed —
  recover with a forward follow-up release, not by deleting a published
  version.
- **Sync-back PR has conflicts** (develop moved while the release was
  open). Rebase the sync branch on develop and force-push it — it's a sync
  artefact, force-push is fine.
- **Sync-back PR still blocked** (only if the app-token mint failed and it
  fell back to `GITHUB_TOKEN`): admin-merge it —
  `gh pr merge <n> --merge --admin --subject "chore: sync release to develop (#<n>)"`.

## References

- `references/example-release.md` — the **direct, single-package** flow
  (ops-hub `@mssfoobar/ui`), happy path end to end.
- `references/example-modules-gis-sdk.md` — the **RC-tier, multi-package**
  flow (the real GIS v1.0.0 release, npm SDK + Go service in one cut),
  kept as a historical narrative including the pipeline failures that have
  since been fixed.
- The repo's `AGENTS.md → Releases` — the canonical workflow doc.
- `README.md → Versioning & Releases` — the developer-facing explainer.
- AOH-7099 — design rationale for the marker-driven flow. AOH-7579 — the
  gitops version-vector that is slated to retire it.

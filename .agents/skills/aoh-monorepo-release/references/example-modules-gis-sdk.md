# Worked example — the GIS v1.0.0 RC release from ops-hub

This is the **real, end-to-end** coordinated release that cut the GIS
module to stable from `mssfoobar/ops-hub`. It exercises the **RC tier**
flow and every failure mode the pipeline threw — and how each was
recovered. If you want the simpler single-package direct flow, see
`example-release.md`.

Packages shipped in one coordinated release:

| Package | Path | Kind | Bump |
|---|---|---|---|
| `@mssfoobar/gis-web-sdk` | `modules/gis/web` | npm (GitHub Packages) | 0.1.0 → **1.0.0** (major) |
| `@mssfoobar/ui` | `packages/ui` | npm (GitHub Packages) | 0.0.1-alpha.2 → **0.1.0** (SDK dep co-bump) |
| `gis-service` | `modules/gis/service` | Go → GHCR image (`"private": true`) | 0.0.0 → **0.1.0** |
| `@mssfoobar/gis-client` | `modules/gis/client` | private lib — **bump only** (not in tag list) | 0.1.0 → 0.2.0 |
| `@mssfoobar/gis-types` | `modules/gis/types` | private lib — **bump only** (not in tag list) | 0.1.0 → 0.2.0 |
| `reference-host`, `showcase` | `apps/*` | apps (`"private": true`) — **bump only** | 0.0.1 → 0.0.2 |

> Only `gis-web-sdk`, `ui`, and `gis-service` got git tags + GitHub
> Releases — they're the GIS-relevant entries in the workflow's
> hard-coded `release_pkgs` list. `gis-client`/`gis-types` and the two
> apps got a version bump but **no tag, no Release, no publish** (they're
> not in `release_pkgs`). See SKILL.md → **What actually gets tagged**.

Three things make this a good reference: it's multi-package, it mixes an
npm SDK with a Go container service, and the pipeline **half-failed
twice** during the cut. Both failures have since been **fixed in the
pipeline** (idempotent self-healing tags, the Go-service build exclusion,
and the app-token sync-back). The recovery steps below are kept as a
**historical narrative** — not live gotchas you should expect today — and
because the recovery *technique* (re-trigger with an empty `chore(release):`
commit) is still the right move whenever a publish job fails for any
reason. See *Lessons* at the end for what's now automatic.

> The step numbers below tell *this release's* story (the RC freeze is
> Step 6; the two failures + recovery stretch Steps 8–9). They don't map
> 1:1 onto SKILL.md's sections — SKILL.md is the canonical procedure, this
> is the narrative.

## Step 1. Verify setup — find the marker directory

```
$ [ -d .release-markers ] && echo "markers: .release-markers" || echo "?"
markers: .release-markers
$ [ -f scripts/release/prepare.mjs ] && echo ok
ok
$ [ -f .github/workflows/publish-on-main.yml ] && echo ok
ok
$ node -e "console.log(require('./package.json').scripts['release:prepare'])"
node scripts/release/prepare.mjs
```

ops-hub uses `.release-markers/` at the repo root (NOT
`.changeset/_release/` — see the V1-scanner note in SKILL.md).

## Step 2. List pending changesets, grouped

At release time there were ~70 pending changesets on develop (alpha work
in flight). The GIS-relevant ones, grouped:

```
@mssfoobar/gis-web-sdk:
  gis-sse-heartbeat-timeout      (patch)  — SSE keepalive 10s→35s
  bookmark-manager-redesign      (minor)  — MapBookmarkManager redesign
  layer-manager-redesign         (minor)  — MapLayerManager redesign
  aoh-7256-gis-sdk-refactor      (minor)  — modlet → SDK package
  aoh-7270-pr-6-cutover          (minor)  — Go backend cutover
  ... (9 GIS-exclusive changesets)

Entangled (touch GIS + shared packages):
  aoh-7255-gis-migration         (minor)  — co-bumps @mssfoobar/ui
  aoh-7322-harness-platform-fixes(minor)
  aoh-7294-pr-1-harness-scaffold (minor)
```

The "menu." Everything else (non-GIS alpha work) stays unqueued so its
alpha publishes keep firing from develop.

## Step 3. Decide what to queue

Plan: ship **all GIS work** as the v1.0.0 module release. 13 markers:
the 9 GIS-exclusive + the 3 entangled ones + the hand-authored
`gis-service-release` changeset (added in Step 4).

Judgment calls surfaced and confirmed with the maintainer:

- **The 3 entangled changesets co-bump `@mssfoobar/ui`.** That's a
  *legitimate* SDK dependency — the GIS SDK consumes `ui`, and the
  migration touched shared `ui` primitives. So `ui` rides along at a
  `minor` (0.0.1-alpha.2 → 0.1.0). `reference-host` / `showcase` (both
  `"private": true` apps) only get their dependency ranges bumped →
  **bump-only**: not published, and not tagged (apps aren't in the
  workflow's `release_pkgs` list).
- **`gis-service` has no feature changeset** (it's a Go service; its work
  was tracked as code, not a `@mssfoobar/*` changeset). Add a
  hand-written one so it bumps (Step 4).
- **Major bump for `gis-web-sdk`** (→ 1.0.0): the modlet→SDK refactor is
  a breaking API change for consumers. Confirmed: major is correct.
- **`gis-client` / `gis-types` are `"private": true`** → `changeset
  publish` skips them, and they're **not** in `release_pkgs` → they get a
  version bump only (no publish, no tag, no Release).

## Step 4. Cut the RC branch and queue markers

Multi-package + wants a QA freeze → **RC tier**.

```bash
$ git checkout develop && git pull --ff-only
$ git checkout -b 20260605/rc

# gis-service has no changeset — author one so release:prepare bumps it.
# NOTE: the key is the unscoped package name `gis-service` (its
# package.json name is "gis-service", NOT "@mssfoobar/gis-service").
$ cat > .changeset/gis-service-release.md <<'EOF'
---
"gis-service": minor
---

Release gis-service: Go backend GA (replaces legacy Java gis-app).
EOF

# Queue all 13 (12 pre-existing + the hand-authored gis-service-release;
# marker name = changeset basename, no .md).
$ touch .release-markers/gis-sse-heartbeat-timeout
$ touch .release-markers/bookmark-manager-redesign
$ touch .release-markers/layer-manager-redesign
$ touch .release-markers/aoh-7256-gis-sdk-refactor
$ touch .release-markers/aoh-7270-pr-6-cutover
$ touch .release-markers/gis-service-release
$ touch .release-markers/aoh-7255-gis-migration
$ touch .release-markers/aoh-7322-harness-platform-fixes
$ touch .release-markers/aoh-7294-pr-1-harness-scaffold
# ... (the rest of the GIS-exclusive set)

$ git add .changeset/gis-service-release.md .release-markers/
$ git commit -m "queue: gis module v1.0.0 release"
```

Branch `20260605/rc` uses the `YYYYMMDD/rc` shape — a ruleset-excepted
naming convention for freeze branches (it does NOT match the
feature-branch regex; see SKILL.md → Two flows: direct vs RC).

## Step 5. Run prepare

The 13 markers correspond to 12 pre-existing changesets + the 1
hand-authored `gis-service-release`, so after authoring it there are 71
changesets on disk; 13 are hoisted and 58 restored:

```bash
$ pnpm release:prepare
Processing 13 marker(s): ...
Stashing 58 top-level changeset(s) aside
Hoisting 13 selected changeset(s)
Running `changeset version`...
🦋  major: @mssfoobar/gis-web-sdk
🦋  minor: @mssfoobar/ui, @mssfoobar/gis-client, @mssfoobar/gis-types, gis-service
🦋  patch: (folded)
Restoring 58 unselected changeset(s)
Clearing 13 marker(s)
```

Diff (reviewed and approved):

```
 modules/gis/web/package.json       0.1.0 → 1.0.0
 modules/gis/service/package.json   0.0.0 → 0.1.0
 packages/ui/package.json           0.0.1-alpha.2 → 0.1.0
 modules/gis/client/package.json    0.1.0 → 0.2.0
 modules/gis/types/package.json     0.1.0 → 0.2.0
 apps/reference-host/package.json   0.0.1 → 0.0.2   (dep-range bump only)
 apps/showcase/package.json         0.0.1 → 0.0.2   (dep-range bump only)
 + CHANGELOG.md entries for each
 - 13 consumed .changeset/*.md deleted
```

The 58 unqueued changesets are still present → develop's alpha publishes
keep firing for non-GIS work.

```bash
git add -A
git commit -m "chore(release): prepare gis-web-sdk@v1.0.0, gis-service@v0.1.0, ui@v0.1.0"
git push -u origin 20260605/rc
```

## Step 6. Freeze + QA on the RC branch

`20260605/rc` is the freeze point. QA ran against it (deployed the
`develop-<sha>` images built from the RC, exercised the GIS map / live
feed / bookmarks against the demo tenant). Any fix during the freeze
would land on the RC branch, not develop. QA passed.

> Two shared workflows were also **extended on the RC** (approved as a
> pipeline evolution): `publish-on-main.yml` gained the `modules/gis/service`
> tag-loop entry + a "Create GitHub Releases" step; `deploy-gis-service.yml`
> gained `main` as a push branch + versioned `:vX.Y.Z` image on a
> `chore(release):` main commit.

## Step 7. Open the release PR → main

```bash
$ gh pr create --base main --title \
  "chore(release): gis-web-sdk@v1.0.0, gis-service@v0.1.0, ui@v0.1.0" --body '...'
```

PR #104. Merged with **"Create a merge commit"** so the subject
`chore(release): gis-web-sdk@v1.0.0, gis-service@v0.1.0, ui@v0.1.0 (#104)`
is preserved for `publish-on-main`'s `head_commit.message` check.

## Step 8. Merge → and here's where it half-failed (twice)

> **Historical.** Both failures below are now fixed in the pipeline (see
> *Lessons*). They're preserved because the recovery *technique* still
> applies whenever a publish job fails for any reason.

**Failure 1 — publish job died at the Build step.** `publish-on-main`'s
`pnpm run build` ran `turbo run build` across the whole repo, which
included `gis-service` whose `build` is `go build` — no Go in the
Node/pnpm publish job. It failed **before** publish/tag/release. No npm
publish, no tags, no Releases. (The separate `deploy-gis-service.yml`
job *did* succeed — it built + pushed
`ghcr.io/mssfoobar/ops-hub/gis-service:v0.1.0`.)

Fix (PR #105, `fix(ci):` subject so it does NOT trigger publish):
```diff
- run: pnpm run build
+ run: pnpm exec turbo run build --filter='!gis-service'
```

**Re-run won't help** — GitHub re-runs use the *original* commit's
workflow file, which lacks the fix (SKILL.md → If something goes wrong). So
re-trigger with an empty `chore(release):` commit on fixed main:

```bash
git checkout -b chore/retrigger-gis-publish origin/main
git commit --allow-empty -m "chore(release): re-trigger GIS publish (gis-web-sdk@v1.0.0, ui@v0.1.0, gis-service@v0.1.0)"
git push -u origin chore/retrigger-gis-publish
gh pr create --base main --title "chore(release): re-trigger GIS publish ..." --body '...'
```

PR #106 merged (merge commit, `chore(release):` subject). **Publish
succeeded:** `@mssfoobar/gis-web-sdk@1.0.0` + `@mssfoobar/ui@0.1.0`
published to GitHub Packages. `changeset publish` is registry-idempotent
— it skipped already-published `auth-sdk`/`logger`/`sse-client`.

**Failure 2 — tags + GitHub Releases silently didn't fire.** The tag
step diffs `package.json` at `HEAD` vs `HEAD~1`. But the re-trigger was
an *empty* commit, so its `HEAD~1` (the #105 fix commit) already had
versions at 1.0.0/0.1.0 → `current == previous` → it tagged nothing.
(Tracked: **AOH-7377**.)

Recovery — created the 3 tags + Releases manually on main HEAD:
```bash
$ MAIN_SHA=$(git rev-parse origin/main)   # 7af5d64...
$ for t in "@mssfoobar/gis-web-sdk@v1.0.0" "@mssfoobar/ui@v0.1.0" "gis-service@v0.1.0"; do
    git rev-parse -q --verify "refs/tags/$t" >/dev/null || git tag -a "$t" "$MAIN_SHA" -m "$t"
  done
$ git push origin "@mssfoobar/gis-web-sdk@v1.0.0" "@mssfoobar/ui@v0.1.0" "gis-service@v0.1.0"

$ extract() { awk '/^## / { if (seen) exit; seen=1; next } seen { print }' "$1"; }
$ extract modules/gis/web/CHANGELOG.md > /tmp/n.md
$ gh release create "@mssfoobar/gis-web-sdk@v1.0.0" --title "@mssfoobar/gis-web-sdk v1.0.0" \
    --notes-file /tmp/n.md --target "$MAIN_SHA"
# repeated for @mssfoobar/ui@v0.1.0 and gis-service@v0.1.0
```

Releases were created for the **published** packages + the image-deployed
`gis-service` (provenance) — i.e. exactly the GIS entries in
`release_pkgs`. `gis-client`/`gis-types` got **neither a tag nor a
Release** (they're not in `release_pkgs`); they only carry a version bump
into develop via the sync-back.

## Step 9. Merge the sync-back PR — needs `--admin`

`publish-on-main` auto-opened PR #107 (`main → develop`,
"chore: sync release to develop") carrying the version bumps, CHANGELOGs,
the 12 consumed-changeset deletions, AND the two workflow CI improvements.

```bash
$ gh pr merge 107 --merge
X base branch policy prohibits the merge.
```

develop's ruleset requires `check / Check` (from `changeset-check.yml`),
which **never ran** (sync PR opened by `GITHUB_TOKEN` → no triggers) and
**couldn't pass anyway** (it requires an *added* changeset; a sync-back
only deletes them). (Tracked: **AOH-7378**.) Admin-merge is the
sanctioned path:

```bash
gh pr merge 107 --merge --admin --subject "chore: sync release to develop (#107)"
```

Verified develop reached parity:
```bash
$ git show origin/develop:modules/gis/web/package.json | grep version
  "version": "1.0.0"
# consumed GIS changesets gone; publish-on-main + deploy-gis-service CI fixes present
```

## Final state

- **GitHub Packages:** `@mssfoobar/gis-web-sdk@1.0.0`, `@mssfoobar/ui@0.1.0`
- **GHCR:** `ghcr.io/mssfoobar/ops-hub/gis-service:v0.1.0`
- **Tags + GitHub Releases:** all three published components
- **develop:** synced (bumps + changeset deletions + CI improvements)
- Worktrees/branches for the RC, the fix, and the re-trigger cleaned up.

## Lessons — now baked into the pipeline

Items 1–4 were the failures this release hit. **All four are now fixed in
the workflow**, so they read as the rationale behind the current pipeline,
not as live risks:

1. **Go services are excluded from the publish build**
   (`--filter='!./modules/*/service'` — directory glob, covers future
   modules). *(landed)*
2. **Re-runs use the old workflow file** — so a failed publish is
   re-triggered with a fresh `chore(release):` commit, not "re-run failed
   jobs". *(still true — GitHub behaviour, not a bug)*
3. **Tags are idempotent + self-healing** — the workflow tags each
   `release_pkgs` package iff its tag is missing, independent of commit
   topology. The old `HEAD~1` diff that silently tagged nothing is gone
   (AOH-7377, landed).
4. **The sync-back PR auto-passes** — it's opened with an app token so
   `changeset-check` fires and the `release-head-ref: main` exemption
   passes it; `--admin` is only the fallback if the app-token mint fails
   (AOH-7378, landed).
5. **A coordinated module release can mix npm packages + a container
   service** in one PR — the npm side goes through `changeset publish`,
   the service through its `deploy-<svc>.yml`, both keyed on the same
   `vX.Y.Z` tag.

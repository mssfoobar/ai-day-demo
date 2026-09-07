# Worked example — the DIRECT (no-RC) flow: `@mssfoobar/ui` from ops-hub

A complete walk-through of the **direct, single-package** variant — branch
off develop, queue markers, PR straight to main. No RC freeze branch. Use
this when the release is one package, low-risk, and doesn't need a QA
freeze window.

> For the **RC tier** (coordinated / multi-package / wants a freeze), see
> `example-modules-gis-sdk.md` — the real GIS v1.0.0 release. The marker
> queue, `release:prepare`, and merge-to-main mechanics are identical; the
> RC flow just inserts a `YYYYMMDD/rc` freeze branch between develop and
> main.

Scenario: "ship the UI bottom-sheet work, but hold the auth-sdk fixes —
they aren't ready." One package, no freeze needed → the direct flow fits.

## Starting state on `develop`

```
$ ls .changeset/*.md | grep -v README
.changeset/auth-sdk-fix.md
.changeset/auth-sdk-typescript-6.md
.changeset/ui-bottom-sheet.md
.changeset/ui-multi-select-fix.md
.changeset/ui-types-node-align-24.md
```

Five pending changesets — three for `@mssfoobar/ui`, two for
`@mssfoobar/auth-sdk`. Both packages have been firing alpha publishes on
every push to develop. The plan: ship all three UI changesets, hold both
auth-sdk ones (awaiting verification against a downstream consumer).

## Step 1. Sanity-check the setup

```
$ [ -d .release-markers ] && echo ok          # ops-hub queue dir (repo root)
ok
$ [ -f scripts/release/prepare.mjs ] && echo ok
ok
$ [ -f .github/workflows/publish-on-main.yml ] && echo ok
ok
```

ops-hub uses `.release-markers/` at the repo root. (Not
`.changeset/_release/` — see the V1-scanner note in SKILL.md.)

## Step 2. List pending changesets, grouped

```
@mssfoobar/ui (3 pending):           [packages/ui]
  ui-bottom-sheet         (minor)  — UI bottom-sheet primitive
  ui-multi-select-fix     (patch)  — Multi-select chip overflow indicator
  ui-types-node-align-24  (patch)  — Bump @types/node to 25

@mssfoobar/auth-sdk (2 pending):     [packages/auth-sdk]
  auth-sdk-fix            (patch)  — Token refresh edge case
  auth-sdk-typescript-6   (patch)  — Bump typescript to 6.0.3
```

## Step 3. Decide what to queue

Ship all three UI changesets; hold both auth-sdk ones. Bump sanity-check:

- `ui-bottom-sheet` declares **minor** — adds a new primitive → correct.
- `ui-multi-select-fix` declares **patch** — bugfix in existing behaviour
  → correct.
- `ui-types-node-align-24` declares **patch** — a devDep bump that doesn't
  change runtime → `patch` is fine.

Final bump for `@mssfoobar/ui`: **minor** (the largest of the three),
`0.2.5 → 0.3.0`.

## Step 4. Branch from develop and queue markers

```bash
git checkout develop && git pull --ff-only
git checkout -b release/ui-20260508

touch .release-markers/ui-bottom-sheet
touch .release-markers/ui-multi-select-fix
touch .release-markers/ui-types-node-align-24

git add .release-markers/
git commit -m "queue: ui release"
```

Branch name `release/ui-20260508` matches the ruleset regex.

## Step 5. Run `pnpm release:prepare`

```
$ pnpm release:prepare

Processing 3 marker(s):
  - ui-bottom-sheet
  - ui-multi-select-fix
  - ui-types-node-align-24

Stashing top-level changesets to /tmp/aoh-changeset-stash-XXXX/
Hoisting 3 selected changeset(s) for processing

Running `changeset version`...
---
🦋  Packages to be bumped at minor:
🦋  @mssfoobar/ui
🦋  Done generating changelog
---
Restoring 2 unselected changeset(s) from stash
Clearing 3 marker(s) from .release-markers/

Done. Review the changes.
```

## Step 6. Review the diff

```
$ git status
On branch release/ui-20260508
Changes not staged for commit:
  modified:   packages/ui/CHANGELOG.md
  modified:   packages/ui/package.json
  deleted:    .changeset/ui-bottom-sheet.md
  deleted:    .changeset/ui-multi-select-fix.md
  deleted:    .changeset/ui-types-node-align-24.md
```

`auth-sdk-fix.md` and `auth-sdk-typescript-6.md` are **not** in the diff —
they stayed in `.changeset/` and keep firing alpha publishes from develop
after the release lands.

```diff
$ git diff packages/ui/package.json
-  "version": "0.2.5",
+  "version": "0.3.0",

$ git diff packages/ui/CHANGELOG.md
+## 0.3.0
+
+### Minor Changes
+- UI bottom-sheet primitive.
+
+### Patch Changes
+- Multi-select chip overflow indicator.
+- Bump @types/node to 25.
+
 ## 0.2.5
```

All three changesets folded into one `0.3.0` entry — the consolidated
clump, not three slivers. That is the whole point of selective release.

## Step 7. Commit and open the release PR

```bash
$ git add -A
$ git commit -m "chore(release): @mssfoobar/ui@v0.3.0"
$ git push -u origin release/ui-20260508

$ gh pr create --base main --title "chore(release): @mssfoobar/ui@v0.3.0" --body '
## Releasing
- @mssfoobar/ui v0.2.5 → v0.3.0 (minor)

## Why
Bottom-sheet primitive is needed by amm and dash; multi-select fix
unblocks the iams admin UI work. The @types/node alignment is hygiene.

## Changesets consumed
- ui-bottom-sheet (minor)
- ui-multi-select-fix (patch)
- ui-types-node-align-24 (patch)

## Held back
- auth-sdk-fix — pending verification against amm
- auth-sdk-typescript-6 — pending coordination with portal

## After merge
- publish-on-main.yml publishes @mssfoobar/ui@0.3.0, tags
  @mssfoobar/ui@v0.3.0, and creates the GitHub Release.
- An auto-PR main → develop ("chore: sync release to develop") appears —
  merge it to sync.
'
```

## Step 8. Merge → the pipeline runs itself

Get one approval (the ruleset requires it before merge to `main`), then
merge with **"Create a merge commit"** so the subject
`chore(release): @mssfoobar/ui@v0.3.0` is preserved. `publish-on-main.yml`
then:

1. Detects the `chore(release):` subject → proceeds.
2. `changeset publish` → only `@mssfoobar/ui@0.3.0` is published
   (per-package; the held auth-sdk versions didn't move).
3. Tags `@mssfoobar/ui@v0.3.0` iff it doesn't already exist (idempotent),
   pushes it, and creates the GitHub Release.
4. Opens the sync-back PR `main → develop` via an app token, so
   `changeset-check` fires and auto-passes the `release-head-ref: main`
   exemption.

## Step 9. Merge the sync-back PR, then verify

```bash
$ gh pr merge <n> --merge          # no --admin needed on the happy path

$ git fetch origin --tags
$ git tag --list | grep ui          # @mssfoobar/ui@v0.3.0 present?
$ gh release list --limit 5         # GitHub Release present?
$ git show origin/develop:packages/ui/package.json | grep version
  "version": "0.3.0"
```

Afterward, develop's alpha publishes for `ui` resume from `0.3.0` forward
(paused until a new ui changeset lands), while `auth-sdk` keeps firing
alphas — both its changesets are still pending and release stable when you
queue their markers in a future release.

## If a step trips

The pipeline self-heals for the classic failure classes (missing tags are
filled on the next run; the sync-back auto-passes its check). The two
things that still need a human: a failed publish must be **re-triggered
with a fresh `chore(release):` commit** (a re-run reuses the old workflow
file), and a **wrong marker before merge** is fixed with
`git rm .release-markers/<name>` + re-run `pnpm release:prepare`. See
SKILL.md → *If something goes wrong*.

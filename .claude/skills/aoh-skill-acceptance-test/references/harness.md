# The harness — builder prompt template + evaluation rubric

## Before you spawn the builder

Set up so the run is realistic and the blockers are attributable:

- **Isolate.** Build in a disposable location (a worktree, or a throwaway
  `apps/<name>` you delete after). Never the primary checkout.
- **Decide the dependency story.** If the skill's package is unpublished /
  in-monorepo, the builder will consume it as `workspace:*`; note that so a
  resolution failure isn't misread as a skill gap.
- **Know the ground truth yourself** (read the module source / reference
  impl) so you can grade the builder's claims. The *builder* can't read it;
  *you* must.

## Builder subagent prompt template

Spawn a fresh `general-purpose` agent (background is fine). Fill every
**{SLOT}** (the six are listed under the template). The hard constraints
are what make the test valid — keep them verbatim.

```text
You are a developer who has just been handed a set of AOH agent skills and asked to
{TASK}. You have NEVER seen the internals of the module you're integrating. This is a
skill-quality test: we are measuring whether the skills ALONE are enough for you to succeed.

## Working location
{WORKING_DIR}   (a disposable, isolated checkout — do all work here)

## Your ONLY documentation (read these; they are your spec)
- {BASE_SKILL_DIR}        — scaffold the base app with this skill (SKILL.md + references/ + assets/)
- {SKILL_UNDER_TEST_DIR}  — the skill under test: integrate it into the scaffolded app

## HARD CONSTRAINTS (these make the test valid — violating them invalidates it)
- Do NOT open, read, grep, cat, or list any of these — treat them as if they don't exist
  (a real external consumer wouldn't have them):
{FORBIDDEN_PATHS}
  You MAY depend on the package(s) the skill wires (e.g. via workspace:* if in-monorepo),
  but you must learn their API ONLY from the skill — never by reading their source.
  "Their source" includes the INSTALLED package under `node_modules/<pkg>/**` — its
  `dist/`, bundled `*.d.ts` types, and `package.json` `exports`. Do not open or grep those
  to recover an API the skill didn't give you (your editor/`svelte-check` may load `.d.ts`
  automatically — that's fine; deliberately reading them to fill a doc gap is not). If the
  skill's prose/snippets don't tell you how to call something, that is a skill gap to
  RECORD, not to resolve by reading types.
  When a skill says "Ref: <some repo path>", treat it as an external pointer you cannot
  access; rely on the skill's own prose/snippets instead.
- Log EVERY file you read (path + why), so we can verify you stayed within the skills.
- Do NOT git commit/add/push. Only edit files the skills tell you to; list any
  workspace-root files you change.

## What to do
1. Follow {BASE_SKILL} end-to-end to scaffold the app.
2. Follow the skill under test end-to-end to integrate the module.
3. Get it as close to ready-to-run as the environment allows: install, typecheck/svelte-check,
   build, and (if feasible) a backend-free / stub dev run. Time-box the build: if you hit
   registry/peer/environment walls clearly NOT about the skill under test, capture the exact
   error and move on — don't grind on environment issues.

## Deliverables — return a structured report
1. Files-read log — every file you opened (proof you stayed within the skills).
2. Execution log — ordered steps, each mapped to the skill instruction it came from.
3. Integration inventory — the files you created/edited for the integration (path + 1-line
   purpose), with the key snippets pasted (provider/mount, one wiring route, the core
   rendering/usage, any adapter).
4. Skill friction log (MOST IMPORTANT) — every place a skill was ambiguous, missing a step,
   wrong, or forced you to guess/infer. Quote the skill line and say exactly what you needed.
   Categorise each as [SKILL-UNDER-TEST] / [BASE-SKILL] / [ENVIRONMENT].
5. Build/run result — what passed, what failed, exact errors, and your attribution
   (skill gap vs environment).
6. Verdict — Using ONLY the skills, could you produce a correct, working setup? Yes / Partially
   / No, and why. Separately: assuming a working base + resolvable deps, was the skill UNDER
   TEST itself sufficient and accurate? (Yes / Partially / No + the specific gaps.)

Be honest and critical — a glowing report that hides friction is worse than useless.
```

Filling the slots (MSR run, for reference) — all six:

- `{TASK}` → "stand up a brand-new AOH web app with the MSR (Multi-Session Replay) module integrated and ready to use"
- `{WORKING_DIR}` → the disposable checkout from the *Isolate* step (e.g. a worktree at `../<repo>-spike`, or a throwaway `apps/<name>` you delete after)
- `{BASE_SKILL}` → `aoh-web-init`
- `{BASE_SKILL_DIR}` → `agents/skills/aoh-web-init/`
- `{SKILL_UNDER_TEST_DIR}` → `agents/skills/aoh-msr-integration/` (the worked example below)
- `{FORBIDDEN_PATHS}` → `modules/msr/**`, `modules/gis/**`, `apps/reference-host/**`, the `aoh-gis-integration` skill

## Evaluation rubric (you, with ground-truth access)

Grade five things; cite the source file for every judgement.

0. **Cold-run integrity (do this FIRST — it gates whether the run counts).**
   Audit the files-read log against the forbidden set: any `{FORBIDDEN_PATHS}`
   entry, or any read of `node_modules/<wired-pkg>/**` source/`dist`/`.d.ts`,
   invalidates the run — re-run cold. Also scan for *silent* leaks: any API the
   builder used correctly that has **no corresponding snippet in the skill**
   (it came "from nowhere") is a sign of an unlogged read — treat the run as
   suspect. A clean cold run is the precondition for trusting the verdict below.
1. **Completeness.** Did the builder produce every artifact the integration
   needs (provider/mount, wiring, config, the consumer-side usage, the
   backend-free path)? Missing artifacts the skill never mentioned → skill
   gap.
2. **Gotcha-handling.** Did the documented pitfalls actually transfer? (For
   MSR: did it render the entities itself? get the z-index/layout right?)
   If a "gotcha" still bit the builder, the warning wasn't clear enough.
3. **The "compiles ≠ works" check (the high-value one).** Don't accept
   "svelte-check passed" as success. For anything whose *body* isn't type
   checked (HTTP responses, stubs, env wiring), verify the builder's output
   would actually **run** — reproduce it, or check the field names/shapes
   against ground truth. The MSR stub bug was invisible to every static
   gate.
4. **Gap attribution.** For each friction-log item, confirm the category.
   Re-tag anything the builder mis-attributed. Only `[SKILL-UNDER-TEST]`
   items gate this PR.

For each confirmed `[SKILL-UNDER-TEST]` gap, **reproduce it** (the failing
stub, the unresolved import, the inaccessible reference), classify
Critical/Medium/Low by "does a consumer following the skill hit a wrong
outcome", fix Critical+Medium on the branch, and **re-verify the fix**
against the real dependency.

## Reading the verdict

- **"Yes" (skill sufficient):** merge after fixing any Low nits you got for
  free.
- **"Partially — skill gaps":** fix them on the branch; re-run or
  spot-verify; then merge.
- **"Partially / No — environment or base-skill only":** the skill under
  test passes; file the base/environment issues separately and merge.

The failure you're hunting is the one where the builder reports "it builds
and lints" and the verdict is a confident "Yes" — but step 3 of the rubric
shows it wouldn't actually run. That gap is the entire reason this test
exists.

## Internal-leakage scan

A consumer installs these skills into their OWN project and has no access to the
`mssfoobar/ops-hub` monorepo — so a skill that points at ops-hub paths, issue
numbers, or the repo itself hands the consumer a dead reference. The cold run
catches this only when the builder happens to hit an unreachable pointer; this
scan catches **every** occurrence, incl. inside shipped `assets/` and the
frontmatter that feeds `registry.json`. It is cheap and deterministic — an agent
can run it in seconds.

### Deny set — any hit is a finding

Scan the whole skill dir (`SKILL.md` + `references/` + `assets/` + `scripts/`):

| Reference | Why it leaks | Fix |
|---|---|---|
| `apps/reference-host…` | the reference impl — a consumer can't read it | inline the snippet, or cite the public API |
| `modules/<m>/{service,web,client,types,worker-template}…` | module **source** paths | cite the published package + subpath export |
| `AOH-####` | internal issue numbers — meaningless to a consumer | drop them (keep the substance) |
| `mssfoobar/ops-hub` | the monorepo itself — the consumer's project isn't in it | use the consumer's own path / a public name |

One tool-agnostic check an agent can run from `agents/` (flag candidates, then
discard the allowed published-image refs):

```bash
grep -rInE 'apps/reference-host|modules/[a-z0-9-]+/(service|web|client|types|worker-template)|AOH-[0-9]+|mssfoobar/ops-hub' skills/<skill> \
  | grep -v 'ghcr.io/mssfoobar/ops-hub/'
```

### Allowed — NOT leakage

- **Published names**: `@mssfoobar/<pkg>` packages and `ghcr.io/mssfoobar/ops-hub/<img>`
  image names — a consumer installs the package and runs the image (hence the
  `grep -v 'ghcr.io/mssfoobar/ops-hub/'` above).
- **The consumer's own paths**: `src/routes/…`, `$lib/…`, and the files the skill
  tells them to create.
- **Public sibling skills**: `aoh-web-init`, `aoh-knowledge`, etc.
- **Scaffolder setup the consumer owns**: in `aoh-web-init` (and other scaffolders)
  the `turbo` / `injectWorkspacePackages` / `pnpm-workspace.yaml` mentions are the
  *consumer's* monorepo config — keep them.

### Eyeball too (context-dependent — not in the deny set)

Registry-consumer-irrelevant build mechanics — `workspace:*`,
`pnpm -F @mssfoobar … run package`, the bare `svelte-package` tool name — leak in an
*integration* skill (the consumer `pnpm add`s the published package) but are
legitimate in a *scaffolder*. Judge per skill; don't hard-flag.

### Exempt skills

Platform-team-only skills, never installed into a consumer project, that document
ops-hub itself: **this skill**, `aoh-monorepo-release`, `aoh-agent-context`,
`aoh-scripting-conventions`, `aoh-spoke`, `aoh-hub-nexus`, `aoh-deploy`. Don't scan
them — the deny-set tokens are legitimate there.

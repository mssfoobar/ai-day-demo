---
name: aoh-skill-acceptance-test
description: >
  Run a cold-execution acceptance test on an AOH **integration** skill
  before merge — a fresh subagent executes the skill with **no access to
  the module source or reference implementation it was distilled from**, on
  a realistic consumer base, to catch gaps doc-review structurally cannot:
  docs that point at files the consumer can't reach, and code that
  compiles/lints clean but never runs. Use when authoring or reviewing an
  AOH integration skill (`aoh-*-integration` and similar SDK-wiring
  skills), validating a skill before merge, asking "is this skill
  sufficient for a real consumer", testing whether an agent can execute a
  skill cold, or running the skill acceptance gate. Trigger phrases:
  "acceptance test this skill", "cold-execution test", "is my skill
  sufficient", "test the skill end-to-end", "validate the skill before
  merge", "will a naive agent succeed with only this skill", "skill
  acceptance gate", "dogfood the skill".
license: Proprietary
metadata:
  owner: AOH Platform Team
  status: v1
---

# AOH skill acceptance test

A **cold-execution acceptance test** is the merge gate for AOH
**integration** skills — skills that wire an SDK or service into a
consumer app (the `aoh-*-integration` family, and anything else whose job
is "here's how to consume package X"). A fresh agent executes the skill
**using only the skill as documentation**, then the author grades whether
the skill alone was enough.

## Why this exists (the doc-review blind spot)

Static review — even a thorough multi-agent pass — checks a skill's
snippets against the **source-of-truth repo** (the module source, the
reference-host, the sibling skill the snippets were distilled from). In
that repo the snippets are correct *by construction*. But a real consumer:

- is on a **different base** (usually an `aoh-web-init` scaffold), whose
  auth module, route groups, and types differ from the reference impl; and
- **cannot see** the module source or reference-host the skill was written
  against.

So two whole classes of defect are invisible to doc-review and only appear
under cold execution:

1. **"The doc points at something the consumer can't reach."** e.g. "copy
   `apps/reference-host/.../msr-stub.ts`" — inaccessible outside the
   platform monorepo.
2. **"Compiles and lints clean, but never runs."** e.g. a stub whose
   response body has the wrong field names — the handler *type* checks the
   signature, not the JSON body, so it builds green and renders nothing.

The MSR run (see Worked example) hit exactly these after three doc
reviewers signed off. That's the whole point: **"compiles + lints" ≠
"works", and "correct against the source repo" ≠ "usable by a consumer".**

## The method

**First — the internal-leakage scan (every consumer-installed skill).** Before the
cold run, apply the internal-leakage rule (`references/harness.md` →
*Internal-leakage scan*): a cheap, deterministic check for ops-hub-internal
references a consumer can't reach — `apps/reference-host`, module-source paths,
`AOH-####`, the `mssfoobar/ops-hub` repo (published `@mssfoobar/*` package and
`ghcr.io/mssfoobar/ops-hub/*` image names are fine). It applies to *every*
installable skill, including scaffolders/others that skip the cold run below, and
must be clean before merge.

Then the cold-execution acceptance test (integration / SDK-wiring skills):

1. **Pick a realistic task + base.** Usually: scaffold a fresh app with
   `aoh-web-init`, then integrate the skill under test. Use the consumer
   base the skill actually targets — not the reference-host.
2. **Spawn a naive builder subagent** (`general-purpose`, fresh context)
   with the **skill-only constraint**: its sole documentation is the skill
   files; it is **forbidden** from reading the module source, the
   reference implementation, and the sibling skill the snippets came from
   (it may *depend on* a workspace package, but not read its source to
   learn the API). It logs every file it reads. Copy the prompt template
   from `references/harness.md` and fill in its slots.
3. **Let it run end-to-end** and get as close to ready-to-run as the
   environment allows (`install` → `svelte-check`/typecheck → `build` →,
   if feasible, a stub-mode `dev`).
4. **Collect its report:** files-read log (proof it stayed in-bounds),
   execution log, integration inventory, **friction log** (every
   ambiguity/missing-step/wrong-snippet, each tagged by which skill or the
   environment), build/run result, and a verdict.
5. **Evaluate against ground truth** (you *can* read the source). Use the
   rubric in `references/harness.md`: separate real skill gaps from base
   /environment noise, and **verify each claimed gap** — reproduce the
   "compiles-but-broken" failure, or confirm the inaccessible reference.
6. **Fix real gaps on the branch, then re-verify the fix** the same way
   the consumer would hit it (e.g. type-check the new stub against the real
   SDK; confirm the corrected snippet drops into the target base).

## The gate

- **Internal leakage: zero tolerance.** Every consumer-installed skill passes the
  internal-leakage scan before merge — no `apps/reference-host` / module-source
  paths, `AOH-####`, or `mssfoobar/ops-hub` repo refs in a doc a consumer installs
  (see `references/harness.md` → *Internal-leakage scan*).
- **Skill gaps (Critical/Medium): fix on the branch before merge.** The
  point is to ship the right thing the first time, like the rest of the
  pre-merge protocol (see `agents/AGENTS.md` → Pre-merge review).
- **Base-skill or environment friction: file separately.** The test
  exercises `aoh-web-init` + the environment too; bugs there are real but
  belong to those owners — open a Linear issue, don't scope-creep this PR.
- **A "Partially" verdict caused only by environment** (e.g. no Keycloak,
  an unpublished package) does **not** block merge — attribute honestly.

## What counts (and what doesn't)

Run the **cold-execution test** for **integration / SDK-wiring skills** — the ones
with snippets a consumer copies and runtime behavior to get wrong. Skills whose
payload is *judgment* or *knowledge* (e.g. `aoh-knowledge`, `aoh-monorepo-release`,
`aoh-agent-context`, this skill) have no "compiles-but-doesn't-run" failure mode;
they take the standard pre-merge review instead. When unsure, run it.

The **internal-leakage scan** is broader: run it on **every consumer-installed
skill** — the integration skills *and* the scaffolders/others a consumer installs
(`aoh-web-init`, `aoh-dashboard`, `aoh-wfe-worker`), including those that skip the
cold run. **Exempt** (never installed into a consumer project; they document
ops-hub itself): this skill, `aoh-monorepo-release`, `aoh-agent-context`,
`aoh-scripting-conventions`, `aoh-spoke`, `aoh-hub-nexus`, `aoh-deploy`.

## Worked example — the MSR run (AOH-7567, ops-hub #213)

A naive subagent scaffolded `aoh-web-init` → integrated `aoh-msr-integration`,
skill-only. It produced a type-clean, building, lint-clean integration on
the first pass — strong evidence the skill's core was sound. It then
surfaced two gaps three doc reviewers had missed:

- **Stub mode mandated but not authorable.** The skill said "copy the
  reference-host stub" (inaccessible) and never gave the wire shapes. The
  builder's *guessed* stub used `data`/`timestamp` instead of
  `entity_state`/`event_timestamp` — it **compiled and linted clean** but
  would never populate entities (`MsrBffHandlers` types the handler
  signature, not the response body). Fixed by shipping a copyable stub +
  the wire schema, **type-checked against the real SDK (0 errors)**.
- **BFF adapter hard-coded the reference-host auth stack**
  (`@mssfoobar/auth-sdk` `accessToken` + `$lib/auth-adapter`); the web-init
  base exposes `locals.authResult.access_token` (snake_case) and has no
  auth-adapter. Fixed by making the token policy per-base.

Base/environment friction (undocumented workspace-dep build step, a
placeholder mismatch, a hard Keycloak dependency) was filed against
`aoh-web-init` separately (AOH-7568), not fixed in the MSR PR.

The harness prompt template and the evaluation rubric are in
`references/harness.md`.

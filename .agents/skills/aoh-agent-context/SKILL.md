---
name: aoh-agent-context
description: >
  Conventions for agent-context engineering — what belongs in AGENTS.md,
  CLAUDE.md, .claude/rules/, and skills, written tool-agnostic (AGENTS.md
  is the cross-tool standard; CLAUDE.md is a one-line `@AGENTS.md` import;
  `.claude/rules/` is Claude-Code-only). Covers the layered loading model,
  authoring rules per layer, length discipline, and when to extract a rule
  vs a skill vs an ADR. **Always consult before creating, editing,
  auditing, or scaffolding any of these files — the layering, import
  pattern, and `paths:` frontmatter discipline differ from generic
  defaults, and acting without checking produces drift across the
  codebase.** Trigger on "what should I put in AGENTS.md", "should
  this be a rule or a skill", "set up agent context for this repo",
  "audit my AGENTS.md", "review my .claude/rules", "scaffold agent
  context", "want this to work in cursor and copilot", "is
  `@AGENTS.md` import on purpose", "this only applies to X — where
  do I document it".
---

# AOH agent-context engineering

The core mental model is **layered loading**. Agent context is loaded into a
session at three triggering levels, and choosing the right level is the most
common authoring decision. Get this right and the agent has the context it
needs without paying the always-loaded tax for everything.

This skill is written to work across agentic coding tools — not just Claude
Code. The cross-tool standard for agent context is `AGENTS.md`. Tool-specific
files (like `CLAUDE.md` or Cursor's `.cursorrules`) are imports or shims that
delegate back to `AGENTS.md`.

## The layered loading model

| File | Loads when | Use for |
|---|---|---|
| `AGENTS.md` (with `CLAUDE.md` as a one-line `@AGENTS.md` import) | Every session, all tools that honor `AGENTS.md` | Stable repo-wide conventions an agent can't discover from the code: branching, commits, release process, behavioral defaults, where work is tracked. |
| `.claude/rules/<topic>.md` (with `paths:` frontmatter) | When files matching `paths:` are read **— Claude Code only** | Subtree-specific conventions — e.g. backend-only patterns, Dockerfile-only safety rules. Other tools won't honor this. |
| `.claude/skills/<name>/SKILL.md` (or wherever the tool installs skills) | When the skill is invoked, or when its description matches user intent | Multi-step procedures the agent runs through deliberately, not on every session. The `SKILL.md` format is becoming a cross-tool standard. |

The litmus test for whether something belongs in always-loaded context: if a
fresh contributor could find the answer by reading the code, leave it out.
Every line in `AGENTS.md` costs tokens forever, on every session, in every
conversation. Spend that budget carefully.

## CLAUDE.md ↔ AGENTS.md

`AGENTS.md` is the source of truth. `CLAUDE.md` is a one-line `@AGENTS.md`
import:

```markdown
@AGENTS.md
```

No symlink, no comment block, no divergence. The reasoning: `AGENTS.md` is the
emerging cross-tool standard (Cursor, Copilot, Aider, Continue, etc.), so
canonical content lives there. `CLAUDE.md` exists only because Claude Code
reads that filename specifically.

Different repos can have `AGENTS.md` content that legitimately diverges where
conventions actually diverge — one repo uses Changesets, another doesn't.
Don't try to make `AGENTS.md` files identical across repos; they should
reflect each repo's reality.

**Why this matters for AOH**: downstream projects scaffolded by `aoh-go-init`
and `aoh-web-init` get both files automatically. Don't break the pattern by
editing `CLAUDE.md` directly — edit `AGENTS.md` and the import keeps both in
sync.

## Editing AGENTS.md

Two tests before adding a line:

1. **Discoverability test** — would a contributor or agent need this told to them, or would they figure it out from the code? If the latter, drop it.
2. **Refactor-stability test** — if this line would still be true after a major refactor, it likely belongs here. If it would need updating with the code, it belongs in code comments, ADRs, or per-rule files instead.

Format conventions:

- **Imperatives over narrative.** "Run lint before committing" beats "We have a convention where lint should be run before committing."
- **Short why-clauses inline** when the why is non-obvious. No `**Why:**` / `**How to apply:**` headings — that level of structure is for memory entries, not `AGENTS.md`.
- **Sections stay short.** If a section grows past ~6 lines, ask whether it's actually a `.claude/rules/` topic or a skill.
- **No discoverable facts.** Don't list build commands, file paths, dependency versions, or current architecture. They go stale fast and the agent can read `package.json` / `go.mod` itself.

## Editing `.claude/rules/<topic>.md` (Claude Code only)

`.claude/rules/` is a Claude Code-specific feature; other tools won't honor
these files. Use it for path-scoped conventions where the rule is large enough
that always-loading it via `AGENTS.md` would be wasteful, and where the rule
is genuinely Claude-Code-specific or where a Claude-Code-only addition is
acceptable.

- **`paths:` frontmatter is required.** Without it, the rule loads every session and you've re-created `AGENTS.md` content with extra steps.
- **Don't restate path scope in prose** at the top of the file. The frontmatter already conveys it; the body should jump straight to the conventions.
- **One topic per file.** Split rather than nest. A 20-line `backend-handlers.md` beats a 60-line `backend.md` with three subsections.
- **Same length discipline as `AGENTS.md`.** Path-triggered loading is cheaper than always-loaded, but it's still loaded — keep it tight.
- **If the convention is also useful for Cursor / Copilot users**, pull it back into `AGENTS.md` even if it costs always-loaded budget. Cross-tool reach often beats token efficiency.

## When to add a skill

A new skill is justified when a procedure is:

1. **Multi-step** — more than three sequential actions.
2. **Repeated occasionally** — not every session, but often enough that re-typing the playbook wastes tokens.
3. **Stable enough** that a written checklist beats re-deriving it each time.
4. **Self-contained** — most of the work is *doing* the procedure, not *deciding whether* to do it.

If a procedure happens every session, it's a rule (or AGENTS.md content). If it's a one-off, just type it inline. If the *deciding* is the hard part, it belongs in an ADR or design doc — not a skill.

For procedures with side effects (releases, deploys, deletions), set `disable-model-invocation: true` in the skill frontmatter where the tool supports it. The skill becomes user-invocable only — the model can't fire it autonomously.

### Repo-level vs distribution-level skills

- **Repo-level** (`<repo>/.claude/skills/`) — procedures specific to *this* codebase: deploying this app, regenerating this API client, running this repo's release script.
- **Distribution-level** (in this `agent-skills` repo, under `skills/`) — procedures useful across multiple AOH projects (e.g. scaffolding new services, error-handling conventions, compose setup, agent-context engineering). Browse `skills/` for the current set. Promote a repo-level skill to distribution when you find yourself copying it into a third project.

### Dogfooding for skill-distribution repos

If a repo's purpose is to author and distribute skills (like `agent-skills` itself), agents working on the repo benefit from those skills being loaded for their own session — the conventions get enforced against the work that's authoring them, trigger phrases get exercised on real queries, and self-inconsistencies the skills are designed to catch surface during development rather than after release.

The pattern: a single relative symlink, `.claude/skills` → `../skills`, committed to git. Every clone reproduces the symlink automatically; edits to `skills/<name>/SKILL.md` are immediately visible at `.claude/skills/<name>/SKILL.md` because both paths resolve to the same file. No setup script, no copy step, no drift.

This is the same import shape used at the file layer for `CLAUDE.md` ↔ `AGENTS.md`: canonical content lives at the tool-agnostic source location, and a Claude-Code-specific shim points at it. Same trade-off too — Windows clones without `core.symlinks=true` (or developer mode) see the symlink as a text file, but the canonical content and downstream consumer install are unaffected; only the in-session dogfooding is degraded for that contributor.

Document the symlink and its Windows caveat in the repo's `AGENTS.md` so a confused contributor finds the explanation in the always-loaded layer rather than having to consult this skill.

## Skill description quality

The `description` field is the entire trigger surface. A skill with great
content and a weak description never fires. Good descriptions are:

- **Specific about triggers.** List user phrasings and contexts that should match. Don't only say what the skill does — say *when to use it*.
- **Mildly pushy.** Models tend to under-trigger skills. Lean toward more trigger phrasings rather than fewer, and include indirect ones ("set up X for this repo", "configure Y", "what should I put in Z").
- **Honest about scope.** Don't promise things the body doesn't deliver — that produces ghost triggers where the skill fires but the content disappoints, training the model to under-trust it.

Look at the descriptions in the `agents/skills/` directory for working
examples — they're tuned to fire reliably on small specific queries while
staying silent on near-miss adjacent ones.

## What does NOT belong in agent context

Even with the layered model, some things still don't belong anywhere in the
layout:

- **Discoverable facts** — build commands, file paths, dependency versions, current architecture. These live in code, README, lockfiles, and ADRs.
- **Transient state** — in-flight work, decisions still being debated, "we're currently migrating from X to Y". This belongs in the issue tracker or ADRs, not `AGENTS.md`.
- **Per-user preferences** — keyboard shortcuts, editor configs, personal aliases. These belong in user dotfiles.
- **Long prose explanations.** If a topic needs more than ~10 lines, link to a doc rather than inlining it. Always-loaded context is expensive.

The strongest signal that agent context has drifted off-track: an `AGENTS.md`
that needs updating every time the code changes. If that's happening, content
is in the wrong layer — pull it down into `.claude/rules/`, into code
comments, or out of agent context entirely.

## AOH-specific patterns to follow

- **Scaffolded projects already have `AGENTS.md` + `CLAUDE.md` import** — the `aia init` turborepo template ships a repo-root pair, and `aoh-go-init`, `aoh-web-init`, and `aoh-module-init` set up per-app/-module pairs automatically. Don't recreate them; extend the existing `AGENTS.md`.
- **Cross-skill references use the skill name, not a path** — write "see the `aoh-knowledge` skill's `references/keycloak-realm-guide.md`" rather than `.claude/skills/aoh-knowledge/references/...`. Skill install paths vary across tools and install modes (user vs project).
- **Tool-specific files are imports** — if you find yourself writing a long `CLAUDE.md` directly, stop. Move the content to `AGENTS.md` and shrink `CLAUDE.md` to the one-line import.

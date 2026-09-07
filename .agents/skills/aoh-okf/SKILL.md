---
name: aoh-okf
description: >
  Author, maintain, and validate knowledge bundles in Google's Open Knowledge
  Format (OKF v0.1) — markdown concept docs with YAML frontmatter, reserved
  index.md/log.md files, and cross-links, consumable by humans and agents.
  Bundles a dependency-free validator (frontmatter conformance + link
  resolution). Use when creating or editing files in a knowledge/ or OKF
  bundle directory, adding a runbook/service/concept page, scaffolding a new
  knowledge bundle for a repo, validating a bundle before commit, or when the
  user mentions OKF, Open Knowledge Format, knowledge bundle, runbook docs,
  concept docs, or index/log upkeep.
---

# OKF knowledge bundles

An OKF bundle is a directory of markdown files: **concept docs** (one fact
domain per file, YAML frontmatter) plus reserved **`index.md`** (directory
listing) and **`log.md`** (change history). Spec:
[OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md).

## Quick start — add a concept doc

```markdown
---
type: Runbook            # required, non-empty. House vocabulary below.
title: Deploy the gateway
description: One sentence for previews and search snippets.
tags: [deploy, gateway]
timestamp: 2026-07-17    # date of last meaningful change
resource: http://host:4000   # optional URI — only for docs about a concrete asset
---

# Deploy the gateway
...body: steps, tables, links...
```

Then complete the **authoring checklist**:

1. Add a line for the doc in its section's `index.md`:
   `* [Title](file.md) - description`
2. For structural changes (new/removed docs), add a dated entry to the
   bundle's `log.md` under a `## YYYY-MM-DD` heading (newest first).
3. Run the validator (below) and fix every error before committing.

**When removing a doc:** delete its line from the section `index.md`, log the
removal in `log.md`, and fix inbound links — the validator flags any you miss
as broken links.

## House rules (stricter than the spec)

- **Relative links only** (`../services/foo.md`, `file.md`, `dir/`) — the
  spec also allows bundle-absolute `/path.md`, but GitHub renders those
  against the repo root, so they break for human readers. The validator
  errors on them by default.
- **No broken links.** The spec tolerates them; we don't — a broken link in
  a runbook is a failed procedure step.
- **`type` vocabulary** — reuse before inventing: `Runbook` (step-by-step
  procedure), `Service` (one running component), `Concept` (how/why a system
  is shaped), `Policy` (rules to follow), `Reference` (lookup tables,
  decisions), `Machine` (a host/box).
- **Don't duplicate deep sources.** Bundle docs link to the authoritative
  files (measured figures, changelogs, compose files) rather than restating
  them — restated numbers rot.
- Reserved files carry **no frontmatter**; exception: the bundle's root
  `index.md` may declare `okf_version: "0.1"` and nothing else.

## Validate

```bash
python3 scripts/validate_okf.py <bundle-dir>          # from this skill's dir
python3 scripts/validate_okf.py knowledge/ --lax      # absolute links warn, not error
```

Exit 0 = conformant (warnings allowed); exit 1 = errors; exit 2 = bad path.
Errors: missing/unclosed frontmatter, empty `type`, frontmatter on reserved
files, broken links, bundle-absolute links, unreadable files/directories,
symlinked directories (not followed — contents would go unvalidated), and
zero-markdown-file bundles. Warnings: missing recommended fields
(title/description/tags/timestamp), malformed `log.md` date headings,
and (under `--lax`) bundle-absolute links. **Broken links fail even under
`--lax`** — that flag relaxes only the relative-links house rule, not the
no-broken-links one. Links may point outside the bundle (e.g. to repo files);
they are existence-checked against the working tree. Links inside code
blocks/spans are treated as examples and skipped. Run the validator on every
bundle change; it is the deterministic gate.

## Scaffold a new bundle

```
knowledge/
├── index.md        # frontmatter: okf_version "0.1" only; lists sections
├── log.md          # "# Directory Update Log" + dated entries
└── <section>/      # e.g. runbooks/ concepts/ services/
    ├── index.md    # section listing, no frontmatter
    └── <doc>.md    # concept docs
```

Start from the reader's tasks (deploy X, onboard Y, debug Z) and write
runbooks executable top-to-bottom by someone new; put shape/why context in
concept docs, one page per running component in service docs. Wire the
repo's AGENTS.md to require bundle updates with every change, or the bundle
rots.

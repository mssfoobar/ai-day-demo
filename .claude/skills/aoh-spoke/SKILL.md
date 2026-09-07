---
name: aoh-spoke
description: >
  Judgment-call helper for project CMs and platform devs working with an AOH
  spoke. The spoke itself is stood up deterministically by the aa-cli (`aia
  spoke init`, `aia spoke up`) consuming compose fragments from
  deploy/spoke-services/. This skill owns the things a CLI can't do without
  interpretation: troubleshooting a running spoke (why isn't my runner
  picking up jobs, why is Nexus 401ing, why is Postgres rejecting
  connections), proposing custom compose modifications (add a project-
  specific volume, wire an external IAMS), and three-way-merging fragment
  upgrades against a CM's local customizations. Trigger keywords: "spoke",
  "aoh spoke", "troubleshoot spoke", "spoke not starting", "runner not
  registered", "forgejo not healthy", "nexus 401", "harbor push fails",
  "backstage can't connect", "aia spoke", "spoke upgrade", "customize
  spoke", "spoke add service".
license: Proprietary
metadata:
  owner: AOH Platform Team
  status: v2-composition-framework
---

# AOH Spoke

Runtime helper for an AOH project spoke. The spoke is the per-project
workspace hosting Forgejo (SCM + CI + internal packages), Nexus (external
registry proxy + internal publishing), Harbor (container + Helm OCI),
Backstage-spoke (developer portal), shared Postgres, and Traefik ingress.

**This skill does not bring the spoke up.** That is aa-cli's job via
`aia spoke` — a deterministic CLI that reads compose fragments from the
ops-hub repo's `deploy/spoke-services/` tree at a pinned tag and assembles
them into a single compose project. See the [spoke-services README](../../../deploy/spoke-services/README.md)
for the fragment layout, and the aa-cli docs for the `aia spoke`
subcommand family.

What this skill *does* own is the stuff that needs judgment:

| Need | What this skill does |
|---|---|
| Something's broken; I can't tell what | Walk the CM through diagnosing which service + why, using `references/TROUBLESHOOTING.md` as a starting catalog |
| I need to customize my spoke (add a volume, swap an image, wire external auth) | Propose the minimal compose-fragment edits, explain the trade-off |
| A new ops-hub release tag dropped; I've made local edits to my spoke | Three-way merge: upstream fragment, user's current state, proposed resolution |
| Do I need Nexus? Do I need Harbor? | Help a CM reason about their service-selection answers before they run `aia spoke init` |

## How the pieces fit

```
┌─────────────────────────────────────────────────────────────────┐
│ ops-hub repo (this repo, pinned-tag consumption)               │
│                                                                 │
│  deploy/spoke-services/   ← compose fragments (deterministic)   │
│    postgres/                                                    │
│    forgejo/                                                     │
│    act-runner/                                                  │
│    nexus/                                                       │
│    harbor/                (currently a stub; AOH-6761)          │
│    backstage-spoke/       (demonstration image; AOH-6764)       │
│    traefik/               (team profile only)                   │
│                                                                 │
│  agents/skills/aoh-spoke/ ← THIS SKILL (agentic)               │
│    SKILL.md               mental model + CM onboarding doc      │
│    references/            troubleshooting catalogs              │
└─────────────────────────────────────────────────────────────────┘
                            │
                            │ aa-cli fetches via
                            │ https://github.com/<org>/ops-hub/
                            │   archive/refs/tags/<tag>.tar.gz
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ aa-cli (~/Workspace/.../aa-cli, separate repo + PR)            │
│                                                                 │
│  aia spoke init   → interactive Q&A + service picker            │
│  aia spoke up     → compose fragments + run bootstraps          │
│  aia spoke down   → stop; --volumes to wipe                     │
│  aia spoke status | logs                                        │
│  aia spoke upgrade → three-way merge (delegates to this skill)  │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ ~/.aoh/spokes/<spoke-name>/  (CM's local state)                │
│   services/                   copied fragments; CM can edit     │
│   .env                        CM's selections + secrets         │
│   .env.shared                 cross-service URLs                │
│   docker-compose.yml          generated with include: fragments │
└─────────────────────────────────────────────────────────────────┘
```

## When to use this skill

Trigger it (or ask the agent directly) when:

- `aia spoke up` reports a failure and the output doesn't make obvious sense
- A service is "healthy" but behaving unexpectedly (runner picks up jobs but they hang; Nexus resolves some packages but not others; Backstage loads but plugins 500)
- A CM wants to customize their spoke past what `aia spoke init` asks — e.g. add a persistent volume for their own tool, swap Backstage for a custom yarn-built image, wire IAMS into Forgejo
- `aia spoke upgrade` detects fragment-vs-local-customization conflicts — this skill drives the three-way merge
- A CM is trying to decide what services their project actually needs (Nexus vs Forgejo built-in for internal packaging; Harbor now or later; Backstage yes/no)

Don't trigger it for:
- Initial bring-up — run `aia spoke init` and `aia spoke up` first
- Adding a *new* service to the platform catalog — that's a new fragment in `deploy/spoke-services/`, which is a platform-team task, not a CM task
- Scaling, HA, multi-node — spoke is single-server by design (compose, not K8s); that's a `aoh-spoke-k8s` discussion that doesn't exist yet

## Profile differences

| | Laptop profile | Team profile |
|---|---|---|
| Audience | Platform dev dress-rehearsing the spoke on their own machine | Project CM running the spoke on a project server |
| Ingress | Each service binds 127.0.0.1:<own-port> | Traefik fronts everything on 443 + dedicated TCP entrypoints |
| TLS | None (plaintext HTTP) | Traefik-terminated (ACME or sealed cert) |
| DNS | `localhost` | Internal zone (`*.<spoke-domain>`) |
| Auth | Local admin accounts per-service | IAMS OIDC federation (AOH-6854 follow-up) |
| Storage | Single named volumes per service | Same, plus team-scale considerations (backups — AOH-6854) |
| Database | Shared Postgres (same as team — consistency) | Shared Postgres with per-service roles (AOH-6854) |
| Secrets | `.env` file | OpenBao-backed once AOH-6742 lands |

## References

- `references/TROUBLESHOOTING.md` — diagnosis flowcharts for the common failures
- `../../../deploy/spoke-services/README.md` — fragment layout + manifest schema
- aa-cli `aia spoke` docs (separate repo) — deterministic CLI surface

## Scope this skill explicitly does NOT cover

- Bringing up the spoke from scratch (→ `aia spoke up`)
- Editing platform-wide fragments (→ PR against `deploy/spoke-services/` in this repo)
- K8s / Helm deployment (hub territory; this is compose-only)
- Air-gap bundle seeding (→ AOH-6753, separate workflow)

## Context

<!-- Background, current state, constraints, stakeholders. -->

## Goals / Non-Goals

**Goals:**
<!-- What this design aims to achieve -->

**Non-Goals:**
<!-- What is explicitly out of scope -->

## Runtime dependencies

| Dependency | Role | Owned by |
|---|---|---|
| `iams-keycloak` | … | Platform — added by `aoh-compose` |
| `iams-aas`      | … | Platform — added by `aoh-compose` |
| … | … | … |

## API surface

| Method | Path | Auth posture | Description | Fronting gateway route |
|---|---|---|---|---|
| POST | /v1/… | BearerAuth, role `…` | … | /aoh/gateway/<module>/v1/… |
| GET  | /livez   | Unauthenticated | … | n/a |
| GET  | /readyz  | Unauthenticated | … | n/a |

## UI / Design System

<!-- Delete this whole section if the change is purely backend. -->

## Decisions

<!-- Key decisions with rationale and alternatives considered. -->

## Risks / Trade-offs

<!-- [Risk] → Mitigation -->

## Migration Plan

<!-- Deploy steps + rollback. Skip if N/A. -->

## Open Questions

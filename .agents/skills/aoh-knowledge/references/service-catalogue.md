# AOH Service Catalogue

Quick reference for the AOH services — what they do, what they depend on, and which
ones to pick for a given use case.

> **Every AOH service is deployed by the consumer — AOH has no operated
> service plane.** IAMS, AMM, RTUS, and SDS have not yet migrated into the
> ops-hub monorepo: deploy their existing pre-v3 releases (iams-aas 1.3.7 /
> iams-keycloak 1.5.1, amm-app 1.0.0, rtus-pms 1.2.0 / rtus-seh 1.2.2,
> sds-server 1.0.0) until their migration lands.

## Dependency Graph

```
CORE (always included):
  infra (Traefik) <-- required by ALL services
    └── iams (Keycloak + AAS) <-- required by MOST services
          └── sds (Session Data Store + Valkey) <-- included with iams

SUPPORTING (pick what you need):
  rtus (Real-time Update Service)        <-- depends on: iams
  gis  (Geospatial Information System)   <-- depends on: iams, rtus
  unh  (Unified Notification Hub)        <-- depends on: iams
  ian  (In-App Notification)             <-- depends on: iams, unh, rtus
  dash (Dashboard Service)               <-- depends on: iams
  ptmgr (Push Token Manager)             <-- depends on: iams
  amm  (Attachment Management Module)    <-- depends on: iams [+ optional ICAP]
  form (Form Builder & Submission Mgmt)  <-- depends on: iams, amm

AUXILIARY (standalone, extend C2 capabilities):
  wfe  (Workflow Engine)                 <-- depends on: iams [+ form for human-in-the-loop Form-task steps]
```

## Service Summary

| Service | Full Name | Type | Purpose | Has Web UI? |
|---------|-----------|------|---------|-------------|
| **infra** | Infrastructure | Core | Traefik reverse proxy, CORS | Traefik dashboard |
| **iams** | Identity & Access Mgmt | Core | Keycloak auth, AAS authorization | Yes (user mgmt) |
| **sds** | Session Data Store | Core | Server-side session storage | No |
| **rtus** | Real-time Update Service | Supporting | Pub/sub + SSE to browsers | No |
| **gis** | Geospatial Info System | Supporting | Maps, overlays, entity tracking | Yes |
| **unh** | Unified Notification Hub | Supporting | Email, push, SMS, custom channels | Yes (templates) |
| **ian** | In-App Notification | Supporting | In-app alerts + real-time delivery | Yes |
| **amm** | Attachment Management | Supporting | File upload/download (local volume default, MinIO opt-in) | MinIO console (opt-in) |
| **form** | Form Builder & Submission Mgmt | Supporting | Drag-and-drop forms, versioning, state machine submissions | No — consumer apps embed the SurveyJS authoring Creator + read/submit renderer via `@mssfoobar/form-web-sdk` |
| **dash** | Dashboard Service | Supporting | Configurable widget dashboards (owns its own tags) | No — consumer apps embed via `@mssfoobar/dash-web-sdk` |
| **ptmgr** | Push Token Manager | Supporting | FCM token storage | No |
| **wfe** | Workflow Engine | Auxiliary | BPMN 2.0 workflow orchestration | No — consumer apps embed the designer + monitor via `@mssfoobar/wfe-web-sdk` |

## Recommendation Bundles

Use these bundles as starting points based on what you're building:

### Minimal (auth only)
**Services:** infra -> iams (includes sds)
**Use when:** Your service just needs authentication and authorization.
**You get:** Keycloak OIDC, JWT validation, role-based access, multi-tenant support.

### Standard (auth + notifications)
**Services:** infra -> iams -> unh
**Use when:** Your service needs to send notifications (email, push, SMS).
**You get:** Everything in Minimal + multi-channel notification delivery with templates.

### Real-time (auth + live updates)
**Services:** infra -> iams -> rtus
**Use when:** Your frontend needs live data pushed from the server.
**You get:** Everything in Minimal + Hazelcast pub/sub, SSE delivery to browsers.

### Geospatial (auth + maps + real-time)
**Services:** infra -> iams -> rtus -> gis
**Use when:** Your service works with geographic data, map overlays, or entity positions.
**You get:** Everything in Real-time + geospatial CRUD, map rendering, entity tracking.

### Full alerts (maps + notifications + in-app)
**Services:** infra -> iams -> rtus -> gis -> unh -> ian
**Use when:** Operational/C2 apps needing maps, multi-channel alerts, and in-app messages.
**You get:** Everything in Geospatial + notification hub + real-time in-app alerts.

### Full operational
**Services:** All of the above + dash + amm
**Use when:** You need the complete platform — dashboards and file management.

### Add-ons (combine with any bundle)
- **+ amm** — File upload/download/storage (local volume by default; MinIO + virus scanning opt-in)
- **+ dash** — Configurable widget dashboards (owns its own tags)
- **+ ptmgr** — FCM push token management (for mobile push notifications)

### Auxiliary (standalone stacks, extend C2 capabilities)
- **+ wfe** — BPMN 2.0 workflow orchestration; human-in-the-loop forms, custom activities

## Use Case Decision Tree

**"I need authentication"** -> IAMS (always the starting point)

**"I need to push data to the browser in real-time"** -> RTUS
- Topics: one-to-many broadcast
- Maps: key-value store with change notifications

**"I need maps or location data"** -> GIS (pulls in RTUS)

**"I need to send emails / push notifications / SMS"** -> UNH
- Supports templates, distribution lists, multiple channels

**"I need in-app notification badges and toasts"** -> IAN (pulls in UNH + RTUS)

**"I need file upload/download"** -> AMM
- Local-volume storage by default (MinIO/S3 opt-in), PDF previews via Gotenberg
- Optional virus scanning via ClamAV + C-ICAP

**"I need to build forms and manage submissions"** -> Form (pulls in AMM)
- Drag-and-drop designer via SurveyJS Creator (40+ question types)
- Versioned, immutable published forms; role-based submission state machine
- Requires a SurveyJS licence key (`SURVEYJS_KEY`) for the designer UI

**"I need a dashboard"** (situation picture, monitoring page, KPI overview, ops board — fixed OR user-customisable) -> DASH
- In AOH, dashboards are a DASH surface — do **not** hand-roll a bespoke `@mssfoobar/ui` page. Fixed dashboards are authored as widgets (`defineWidget`) and seeded via the dash REST API; the data still comes from your domain service. See `service-selection-criteria.md` → DASH and the `aoh-dashboard` skill.

**"I need mobile push notifications"** -> PTMGR + UNH
- PTMGR stores FCM tokens, UNH delivers notifications

**"I need to automate multi-step business processes"** -> WFE
- Embeddable BPMN-style designer + monitor (`@mssfoobar/wfe-web-sdk`), durable execution via Temporal
- Supports human-in-the-loop via form tasks (delegated to the Form module), custom Go activities

## Keycloak Client Types

Each AOH service needs a Keycloak client in the realm:

**Public clients** (web frontends that redirect users to login):
`iams, gis, ian, rtus, dash, unh-web, form-web, form-submitter-web`
- Browser-based OIDC flow, no client secret
- WFE ships no web frontend — a host app embedding `@mssfoobar/wfe-web-sdk`
  registers its own public client (as dash-web-sdk consumers do)

**Confidential clients** (backend service accounts for server-to-server calls):
`sds-client, unh-app, ptmgr`
- Client secret, service account with realm-management roles

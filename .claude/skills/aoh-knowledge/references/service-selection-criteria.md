# Picking an AOH module: decision-making criteria

When a developer's use case could plausibly be served by an AOH module (Form, DASH, RTUS, GIS, WFE, AMM, UNH, IAN, …), **think before recommending the service**. AOH modules are powerful but each one adds a service container, often a database, a Traefik route, environment vars, a network hop on the request path, and a coupling between your application's domain data and an external service's schema.

**Default to in-app primitives** (Svelte forms, plain HTTP endpoints in your existing domain service, your own Postgres tables) unless one of the module's specific, named capabilities is *required* by the use case.

This file lays out the criteria. Apply it before answering "which AOH module should I use for X?" — sometimes the answer is "none; build it in your existing service."

## The core principle

> An AOH module earns its cost when the use case requires a capability the module *uniquely* provides.

If everything the use case needs can be built with framework-level primitives — Svelte + Zod for form UI, your existing Go service for persistence, Postgres for storage, RTUS for live updates — build it that way.

The common failure mode is **feature aggregation as justification**: listing six things an AOH module provides ("validation, RBAC, audit, autosave, versioning, state machine") as though all six justify the dependency. Only the items *unique* to the module count. Validation comes from Zod. RBAC comes from any AAS-protected endpoint. Audit comes from any `aohlog`-wired service. The only thing Form uniquely provides is **dynamic-schema authoring + versioned read-compat** — everything else on its feature list is a wash.

## Five decision questions

Walk these in order. If any answer is "no / unclear / maybe later," the module is probably wrong.

1. **What is the module's *unique* capability for this use case?** Name it precisely. ("Form persists schemas with versioning and a backward-compat read model" — specific. "Form does validation" — not unique to Form.)
2. **Does the use case require that capability today?** Not "might use it later" — actual present requirement. Future-proofing via AOH modules is usually premature commitment.
3. **What's the runtime cost?** Container, DB, compose entry, Traefik route, env vars, a network hop per request. Worth it for the one capability identified in (1)?
4. **What's the data-ownership cost?** Schema-in-code (typed end-to-end through TypeScript + Go structs + OpenAPI) vs schema-in-service (configured via another team's admin UI). The latter breaks codebase-as-source-of-truth and complicates downstream queries.
5. **Is there an existing service in your codebase that already owns this domain?** If the use case is "field reporter submits an incident" and `incident-svc` already exposes `POST /v1/incidents`, the choice isn't "Form or a new bespoke service" — it's "Svelte form posting to the existing endpoint."

## Service-by-service quick checks

Rules of thumb for when each service *is* the right call vs when its features are over-applied. Always combine with the five questions above.

### Form

**Use Form when:**
- A non-developer persona (admin, ops manager) **configures forms via UI** without a code change.
- Schemas vary **per tenant** or per deployment.
- Submitted records must remain readable by the schema version they were submitted against (regulatory / legal forms).
- Schema churn is driven by **business users**, not by the dev team's roadmap.

**Skip Form when:**
- The form has a **fixed schema** authored by developers.
- Validation rules depend on application state (e.g. "this field is required only when status is X").
- You want **typed end-to-end** contracts (TS ↔ Go ↔ DB).
- The submission is **tightly coupled** to a domain object that lives in your own service — your service should own both the schema and the persistence path.

**Default:** Svelte form + Zod (or Valibot) + your domain service's `POST /v1/<resource>` endpoint.

### DASH

**In AOH, a dashboard is a DASH surface — this is a named exception to this file's "default to in-app primitives" bias.** When the requirement is a dashboard, default to DASH via `@mssfoobar/dash-web-sdk`, *including* fixed, single-layout, real-time-KPI dashboards. Hand-rolling a bespoke `@mssfoobar/ui` dashboard page is the wrong default: it drifts the product away from the platform's dashboarding model (consistent widget catalogue, seeding, and later customisation / central admin tooling without a rewrite).

**Use DASH when** the surface is a dashboard at all — a situation picture, monitoring page, KPI overview, ops board — whether it is:
- user-customisable (add/remove widgets, drag-to-arrange, save layouts, multiple configs per user, favourites, sharing), OR
- a **fixed** layout the dev team owns and **seeds** once via the dash REST API (the common case for a curated operational overview). "It's a fixed layout" / "the dev team owns the widgets" is **not** a reason to skip DASH — those are ordinary DASH usages: author the widgets with `defineWidget`, seed the layout.

**Skip DASH only when** the surface isn't really a dashboard — e.g. a single stat tile embedded in an otherwise-non-dashboard screen, or one-off inline metrics on a details / list page. If the user (or the spec) calls it a "dashboard," that exclusion does not apply.

**Default:** DASH via `@mssfoobar/dash-web-sdk` — author widgets with `defineWidget()`, register them in the consumer app, and for a fixed dashboard **seed it** through the dash REST API (`POST /dashboard` with inline widgets). See the **`aoh-dashboard`** skill for pages, widgets, and the seeding workflow.

> **DASH stores config, not your data — so "use DASH" and "build a domain endpoint" are both true.** Two independent things: (1) *use DASH* for the dashboard surface, and (2) *still build the domain read/aggregation endpoint* that feeds the widgets on your existing service (DASH does not aggregate your data). The widget *implementations* live in your consumer app (via `defineWidget()`); only the dashboard *config* lives in dash-service.

### RTUS

**Use RTUS when:**
- The UI must reflect changes **within ~2 seconds** of an event in another session.
- The data changes frequently and polling would be wasteful or slow.
- Multiple browser sessions need the same live view (e.g. ops dashboard).

**Skip RTUS when:**
- Data changes infrequently (every few minutes — polling is fine).
- The user explicitly triggers a refresh.
- Eventual consistency on next navigation is acceptable.

### GIS

**Use GIS when:**
- A **map display** is a product requirement.
- Geospatial queries (within-radius, polygon containment) drive functionality.
- Layer / overlay management with persistence is needed.

**Skip GIS when:**
- Coordinates are stored but never visualised.
- A static address string is sufficient.

### WFE

**Use WFE when:**
- A process has **long-running steps** (hours, days), **branching logic**, or **human-in-loop** approvals.
- Compensation / retry behaviour must survive service restarts.

**Skip WFE when:**
- The "workflow" is a synchronous request/response, completed in one HTTP call.
- The state machine is simple and small enough to live in your service's domain code (`reported → acknowledged → closed` is not a WFE-worthy workflow).

### AMM

**Use AMM when:**
- Users upload files (photos, documents, evidence).
- Antivirus scanning, signed-URL access, or S3/MinIO-backed storage is required (opt-in; AMM defaults to a local volume).

**Skip AMM when:**
- No file upload; just text and structured data.

### UNH / IAN

**Use UNH when:**
- Notifications cross channels (email, SMS, push).
- Templates and delivery preferences are configured centrally.

**Use IAN when:**
- The notification is **in-app** (toast or persistent inbox).
- The user must see it on next login if they were offline.

**Skip both when:**
- An ephemeral toast in the current session is enough — `@mssfoobar/ui`'s `<Toast>` covers it.

### SDS

**Use SDS when:**
- Tokens must stay server-side (browser holds only a session ID cookie).
- Multiple frontends share the same session.

This is the platform default for any AOH web app that uses the OIDC flow from `aoh-web-init`. Skipping SDS means either keeping tokens in browser cookies (NOT recommended) or your app is stateless and doesn't need session continuity.

## Anti-patterns

**False dichotomy: "AOH module vs new bespoke service."**
Almost always wrong. The third option is "extend an existing service in your own codebase." If you're proposing a 4-field reporting flow and you already have an `incident-svc` with `POST /v1/incidents`, that's where the form posts. Don't invent a fork.

**Feature aggregation as justification.**
Six features an AOH module "provides for free" doesn't justify the dependency unless those features are *unique to that service*. Audit your justification list against the platform's actual unique capability set — most items on the list are framework-level features you'd get from any AAS-protected, `aohlog`-wired service.

**Future-proofing via AOH modules.**
"We might want to make this admin-configurable in 6 months" is not a reason to commit to an AOH module today. Build for current requirements; migrate when the requirement actually materialises. The cost of migrating later is almost always lower than the cost of carrying the dependency for 6 months without using its unique capability.

**Confusing schema versioning with enum value additions.**
Adding a new severity value (`emergency`) to a Postgres enum is a one-line migration. Form's schema versioning is for when the form's **field set or structure** changes (add a field, remove a field, change a field's type) and old submissions must remain readable by their original schema. Different problem.

**Treating "the platform ships this, so we should use it" as a rule.**
The platform ships services; using them is a choice. AOH's modular framework lets each application pick the services it needs, not all of them.

## How to deliver the recommendation

When a developer asks "should I use service X for this?":

1. State the module's *unique* capability for the use case (one sentence).
2. Test it against the five decision questions.
3. If the answer is "skip the module," name the simpler alternative concretely: "Svelte form + Zod posting to `<existing-service>`'s `POST /v1/<resource>`." Don't leave the developer to guess.
4. If the answer is "use it," reference the relevant per-service skill or pattern doc (e.g. `aoh-dashboard` for DASH consumer patterns) so they don't reinvent.

# Authoring the MSR stub (backend-free dev)

The MSR skill leans on **stub mode** for backend-free development: when
`MSR_URL` is unset, your BFF adapter returns an in-memory handler set
instead of proxying to `msr-service` (see `bff.md`). This file gives you
what you need to write that stub: the **exact wire shapes** the SDK worker
parses, and a **complete, copyable** stub you can drop in and adapt.

> **Why this matters — the silent-stub trap.** `MsrBffHandlers` only types
> the handler *signatures* (`event → Promise<Response>`); it does **not**
> type the JSON *body* you put in the `Response`. So a stub with the wrong
> field names (`data` instead of `entity_state`, `timestamp` instead of
> `event_timestamp`, …) **compiles clean and lints clean** but the worker
> can't parse it — playback "runs" yet `replayStore.entities` stays empty
> and nothing renders. The field names below are load-bearing; match them
> exactly.

## Wire shapes the worker/store parse

All shapes are **snake_case** (they mirror `msr-service`'s Go json tags,
re-declared in `@mssfoobar/msr-types`).

**Success envelope** (every 2xx body except the 204s):

```jsonc
{ "data": <payload>, "message": "…optional…", "sent_at": "<RFC3339>" }
```

**Pagination envelope** (`admin/sessions` only) adds:

```jsonc
{ "data": [...], "sent_at": "…", "page": { "number": 1, "size": 50, "total_records": 3, "count": 3, "sort": null } }
```

**Error envelope** (non-2xx) — note the distinct field set (`errorCode` /
`errorMessage`), unlike the success envelope; `trace_id` (the OTEL trace id)
is omitted when no trace is active:

```jsonc
{ "timestamp": "<RFC3339>", "trace_id": "<hex>", "errorCode": "INVALID_REQUEST", "errorMessage": "…", "details": { } }
```

**Entity/event payloads** — the heart of replay:

| Endpoint | `data` item shape | Notes |
|---|---|---|
| `GET /replay/state` | `{ entity_id, entity_state, op: "r", table_name }` | The snapshot at a timestamp. `op` is `"r"` (read). **No `event_timestamp`.** |
| `GET /replay/events` | `{ entity_id, entity_state, op: "u", table_name, event_timestamp }` | Changes in the window. `op` is usually `"u"`; `"c"`/`"u"` upsert, `"d"` deletes. **Has `event_timestamp`** (RFC3339). |

`entity_state` is the row's columns as a JSON object — it becomes
`entity.state` in `replayStore.entities`, and `entity_id` / `table_name`
become `entity.id` / `entity.tableName`. The store keys entities by
`` `${table_name}:${entity_id}` ``.

**Session** (`session.create`, `admin.sessions`):

```ts
{ id, user_id, status: "ACTIVE" | "INACTIVE", last_seen_at, tenant_id,
  occ_lock, created_at, update_at /* sic — not updated_at */, created_by, updated_by }
```

**AdminSettings** (`admin/settings` GET; the replay range is derived from
this):

```ts
{ max_active_sessions, data_retention_cron_expression,
  max_playback_range /* DAYS */, earliest_valid_timestamp? }
```

PUT `/admin/settings` echoes back **only the submitted keys** (a partial),
never the full document. Only those four keys are mutable.

**Status codes the worker keys on:**

- `400 INVALID_REQUEST` — missing `timestamp` / `start` / `end`.
- `400 BAD_REQUEST` — `replay/state` with no active session.
- `403 SESSION_TERMINATED` on the `replay/events` poll — the exact signal
  the worker uses to drive the store to `terminated`. Your stub must emit
  this once the session is gone, or "terminate" won't propagate.

## The copyable stub

Drop this at `src/lib/server/msr-stub.ts` and wire it in your adapter
(`if (!env.MSR_URL) return msrStubHandlers;` — see `bff.md`). It is
single-identity and module-scoped (state shared across requests). **To
make it your own, change `STUB_TABLE` + `entityStateAt()`** — everything
else is the protocol.

```ts
// src/lib/server/msr-stub.ts — backend-free MSR BFF (used when MSR_URL is unset).
// Same shape as MsrBffHandlers, so your route files switch with one ternary.
import type { MsrBffEvent, MsrBffHandlers } from "@mssfoobar/msr-web-sdk/server";
import { trace } from "@opentelemetry/api";

// ---- the single identity this stub runs as ----
const USER_ID = "dev-user";
const TENANT_ID = "dev-tenant";

// ---- deterministic demo entities: position is a PURE function of time, so
//      replay/state at T and replay/events over any window are reproducible.
//      SWAP THIS (and STUB_TABLE) for your own domain rows. ----
const STUB_TABLE = "demo.vehicle";
const STEP_MS = 2000; // one update per entity per 2s
const ENTITY_IDS = ["v-1", "v-2", "v-3"];
const BASES = [
  { name: "Alpha", lon: 103.7, lat: 1.28, vlon: 0.002, vlat: 0.0016 },
  { name: "Bravo", lon: 104.05, lat: 1.46, vlon: -0.0018, vlat: -0.0011 },
  { name: "Charlie", lon: 103.86, lat: 1.22, vlon: 0.0004, vlat: 0.0022 },
];
function entityStateAt(i: number, epochMs: number) {
  const step = ((Math.floor(epochMs / STEP_MS) % 150) + 150) % 150; // wrap, stay positive
  const b = BASES[i];
  return { name: b.name, lon: b.lon + step * b.vlon, lat: b.lat + step * b.vlat };
}

// ---- envelopes ----
const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
const ok = <T>(data: T, message?: string) => ({
  data,
  ...(message ? { message } : {}),
  sent_at: new Date().toISOString(),
});
const fail = (errorCode: string, errorMessage: string, status: number) => {
  // The OTEL trace id is the single correlation key; omit it when no trace
  // is active (the HTTP instrumentation normally sets one at the edge).
  const traceId = trace.getActiveSpan()?.spanContext().traceId;
  return json(
    {
      timestamp: new Date().toISOString(),
      ...(traceId ? { trace_id: traceId } : {}),
      errorCode,
      errorMessage,
    },
    status,
  );
};

// ---- module-scoped session + settings state ----
interface StubSession {
  id: string;
  user_id: string;
  status: "ACTIVE" | "INACTIVE";
  last_seen_at: string;
  tenant_id: string;
  occ_lock: number;
  created_at: string;
  update_at: string;
  created_by: string;
  updated_by: string;
}
const sessions = new Map<string, StubSession>();
const activeCount = () => [...sessions.values()].filter((s) => s.status === "ACTIVE").length;

const settings = {
  max_active_sessions: 5,
  data_retention_cron_expression: "0 0 * * *",
  max_playback_range: 24, // DAYS
  earliest_valid_timestamp: null as string | null,
};
const earliestValid = () => settings.earliest_valid_timestamp ?? new Date(Date.now() - 2 * 3_600_000).toISOString();

export const msrStubHandlers: MsrBffHandlers = {
  session: {
    create: {
      POST: async () => {
        const now = new Date().toISOString();
        const existing = [...sessions.values()].find((s) => s.status === "ACTIVE" && s.user_id === USER_ID);
        if (existing) {
          existing.last_seen_at = now; // one ACTIVE session per user (upsert), like the service
          return json(ok(existing, "Session created successfully"), 201);
        }
        const s: StubSession = {
          id: crypto.randomUUID(),
          user_id: USER_ID,
          status: "ACTIVE",
          last_seen_at: now,
          tenant_id: TENANT_ID,
          occ_lock: 0,
          created_at: now,
          update_at: now,
          created_by: USER_ID,
          updated_by: USER_ID,
        };
        sessions.set(s.id, s);
        return json(ok(s, "Session created successfully"), 201);
      },
    },
    terminate: {
      POST: async ({ request, params }: MsrBffEvent) => {
        const segs = new URL(request.url).pathname.split("/").filter(Boolean);
        const id = params?.sessionId ?? (segs.at(-1) === "terminate" ? segs.at(-2) : undefined);
        const s = id ? sessions.get(decodeURIComponent(id)) : undefined;
        if (!s) return fail("INVALID_REQUEST", "session not found", 400);
        s.status = "INACTIVE";
        s.occ_lock += 1;
        return new Response(null, { status: 204 });
      },
    },
  },
  admin: {
    sessions: {
      GET: async ({ request }: MsrBffEvent) => {
        const q = new URL(request.url).searchParams;
        const page = Number(q.get("page")) || 1;
        const size = Number(q.get("size")) || 50;
        const all = [...sessions.values()];
        const data = all.slice((page - 1) * size, (page - 1) * size + size);
        return json({
          ...ok(data),
          page: { number: page, size, total_records: all.length, count: data.length, sort: null },
        });
      },
    },
    settings: {
      GET: async () => json(ok({ ...settings, earliest_valid_timestamp: earliestValid() })),
      PUT: async ({ request }: MsrBffEvent) => {
        // Apply + echo back ONLY the submitted mutable keys (never the full doc).
        const u = (await request.json().catch(() => ({}))) as Partial<typeof settings>;
        const echoed: Partial<typeof settings> = {};
        if (typeof u.max_active_sessions === "number") {
          settings.max_active_sessions = echoed.max_active_sessions = u.max_active_sessions;
        }
        if (typeof u.data_retention_cron_expression === "string") {
          settings.data_retention_cron_expression = echoed.data_retention_cron_expression =
            u.data_retention_cron_expression;
        }
        if (typeof u.max_playback_range === "number") {
          settings.max_playback_range = echoed.max_playback_range = u.max_playback_range;
        }
        if (typeof u.earliest_valid_timestamp === "string") {
          settings.earliest_valid_timestamp = echoed.earliest_valid_timestamp = u.earliest_valid_timestamp;
        }
        return json(ok(echoed));
      },
    },
    terminate: {
      POST: async ({ request }: MsrBffEvent) => {
        const body = (await request.json().catch(() => ({}))) as { session_ids?: unknown };
        if (!Array.isArray(body.session_ids) || body.session_ids.length === 0)
          return fail("INVALID_REQUEST", "session_ids must be a non-empty array", 400);
        for (const id of body.session_ids as string[]) {
          const s = sessions.get(id);
          if (s?.status === "ACTIVE") {
            s.status = "INACTIVE";
            s.occ_lock += 1;
          }
        }
        return new Response(null, { status: 204 });
      },
    },
  },
  replay: {
    state: {
      GET: async ({ request }: MsrBffEvent) => {
        const ts = new URL(request.url).searchParams.get("timestamp");
        if (!ts) return fail("INVALID_REQUEST", "Missing required 'timestamp' query parameter", 400);
        const ms = Date.parse(ts);
        if (Number.isNaN(ms)) return fail("INVALID_REQUEST", "timestamp is invalid", 400);
        if (activeCount() === 0) return fail("BAD_REQUEST", "no active session found", 400);
        // /replay/state items: op "r", NO event_timestamp.
        const data = ENTITY_IDS.map((entity_id, i) => ({
          entity_id,
          entity_state: entityStateAt(i, ms),
          op: "r",
          table_name: STUB_TABLE,
        }));
        return json(ok(data, "Get initial state successfully"));
      },
    },
    events: {
      GET: async ({ request }: MsrBffEvent) => {
        const q = new URL(request.url).searchParams;
        const start = q.get("start");
        const end = q.get("end");
        if (!start || !end)
          return fail("INVALID_REQUEST", "Missing required 'start' and/or 'end' query parameters", 400);
        const a = Date.parse(start);
        const b = Date.parse(end);
        if (Number.isNaN(a) || Number.isNaN(b)) return fail("INVALID_REQUEST", "start or end is invalid", 400);
        // After termination the service 403s SESSION_TERMINATED on the events
        // poll — the exact signal the SDK worker keys on to end playback.
        if (activeCount() === 0)
          return fail("SESSION_TERMINATED", "Your session has been terminated. Please return to the lobby.", 403);
        // /replay/events items: op "u", WITH event_timestamp. Half-open [start, end)
        // so consecutive poll windows never duplicate or drop a boundary event.
        const data: unknown[] = [];
        for (let t = Math.ceil(a / STEP_MS) * STEP_MS; t < b; t += STEP_MS) {
          for (const [i, entity_id] of ENTITY_IDS.entries()) {
            data.push({
              entity_id,
              entity_state: entityStateAt(i, t),
              op: "u",
              table_name: STUB_TABLE,
              event_timestamp: new Date(t).toISOString(),
            });
          }
        }
        return json(ok(data, "Get events successfully"));
      },
    },
  },
};
```

## Verifying the stub actually drives the UI

Compiling is not enough (the type doesn't check the body). Confirm at
runtime: with `MSR_URL` unset, start a replay and watch
`replayStore.entities.size` go above 0 and `replayStore.playbackTime`
advance. If the lobby/controller work but `entities` stays empty, your
`entity_state` / `event_timestamp` / `op` / `table_name` field names don't
match the table above.

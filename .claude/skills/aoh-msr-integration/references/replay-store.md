# The replay store & worker pipeline

This is the heart of MSR and the part with no GIS analogue. Read it before
building a replay surface.

## The contract

`replayStore` (from `@mssfoobar/msr-web-sdk/store`) is a module-level
singleton wrapping a Web Worker. The worker polls the host's BFF, buffers
CDC change events, and applies them **frame-by-frame** at the configured
frame rate into a reactive map. Your app:

1. reads `replayStore.entities` (a `SvelteMap`) and **renders it however
   it likes**, and
2. reads `replayStore.status` / `.playbackTime` to drive surrounding UI.

The SDK never draws your entities. As playback advances the map mutates
and — because the store's getters are `$state`-backed — your `$derived`
reads re-paint automatically.

> **One store, one replay.** `replayStore` is a singleton (one worker,
> one set of reactive state). A single app/tab can run **one** replay at a
> time. There is no API to instantiate a second independent store.

## The entity model

```ts
interface Entity<T = EntityState> {  // EntityState defaults to any JSON value
  id: string;          // the row's entity id (bare — NOT the composite key)
  state: T;            // the row's columns as an object (your domain shape)
  tableName: string;   // the source CDC table, e.g. "gis.geo_entity"
}
```

Internally the map is keyed by a **composite** `` `${tableName}:${id}` `` so
ids stay unique across tables. `entity.id` is the bare id; `entity.state`
is whatever columns the CDC table carries (you know the shape for your
domain — narrow `T` for type safety).

Access helpers on the store:

| Helper | Returns |
|---|---|
| `entities` | `SvelteMap<string, Entity>` keyed by composite key — the full reactive set. |
| `getEntity(table, id)` | One `Entity` or `undefined`. |
| `getTableEntitiesMap(table)` | `Map<string, Entity>` keyed by **bare id** for one table (convenient for lookups). |
| `getEntitiesFromTable(table)` | `Entity[]` for one table. |
| `getAvailableTables()` | sorted unique table names currently present. |

## Rendering recipe (the payoff)

Read the map with `$derived` so it stays reactive, then render. A live
table, for example:

```svelte
<script lang="ts">
  import { replayStore } from "@mssfoobar/msr-web-sdk/store";

  const entities = $derived(
    [...replayStore.entities.values()].sort((a, b) =>
      `${a.tableName}:${a.id}`.localeCompare(`${b.tableName}:${b.id}`),
    ),
  );
</script>

{#each entities as e (e.tableName + ":" + e.id)}
  <tr>
    <td>{e.tableName}</td>
    <td>{e.id}</td>
    <td>
      {#each Object.entries(e.state ?? {}) as [k, v] (k)}
        <span>{k}: {String(v)}</span>
      {/each}
    </td>
  </tr>
{/each}
```

Same idea for any visualisation — feed positions to a GIS map, draw to a
`<canvas>`, animate DOM markers. The store is renderer-agnostic; you own
the view. Keep the render **pure** (read-only over `entities`); never write
back into the map.

> **Layout reminder.** When you render your visualisation in the same
> container as `<MultiSessionReplay>`, the SDK's controller has **no
> z-index** — put your backdrop behind it with `-z-10` inside an `isolate`
> container, and use a near-opaque background rather than `backdrop-blur`
> (blur creates a stacking context that covers the controller). Full
> layout recipe in `SKILL.md` Step 5.

## The reactive read surface

All getters are `$state`-backed (read-only — there are no setters):

| Getter | Type | Meaning |
|---|---|---|
| `status` | `ReplayStatus` | The state-machine value (below). |
| `entities` | `SvelteMap<string, Entity>` | Current reconstructed state. |
| `playbackTime` | `Date \| null` | The frame currently applied (advances during playback). |
| `replayStartTime` / `replayEndTime` | `Date \| null` | The fixed bounds of the active session's range. |
| `errorMessage` | `string \| null` | Last error / termination reason. |

## The status state machine

```text
idle ──dismissLobby──▶ selecting ──initiateReplay──▶ initializing ──▶ ready
                                                                        │
                                                       play ◀───────────┘
                                                        │   ▲
                                                        ▼   │ pause
                                                     playing┘
                                                        │
                                                        ▼ (range end)
                                                     finished ──restart──▶ initializing
   error  ◀── any failure            terminated ◀── admin terminate
```

| Status | Meaning | What renders |
|---|---|---|
| `idle` | Loaded; lobby card showing. | `<MultiSessionReplay>` lobby. |
| `selecting` | Lobby dismissed; choosing a start time. | The init card + date dialog. |
| `initializing` | Fetching initial state at the chosen timestamp. | Controller in a loading state. |
| `ready` | Initial state loaded; not yet playing. | Entities present; play enabled. |
| `playing` | Polling + applying changes; `playbackTime` advances. | Live updates. |
| `paused` | Polling paused; state frozen. | Frozen frame. |
| `finished` | Reached the end of the range. | Restart available. |
| `terminated` | Session ended by an admin. | "Session Terminated" card. |
| `error` | A failure occurred; `errorMessage` set. | "Playback Error" card. |

`<MultiSessionReplay>` + `<ReplayController>` drive all of these
transitions for you. You only need the control surface below if you're
building a **custom** controller.

## The imperative control surface (custom controllers)

`<MultiSessionReplay>` calls these for you. Use them directly only when
building a bespoke replay UI. Each is a no-op (with a warn log) if called
from a status it isn't valid in.

| Function | Effect |
|---|---|
| `configure({ bffBase })` | Point the store + worker at a non-default BFF base. (Components do this from `MsrProvider` context automatically.) |
| `dismissLobby()` | `idle → selecting`. |
| `setPlaybackRange(start, end)` | Fix the session range; `→ selecting`. |
| `initiateReplay(ts, perf?, stateFilter?, playbackFilter?)` | Start a session at `ts` (must be in range); `→ initializing`. |
| `play(speed?, frameRate?, pollWindowMs?, minPollingInterval?)` | `ready\|paused → playing`. |
| `pause()` | `playing → paused`. |
| `jumpToTimestamp(ts, perf?, stateFilter?, playbackFilter?)` | Scrub to `ts` (from `paused`/`ready`/`finished`/…); re-inits state there. |
| `restart()` | From `finished`, jump back to range start. |
| `setSpeed(speed, frameRate?, pollWindowMs?, minPollingInterval?)` | Change playback speed live. |
| `reset()` (async) | Terminate the **backend** session + worker session, clear state → `idle`. Await it before starting a new range to avoid races. |
| `resetToLobby()` | Terminate the worker session, clear state → `idle` (used by the error/terminated cards). |

A minimal custom flow: `setPlaybackRange(start, end)` →
`initiateReplay(start)` → (on `ready`) `play()`.

## How it works under the hood (for debugging)

- When the store module loads **in the browser**, it spawns the worker
  via `createMsrWorker()` (`new URL("../workers/msr.worker.js",
  import.meta.url)` — build-safe; **never** a `?worker` import). On the
  server the worker is never created (the `typeof window` guard).
- **Session handshake:** the worker emits `REQUEST_SESSION_CREATION`; the
  store calls `createSession()` (→ BFF `POST /session`) and posts
  `SESSION_CREATED` back. A failure here puts the store in `error`.
- **State application:** `c`/`u` change ops upsert the entity; `d` deletes
  it. `FRAME_UPDATE` also advances `playbackTime`. `READY` replaces the
  whole map with the initial snapshot.
- **Termination:** an admin terminate surfaces as the BFF returning
  `403 SESSION_TERMINATED` on the events poll; the worker reports it and
  the store goes `terminated` and clears entities.
- The store guards all browser-only code behind a `typeof window` check,
  so importing it during SSR is safe (the worker only spins up client-side).

## Performance tuning (`MsrPerformanceOptions`)

Pass via `<MultiSessionReplay performanceOptions={…}>` (or the control
functions). Defaults shown:

| Option | Default | Trade-off |
|---|---|---|
| `pollWindowMs` | `3000` | ms of CDC fetched per request. Larger = fewer/heavier requests; smaller = more frequent/lighter. |
| `frameRate` | `30` | Frames/sec applied. Higher = smoother + more CPU. |
| `maxBufferSize` | `1_000_000` | Max buffered changes before back-pressure. Higher = longer sessions without overflow, more memory. |
| `minPollingInterval` | `100` | ms floor between polls — prevents request storms at high speed. |

## Per-table stale-data filters

For domains where some tables carry long-lived rows you don't want
re-loaded at every scrub, supply a filter config. Both take a `tables`
list and a `getMinTimestamp` callback returning the cutoff — rows/events
older than it are excluded **for those tables only**.

```ts
<MultiSessionReplay
  …
  stateFilterConfig={{
    tables: ["patients", "medications"],
    getMinTimestamp: ({ targetTimestamp }) => {
      const weekAgo = new Date(targetTimestamp);
      weekAgo.setDate(weekAgo.getDate() - 7);
      return weekAgo;            // ignore those tables' rows older than a week before the target
    },
  }}
  playbackFilterConfig={{
    tables: ["logs"],
    getMinTimestamp: ({ targetTimestamp }) => {
      const dayAgo = new Date(targetTimestamp);
      dayAgo.setDate(dayAgo.getDate() - 1);
      return dayAgo;
    },
  }}
/>
```

Callback context: `stateFilterConfig.getMinTimestamp` gets
`{ targetTimestamp, playbackStartTime, playbackEndTime }`;
`playbackFilterConfig.getMinTimestamp` additionally gets
`{ eventWindowStart, eventWindowEnd }`. `targetTimestamp` is the
currently-targeted time (it changes on scrub/jump); the playback bounds
are fixed for the session. These are pure functions — keep them
side-effect-free.

## Pitfalls

- **`entities` is read-only.** Render from it; never mutate it. State is
  applied solely by the worker via change events.
- **`entity.id` is the bare id**, not the composite map key — use it
  directly for your domain lookups.
- **Don't poll the API client and the store in parallel** for the same
  data — the store *is* the consumer of the API client; reading
  `replayStore.entities` is the supported path.
- **Backend session leakage:** a hard navigation away mid-replay leaves a
  backend session until cleanup. The SDK terminates on `reset()` /
  `resetToLobby()`; for custom flows call one of those on teardown.

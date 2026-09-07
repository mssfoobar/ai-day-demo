# `@mssfoobar/msr-web-sdk` component & export catalog

Every public export, what it does, and the subpath to import it from. The
barrel (`@mssfoobar/msr-web-sdk`) re-exports everything; the subpaths
exist for granular imports / smaller dependency graphs. Both forms work.

Companion packages: the typed REST client is `@mssfoobar/msr-client`; wire
shapes (DTOs) originate in `@mssfoobar/msr-types`. This SDK bundles both
and re-exports the DTO types you need through **`@mssfoobar/msr-web-sdk/types`**
— import them from that subpath, not the bare `@mssfoobar/msr-types`
(transitive-only, unresolvable by name from a consumer under pnpm). The
SDK's own TypeScript type declarations that ship with the package are the
authoritative prop reference.

## Providers & context

| Export | Subpath | Role |
|---|---|---|
| `MsrProvider` (default) | `/provider` | App-root context. Mount once, high in the tree. See props below. |
| `GetMsrProviderContext()` | barrel | Returns the active `MsrProviderValue` or `undefined` (no provider mounted → components fall back to safe defaults; standalone/test use keeps working). |
| `GetDarkModeStore()` | barrel | The provider's `dark_mode_store`, or a fresh `writable(false)` if none. |
| `MSR_PROVIDER_CONTEXT` | barrel | The context `Symbol`. |
| types | barrel | `MsrProviderValue`, `MsrAuthHeaders`, `MsrAuthAdapter`. |

### `<MsrProvider>` props (`MsrProviderValue`)

| Prop | Type | Required | Notes |
|---|---|---|---|
| `dark_mode_store` | `Writable<boolean>` | **yes** | Host-owned theme store; the SDK only reacts to it. |
| `msr_bff_base` | `string` | no | Base path of the host's MSR BFF routes. Default `"/aoh/msr/api"`. Store + API client + worker derive every fetch path from this. |
| `msr_url` | `string` | no | Only if the host lets the SDK call `msr-service` directly. **Not used today** — leave unset; the SDK talks only to the BFF. |
| `auth_adapter` | `() => MsrAuthHeaders \| Promise<MsrAuthHeaders>` | no | Reserved for future SDK-internal fetches; not called today. `MsrAuthHeaders = Record<string,string>`. |
| `feature_flags` | `Record<string, boolean>` | no | SDK-side toggles. |
| `log_level` | `"trace"\|"debug"\|"info"\|"warn"\|"error"\|"silent"` | no | MSR SDK log verbosity. Default `"warn"`. Reactive — toggling it takes effect without remount. |
| `children` | `Snippet` | yes | App subtree. |

## The replay surface

| Export | Subpath | Role |
|---|---|---|
| `MultiSessionReplay` (default) | `/multi-session-replay` | The whole replay **chrome**: lobby card → date-picker dialog → init card → draggable controller. Sets the `"msr"` context the controller / init card read. Renders **none of your data** — you render `replayStore.entities`. |
| `MsrState` (type) | `/multi-session-replay` | Shape of the `"msr"` context it publishes (rarely needed directly). |
| `ReplayController` | `/replay-controller` | Play/pause, speed select, timeline scrub, terminate, minimise — the bottom-centre controller. Reads the `"msr"` context, so it only works **inside** a `<MultiSessionReplay>` (or a host that sets the same context). Prop: `openDateDialog` (`bindable boolean`). |
| `PlaybackButton` | `/replay-controller` | The play/pause button alone. Props `PlaybackButtonProps { isFetching, uiState }` (the type is exported from the barrel, not the subpath). |
| `ReplayInitCard` | `/replay-init-card` | The "select a playback date" card shown while `status === "selecting"`. Reads the `"msr"` context. Prop: `openDateDialog` (`bindable boolean`). |

### `<MultiSessionReplay>` props

| Prop | Type | Default | Notes |
|---|---|---|---|
| `lobbyTitle` | `string` | — (required) | Lobby card title. |
| `lobbyDescription` | `string` | — (required) | Lobby card body. |
| `lobbyButtonText` | `string` | — (required) | Lobby CTA label. |
| `playbackWindowMinutes` | `number` | `120` | Per-session replay **duration** (minutes). Drives the derived end time, DateTimePicker bounds, and the default buffers. |
| `startBufferMinutes` | `number` | `= playbackWindowMinutes` | Pushes the earliest selectable time forward (off the data-cleanup boundary). |
| `endBufferMinutes` | `number` | `= playbackWindowMinutes` | Pulls the latest selectable time back (off "now", where CDC may not have caught up). |
| `performanceOptions` | `MsrPerformanceOptions` | see below | Polling / frame-rate / buffer tuning. |
| `stateFilterConfig` | `MsrStateFilterConfig` | — | Exclude stale rows for named tables during initial-state load. |
| `playbackFilterConfig` | `MsrPlaybackFilterConfig` | — | Exclude stale events for named tables during playback. |
| `errorHandling` | `ErrorHandlingConfig` | Sonner toast | `onError` hook + `defaultOptions` (`showToast` / `logError` / `throwError`). |
| `blurTargetElement` | `HTMLElement` | — | An element to blur (`filter: blur(4px)`) while the lobby / error / terminated cards are up — e.g. a background map. |

`MsrPerformanceOptions`: `pollWindowMs` (3000 — ms of CDC fetched per
request), `frameRate` (30 fps), `maxBufferSize` (1_000_000 buffered
changes), `minPollingInterval` (100 ms floor between polls). Filter
configs + the full tuning rationale are in `replay-store.md`.

## Replay store & worker (non-component)

| Export | Subpath | Role |
|---|---|---|
| `replayStore` | `/store` | The reactive replay store — `status`, `entities` (a `SvelteMap`), `playbackTime`, control functions. **The integration surface you read.** Full contract in `replay-store.md`. |
| `ReplayStore` (type) | `/store` | `typeof replayStore`. |
| `createMsrWorker()` | barrel | Internal — the only sanctioned way the store spawns the replay Web Worker (`new URL(...)`, build-safe). Consumers don't call it. |

## Admin

| Export | Subpath | Role |
|---|---|---|
| `SessionListPage` | `/pages/session-list` | Ready-made admin table: search, terminate one / many, status badges. Reads `MsrProvider` context for the BFF base; terminate actions hit the BFF directly. |

### `<SessionListPage>` props

| Prop | Type | Required | Notes |
|---|---|---|---|
| `data` | `UserSession[]` | **yes** | The session list — the host loads it server-side (via the `admin.sessions` BFF route or `@mssfoobar/msr-client` directly). |
| `onRefresh` | `() => void \| Promise<void>` | no | Called after a terminate so the host can re-fetch `data` (e.g. `invalidateAll`). |

`UserSession` = the wire `Session` plus optional `fullName` / `userName`
(IAMS enrichment). Without enrichment the table shows the raw `user_id`.

## Form / shared primitives

These are `@mssfoobar/ui`-based building blocks the SDK uses internally
and re-exports for custom replay UIs. Exact props: the TypeScript type
declarations shipped with each component.

| Export | Subpath | Role |
|---|---|---|
| `DateTimePicker` | `/date-time-picker` | Date+time field used by the playback dialog. Controlled (`initialValue` + `onChange`) **or** uncontrolled (`value`). Common props: `id`, `min`, `max`, `disabled`, `precision` (`"seconds"\|"minutes"\|"hours"`), `ampm`, `placeholder`, `class`. Type `DateTimePickerProps`. |
| `TimelineSlider` | `/timeline-slider` | A standalone scrub slider (does **not** need the `"msr"` context). Props: `value`, `onChange`, `startTime`, `endTime`, `orientation`, `disabled`, `isLoading`, `onStartDragging`, `onStopDragging`, `class`, `ref`. Build a custom scrubber from it. |
| `Combobox` | `/combobox` | Searchable select (the controller's speed picker). Type `ComboboxOption`. |
| `Input` | `/input` | Styled text input, with `InputVariant` / `InputVariants` types. (The `inputVariants` value lives in the component module — import the `Input` component, not the variants helper, from this subpath.) |
| `ConfirmDialog` | `/confirm-dialog` | Confirmation modal. `ConfirmDialogState { title, description, submitText?, isAlert?, onSubmit? }`; `ConfirmDialogProps` adds `open`. |

## Server (BFF)

| Export | Subpath | Role |
|---|---|---|
| `msrBffHandlers(cfg)` | `/server` | Factory returning the seven route handler groups (`session.create/terminate`, `admin.sessions/settings/terminate`, `replay.state/events`; `admin.settings` has GET + PUT). Built on `@mssfoobar/msr-client`. Full details: `bff.md`. |
| types | `/server` | `MsrBffConfig` (extends `MsrClientConfig`, adds `enrichSessions`), `MsrBffHandlers`, `MsrBffEvent`, `MsrBffHandler`. |

For backend-free dev you supply your own `MsrBffHandlers`-shaped stub —
`references/stub.md` has a copyable one plus the wire shapes it must emit.

## Browser API client (advanced — for custom UIs)

The SDK uses a few of these internally (the store calls `createSession` /
`terminateSession`; `<MultiSessionReplay>` calls `getPlaybackRange`; the
worker does its own fetching). The rest are conveniences a host can call
directly when building a bespoke replay UI without `<MultiSessionReplay>`.
Each fetches the host's BFF (`bffBase` from `MsrApiOptions`, default
`/aoh/msr/api`) and routes errors through the SDK's Sonner-toast system.

| Export | Role |
|---|---|
| `createSession(opts?)` | `POST <bff>/session` → `{ sessionId }`. |
| `terminateSession(id, opts?)` | `POST <bff>/session/:id/terminate`. |
| `getPlaybackRange(opts?)` | Derives the selectable `{ maxDays, startDate, endDate, earliestValidDate? }` from `GET <bff>/admin/settings`. |
| `getInitialState(ts, opts?)` | `GET <bff>/replay/state?timestamp=…` → `ReplayEvent[]`. |
| `getReplayEvents(start, end, opts?)` | `GET <bff>/replay/events?start=…&end=…` → `ReplayEvent[]`. |
| `listSessions(opts?)` | `GET <bff>/admin/sessions` → `{ sessions, totalRecords }`. |
| `updateMaxActiveSessions(n, opts?)` | `PUT <bff>/admin/settings`. |
| `configureMsrErrorHandling(cfg)` / `handleMsrError(err, ctx, opts?)` | Install / invoke the error handler. |
| `MsrApiOptions` (type) | `{ bffBase?: string }`. |

## Nav, paths, constants, utils, types

| Export | Subpath | Role |
|---|---|---|
| `msrNav` | `/nav` | `{ code: "MSR", header: { name, url }, sidebar: [{ name, url, icon }] }`. Feed to your sidebar/router. Types `MsrNav`, `MsrNavSidebarEntry`. |
| `DEFAULT_MSR_BFF_BASE` | barrel | `"/aoh/msr/api"`. |
| `msrBffPaths(base?)` | barrel | The BFF route map relative to `base` (`SESSION_CREATE`, `REPLAY_STATE`, …). |
| `ROUTES` | barrel | `{ BASE: "/aoh/msr", SESSIONS: "/aoh/msr/sessions" }`. |
| `PAGINATION_LIMIT` | barrel | `1000` (default page size). |
| `DEFAULT_MAX_REPLAY_DURATION` | barrel | `7200` (2 h, seconds). |
| `TimePeriod` | barrel | `AM` / `PM` enum. |
| `TIMEPICKER_FORMAT` | barrel | `"YYYY-MM-DD[T]HH:mm:ss"` (dayjs). |
| `debounce`, `filterList` | barrel | Small UI utils. |
| types | `/types` | `Entity`, `ChangeEvent`, `ReplayStatus`, `PlaybackRange`, `ReplayEvent`, `UserSession`, `SessionData`, `MsrPerformanceOptions`, `MsrStateFilterConfig`, `MsrPlaybackFilterConfig`, `MsrError`, `ErrorHandlingConfig`, plus re-exports of `Session` / `SessionStatus` / `AdminSettings` / `AdminSettingsUpdate` / `PageMeta` from `@mssfoobar/msr-types`. |

## Import-style cheat sheet

```ts
// Barrel (everything):
import { MultiSessionReplay, replayStore, SessionListPage, msrNav } from "@mssfoobar/msr-web-sdk";

// Subpaths (granular):
import MsrProvider from "@mssfoobar/msr-web-sdk/provider";
import MultiSessionReplay from "@mssfoobar/msr-web-sdk/multi-session-replay";
import { replayStore } from "@mssfoobar/msr-web-sdk/store";
import { SessionListPage } from "@mssfoobar/msr-web-sdk/pages/session-list";
import { msrBffHandlers } from "@mssfoobar/msr-web-sdk/server";   // server-only
import { msrNav } from "@mssfoobar/msr-web-sdk/nav";
import type { Entity, UserSession } from "@mssfoobar/msr-web-sdk/types";
```

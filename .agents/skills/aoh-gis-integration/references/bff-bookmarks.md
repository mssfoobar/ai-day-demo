# BFF reference — bookmark proxy (`+page.server.ts`)

The complete server-side BFF for the GIS bookmark feature. Copy this into
your map route's `+page.server.ts` (next to the `+page.svelte` that mounts
`<MapBookmarkManager>`) and adapt the auth + env bits to your app.

`<MapBookmarkManager>` (client) drives two SvelteKit form actions on the
same route — `?/create_bookmark` and `?/delete_bookmark` — and reads an
initial list from `load`. The BFF is what lets those work **without
exposing the gateway URL or the auth token to the browser**.

## The three gotchas (why this file isn't trivial)

1. **Fail-closed auth.** With auth enabled (`IAM_URL` set), read the
   token from `locals.authResult` (SDS-backed, server-only). If absent,
   send **no** `Authorization` and let the upstream 401 — never fall back
   to a dev/test identity against a real service. **The token field name
   depends on your base** — see "Which `authResult` shape" below.
2. **devalue, not JSON.** The component posts via
   `superForm({ dataType: 'json' })`, which serializes the whole form
   with **devalue** under `__superform_json` field(s). Decode with
   `devalue.parse(getAll('__superform_json').join(''))`. A `JSON.parse`
   silently yields an empty payload → every create fails validation.
3. **Return a `SuperValidated`-shaped `form`.** The client's `superForm`
   enhance throws "No form data returned from ActionResult" unless the
   result has a `form` object with `id` (`""` to match the component's
   default formId), `valid`, `errors`, `data`. Without it the component's
   `onResult` (which mutates the map) never fires.

## Which `authResult` shape (base-dependent)

`locals.authResult` is a **discriminated union populated by your base's auth
hook** — its exact shape differs by base, and the token field name is the
part that bites. Match your base:

- **`aoh-web-init` base (the recommended base — use this path as written).**
  The scaffold defines its own `AuthResult` in
  `$lib/aoh/core/provider/auth/auth`:
  ```ts
  type AuthResult =
    | { success: true; claims: AuthClaims; access_token?: string }  // snake_case
    | { success: false };
  ```
  Read the token as `authResult.access_token` after narrowing on
  `authResult.success` — exactly as `authHeaders()` does above.

- **`@mssfoobar/auth-sdk` base (the alternative).** Here
  `AuthResult` comes from `@mssfoobar/auth-sdk` and the token field is
  **`accessToken`** (camelCase). Swap the import to
  `import type { AuthResult } from "@mssfoobar/auth-sdk"` and read
  `authResult.accessToken`.

Both branches gate on `authResult.success` first, then read the (optional)
token field; a missing token means fail-closed (no `Authorization`).

## REST contract (gis-service)

| Action | Method + path | Success | Body |
|---|---|---|---|
| list | `GET {GIS_URL}/bookmark` | 200 | `{ data: BookmarkDto[] }` |
| create | `POST {GIS_URL}/bookmark` | 200 | `{ data: BookmarkDto }` |
| delete | `DELETE {GIS_URL}/bookmark/id/{id}` | 204 | — |

`BookmarkDto`: `{ id, occ_lock, name, lon, lat, alt, zoom, pitch, yaw, roll }`.
The SDK's `MapBookmark` shape is
`{ id, occ_lock, name, camera_view: { position: [lon,lat,alt], zoom, direction: [pitch,yaw,roll] } }`
(`MapBookmarkManager` calls `fly_to(bookmark.camera_view)` — the key
**must** be `camera_view`, not `location`) — convert in `dtoToSdkShape`.

## Full implementation

```ts
import { fail, type Actions, type ServerLoad } from "@sveltejs/kit";
import { env } from "$env/dynamic/private";
import { logger } from "@mssfoobar/logger";
import { randomUUID } from "node:crypto";
import { parse as devalueParse } from "devalue";
// aoh-web-init scaffold: the AuthResult type lives in the scaffold, not in
// a package — its token field is `access_token` (snake_case). On an
// auth-sdk base it's `import type { AuthResult } from "@mssfoobar/auth-sdk"`
// and the field is `accessToken` (camelCase). See "Which authResult shape"
// above.
import type { AuthResult } from "$lib/aoh/core/provider/auth/auth";

const log = logger.child({ src: "aoh/gis/+page.server.ts" });
const MAX_BOOKMARK_NAME_LENGTH = 55;

type BookmarkDto = {
  id: string; occ_lock: number; name: string;
  lon: number; lat: number; alt: number;
  zoom: number; pitch: number; yaw: number; roll: number;
};

// --- Auth: forward the user's token; FAIL CLOSED. -----------------------
// When IAM_URL is set, read the SDS-backed token from locals.authResult
// (server-side only — it never reaches the browser). No valid token →
// send no Authorization and let upstream 401. Never substitute a dev
// identity against a real service. Only when auth is DISABLED may you use
// a deterministic dev bearer (test/e2e tiers).
function authHeaders(locals: App.Locals): Record<string, string> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (env.IAM_URL) {
    const authResult = (locals as { authResult?: AuthResult }).authResult;
    // aoh-web-init scaffold: token field is `access_token` (snake_case).
    // On an auth-sdk base use `authResult.accessToken`.
    if (authResult?.success && authResult.access_token) {
      headers.Authorization = `Bearer ${authResult.access_token}`;
    }
    return headers; // fail closed: no Authorization when no valid token
  }
  // Auth disabled — deterministic dev identity (see references/auth-and-config.md).
  headers.Authorization = TEST_BEARER;
  return headers;
}

function dtoToSdkShape(b: BookmarkDto) {
  return {
    id: b.id,
    occ_lock: b.occ_lock,
    name: b.name,
    camera_view: {
      position: [b.lon, b.lat, b.alt] as [number, number, number],
      zoom: b.zoom,
      direction: [b.pitch, b.yaw, b.roll] as [number, number, number],
    },
  };
}

// Minimal SuperValidated-shaped object so the client superForm enhance
// accepts the result. isValidationObject only requires id (string),
// valid (boolean), errors. id "" matches the component's default formId.
function superformResult(data: Record<string, unknown>) {
  return { id: "", valid: true, posted: true, errors: {}, data };
}

// The form is devalue-encoded under __superform_json (chunked). Decode
// exactly as superforms' server-side superValidate does. Fall back to
// flat fields so a consumer wiring without dataType:'json' still works.
async function readBookmarkPayload(request: Request): Promise<Record<string, unknown>> {
  const formData = await request.formData();
  const chunks = formData.getAll("__superform_json");
  if (chunks.length > 0) {
    try {
      const serialized = chunks.map((c) => (typeof c === "string" ? c : "")).join("");
      const parsed = devalueParse(serialized) as unknown;
      return typeof parsed === "object" && parsed !== null
        ? (parsed as Record<string, unknown>)
        : {};
    } catch (error) {
      log.warn({ error }, "Failed to devalue-parse __superform_json payload");
      return {};
    }
  }
  const out: Record<string, unknown> = {};
  formData.forEach((value, key) => { out[key] = value; });
  return out;
}

function coerceNumber(v: unknown, fallback = 0): number {
  if (typeof v === "number" && Number.isFinite(v)) return v;
  if (typeof v === "string") { const n = Number(v); return Number.isFinite(n) ? n : fallback; }
  return fallback;
}
function isUuidV4(v: unknown): v is string {
  return typeof v === "string" &&
    /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(v);
}

export const load: ServerLoad = async ({ fetch, locals }) => {
  const gisUrl = env.GIS_URL;
  if (!gisUrl) return { bookmarks: [] }; // or seed a stub list for local dev
  try {
    const res = await fetch(`${gisUrl}/bookmark`, { method: "GET", headers: authHeaders(locals) });
    if (!res.ok) { log.error({ status: res.status }, "fetch bookmarks failed"); return { bookmarks: [] }; }
    const body = (await res.json()) as { data: BookmarkDto[] };
    return { bookmarks: body.data.map(dtoToSdkShape) };
  } catch (error) {
    log.error({ error }, "error reaching GIS_URL"); return { bookmarks: [] };
  }
};

export const actions: Actions = {
  create_bookmark: async ({ request, fetch, locals }) => {
    const payload = await readBookmarkPayload(request);
    const name = typeof payload.name === "string" ? payload.name.trim() : "";
    if (name.length === 0 || name.length > MAX_BOOKMARK_NAME_LENGTH) {
      return fail(400, { message: "invalid_name" });
    }
    const lon = coerceNumber(payload.lon), lat = coerceNumber(payload.lat);
    if (lon < -180 || lon > 180 || lat < -90 || lat > 90) {
      return fail(400, { message: "invalid_coordinates" }); // WGS84; matches gis-service validator
    }
    const data = {
      name, lon, lat,
      alt: coerceNumber(payload.alt), zoom: coerceNumber(payload.zoom),
      pitch: coerceNumber(payload.pitch), yaw: coerceNumber(payload.yaw),
      roll: coerceNumber(payload.roll),
    };
    const gisUrl = env.GIS_URL;
    if (!gisUrl) { // stub mode
      const dto: BookmarkDto = { id: randomUUID(), occ_lock: 0, ...data };
      return { form: superformResult(data), bookmark: dto };
    }
    const res = await fetch(`${gisUrl}/bookmark`, {
      method: "POST", body: JSON.stringify(data), headers: authHeaders(locals),
    });
    if (!res.ok) {
      log.error({ status: res.status }, "upstream rejected create_bookmark");
      return fail(res.status, { form: superformResult(data), message: "upstream_error" });
    }
    const result = (await res.json()) as { data: BookmarkDto };
    return { form: superformResult(data), bookmark: result.data }; // component reads result.data.bookmark
  },

  delete_bookmark: async ({ request, fetch, locals }) => {
    const payload = await readBookmarkPayload(request);
    if (!isUuidV4(payload.id)) return fail(400, { message: "invalid_id" });
    const id = payload.id;
    const gisUrl = env.GIS_URL;
    if (!gisUrl) return { form: superformResult({ id }) }; // stub mode
    const res = await fetch(`${gisUrl}/bookmark/id/${id}`, {
      method: "DELETE", headers: authHeaders(locals),
    });
    if (res.status !== 204) {
      log.error({ status: res.status }, "upstream rejected delete_bookmark");
      return fail(res.status, { form: superformResult({ id }), message: "upstream_error" });
    }
    return { form: superformResult({ id }) }; // delete onResult reads result.data.form.data.id
  },
};
```

## Notes

- `TEST_BEARER` is only for auth-disabled local/e2e runs. See
  `references/auth-and-config.md` for minting a parseable unsigned JWT
  that the test stack's userinfo round-trip accepts. **Never** ship an
  unsigned token against a real issuer.
- Stub mode (`GIS_URL` unset) keeps the page browseable end-to-end with
  no backend — invaluable for local dev and the fast E2E tier. Use a
  module-scoped `Map` to persist stub bookmarks across requests within a
  worker if you want create/delete to round-trip in stub mode.
- The same `authHeaders` pattern applies to **any** GIS REST call you
  proxy (entities, layers, etc.), not just bookmarks.

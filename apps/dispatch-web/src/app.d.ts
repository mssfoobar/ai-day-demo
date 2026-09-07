// See https://kit.svelte.dev/docs/types#app

// for information about these interfaces
declare global {
	namespace App {
		/**
		 * The shape `page.error` carries. `hooks.server.ts`'s `handleError`
		 * returns it, so `+error.svelte` can render the shared AOH error surface
		 * instead of a bare status plus framework message.
		 *
		 * `message` is inherited from SvelteKit's own declaration and stays
		 * required; the hook sets it to the same string as `userMessage` so no
		 * consumer of the framework field renders developer-facing text.
		 *
		 * The other four are optional because an *expected* error — anything
		 * thrown via `error(status, …)`, including SvelteKit's own 404 — bypasses
		 * `handleError` and arrives as `{ message }` alone; `+error.svelte`
		 * derives the presentation from `page.status` in that case.
		 *
		 * `errorMessage` and `details` are deliberately absent: they are
		 * developer-facing and must never reach a user-facing surface.
		 */
		interface Error {
			/** Machine-readable code, `UPPER_SNAKE_CASE` (e.g. `INTERNAL_ERROR`). */
			errorCode?: string;
			/** RFC 3339 instant the failure was rendered. */
			timestamp?: string;
			/** The only failure string a user is shown. */
			userMessage?: string;
			/** Whether the error surface may offer a retry. */
			isRetryable?: boolean;
			/**
			 * 32-hex OpenTelemetry trace id — present only when a span was active.
			 * Never a placeholder and never the literal `unknown`.
			 */
			trace_id?: string;
		}
		/**
		 * Empty on purpose. This app has no authentication, so there is no
		 * `authResult`, no OIDC client, and no post-login redirect to carry —
		 * see the openspec change `baseline-dispatch-console` (design.md D1).
		 */
		// eslint-disable-next-line @typescript-eslint/no-empty-object-type
		interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}

	type HTTPResponseBody<T> = {
		data?: T;
		message?: string;
		sent_at?: ISO8601Date;
		errors?: Array<Error>;

		// Pagination block populated for paginated endpoints. `sort` echoes the
		// applied clauses as ["field,direction", ...] (array, not a joined string)
		// so callers can parse it back symmetrically.
		page?: {
			number: number;
			size: number;
			total_records: number;
			count: number;
			sort?: string[];
		};
	};
}

// Keeps this file a module so `declare global` above is an augmentation rather than a
// redeclaration. The scaffold got this implicitly from its auth imports; those are gone
// with the auth layer, so the marker has to be explicit.
export {};

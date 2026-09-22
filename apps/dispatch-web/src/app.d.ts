// See https://kit.svelte.dev/docs/types#app

// @mssfoobar/gis-web-sdk's component `.d.ts` files declare their props as
// `interface Props extends Gis.MapOptions`. That global namespace only resolves when
// the SDK's `types-namespace.d.ts` is in the app's TypeScript program — without this
// reference every `<Map>` prop type-errors as an unknown property, even though the
// build succeeds. Referenced once here rather than per page.
/// <reference types="@mssfoobar/gis-web-sdk/types" />

import type { AuthResult } from '$lib/aoh/core/provider/auth/auth';
import type { Configuration } from 'openid-client';

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
		 * What `hooks.server.ts`'s auth handle puts on every request.
		 *
		 * `authResult` carries the operator's *claims* and — on the success branch —
		 * the access token, read from SDS server-side. The token is deliberately
		 * never returned from a `load`: the browser gets only the opaque
		 * `web_auth_session_id` cookie.
		 */
		interface Locals {
			authResult: AuthResult;
			clients?: {
				oidc_config?: Configuration;
			};
			/** Where the operator was heading before they were sent to sign in. */
			originalUrl?: string;
		}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}

	/** RFC 3339 / ISO 8601 instant, as the wire carries it. */
	type ISO8601Date = string;

	/**
	 * One entry in the AOH failure envelope's `errors` array — `"<field>: <reason>"`.
	 *
	 * Declared explicitly because inside `declare global` a bare `Error` resolves to the
	 * global JavaScript exception type (`name`/`message`/`stack`), which is not what a
	 * response envelope carries.
	 */
	type HTTPResponseError = { message: string };

	type HTTPResponseBody<T> = {
		data?: T;
		message?: string;
		sent_at?: ISO8601Date;
		errors?: Array<HTTPResponseError>;

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
// redeclaration. The auth imports at the top would do it implicitly; the explicit marker
// stays so removing an import cannot silently turn this into a redeclaration.
export {};

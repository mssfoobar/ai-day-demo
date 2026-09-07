<script lang="ts">
	// The app's single error surface, built on `@mssfoobar/ui`'s shared
	// `ErrorSurface` so a newly scaffolded app is born conformant with the AOH
	// error contract — no error-handling code for its author to write.
	//
	// Two shapes arrive here:
	//
	//  * an *unexpected* exception, converted by `hooks.server.ts`'s
	//    `handleError` into the conformant shape (`errorCode`, `timestamp`,
	//    `userMessage`, `isRetryable`, and `trace_id` when a span was active); and
	//  * an *expected* error — anything thrown via `error(status, …)`, including
	//    SvelteKit's own 404 — which bypasses `handleError` and arrives as
	//    `{ message }` alone, so the presentation is derived from `page.status`.
	//
	// `page.error.message` is deliberately never rendered: in the expected case it
	// is a framework/developer string, and developer-facing text must never reach
	// a user. `@mssfoobar/errors` resolves the user-facing message instead.
	import { page } from '$app/state';
	import { invalidateAll } from '$app/navigation';
	import { ErrorSurface, type ErrorSurfaceFailure } from '@mssfoobar/ui/error-surface';
	import {
		errorCodeForClass,
		failureClassForStatus,
		isRetryableByClass,
		resolveUserMessage
	} from '@mssfoobar/errors';

	function resolveFailure(): ErrorSurfaceFailure {
		const error = page.error;
		if (error?.userMessage) {
			return {
				userMessage: error.userMessage,
				errorCode: error.errorCode,
				timestamp: error.timestamp,
				trace_id: error.trace_id,
				isRetryable: error.isRetryable
			};
		}

		// No enriched payload: derive everything from the status.
		const failureClass = failureClassForStatus(page.status);
		const errorCode = error?.errorCode ?? errorCodeForClass(failureClass);
		return {
			userMessage: resolveUserMessage(errorCode),
			errorCode,
			timestamp: error?.timestamp,
			trace_id: error?.trace_id,
			isRetryable: error?.isRetryable ?? isRetryableByClass(failureClass)
		};
	}

	const failure = $derived(resolveFailure());

	// Re-run the failed route's load functions in place. Deliberately not
	// `location.reload()`: the retry affordance must not navigate or reload.
	async function retry() {
		await invalidateAll();
	}
</script>

<div class="grid min-h-screen place-items-center p-6">
	<ErrorSurface {failure} variant="page" title={`Error ${page.status}`} onRetry={retry} />
</div>

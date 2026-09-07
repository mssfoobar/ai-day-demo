import { env as envPrivate } from '$env/dynamic/private';
import { log } from '$lib/aoh/core/logger/Logger';
import type { Handle, HandleServerError } from '@sveltejs/kit';
import { sequence } from '@sveltejs/kit/hooks';
import { createObservabilityHandle } from '@mssfoobar/observability/sveltekit';
import { trace } from '@opentelemetry/api';
import {
	type ErrorCatalogue,
	errorCodeForClass,
	failureClassForStatus,
	isTraceId,
	resolveIsRetryable,
	resolveUserMessage
} from '@mssfoobar/errors';

log.info('Environment Mode (NODE_ENV): ' + envPrivate.NODE_ENV);

/* -------------------------------------------------------------------------- */
/*                             NO AUTHENTICATION                              */
/* -------------------------------------------------------------------------- */

/*
 * This app has NO authentication. The `aoh-web-init` scaffold ships an OIDC/Keycloak
 * layer here; it was removed deliberately for the workshop baseline — see the openspec
 * change `baseline-dispatch-console` (design.md D1).
 *
 * Worth knowing before anyone reinstates it: the scaffold performed OIDC discovery in a
 * TOP-LEVEL `await` in this module. That runs at server startup, before any routing, so
 * with no reachable identity provider every route — including unauthenticated ones —
 * failed with HTTP 500. Auth here is all-or-nothing; it cannot be left inert.
 *
 * Restoring auth means restoring the whole set together: this module's discovery +
 * auth handle, `src/lib/aoh/core/provider/auth/`, the `(public)/aoh/api/auth/*` routes,
 * the `(private)` group and its layout, the gateway proxy, and the `App.Locals` fields
 * in `src/app.d.ts` — plus running `iams-keycloak`, `iams-aas`, `sds-server` and `valkey`.
 */

/* -------------------------------------------------------------------------- */
/*                           VALIDATE ENV VARIABLES                           */
/* -------------------------------------------------------------------------- */

const missingEnvVars: string[] = [];

if (!envPrivate.ORIGIN) missingEnvVars.push('ORIGIN');

if (!envPrivate.FRAME_ANCESTORS) {
	log.warn(`FRAME_ANCESTORS is not set.`);
}

if (!envPrivate.X_FRAME_OPTIONS) {
	log.warn(`X_FRAME_OPTIONS is not set.`);
}

if (missingEnvVars.length > 0) {
	log.error({ missingEnvVars }, 'Server cannot start with missing environment variables');
	process.exit(1);
}

/* -------------------------------------------------------------------------- */
/*                              SECURITY HEADERS                              */
/* -------------------------------------------------------------------------- */

const securityHeadersHandle: Handle = async ({ event, resolve }) => {
	const response = await resolve(event);

	const frameAncestors = envPrivate.FRAME_ANCESTORS;
	if (frameAncestors) {
		response.headers.set('Content-Security-Policy', `frame-ancestors ${frameAncestors};`);
	}

	const xFrameOptions = envPrivate.X_FRAME_OPTIONS;
	if (xFrameOptions) {
		response.headers.set('X-Frame-Options', xFrameOptions);
	}

	return response;
};

/* -------------------------------------------------------------------------- */
/*                                OBSERVABILITY                               */
/* -------------------------------------------------------------------------- */

// createObservabilityHandle renames the auto-instrumented server span to the
// matched SvelteKit route (low cardinality), so it runs outermost.
// (No-op when OTEL is disabled — there is simply no active span.)
export const handle: Handle = sequence(createObservabilityHandle(), securityHeadersHandle);

/* -------------------------------------------------------------------------- */
/*                                 ERROR SEAM                                 */
/* -------------------------------------------------------------------------- */

/**
 * `errorCode` → copy catalogue for this app. Empty on purpose: `@mssfoobar/errors`
 * ships the *seam*, not the copy, and an unmapped code resolves to the package's
 * generic message rather than to a blank UI. Add entries here — one file, keyed
 * by the code — instead of hard-coding strings at call sites.
 */
const ERROR_CATALOGUE: ErrorCatalogue = {};

/**
 * Turn an unhandled exception into the conformant user-facing shape that
 * `+error.svelte` renders through `@mssfoobar/ui`'s `ErrorSurface`.
 *
 * Three rules this enforces:
 *
 *  1. **The exception never reaches the client.** The returned `message` is the
 *     resolved `userMessage`, so even SvelteKit's own static fallback page shows
 *     a user-facing string and nothing else. Never forward `err.message` here.
 *  2. **`trace_id` is the active span's or absent** — never a minted UUID, never
 *     the literal `unknown`. `isTraceId` also rejects the all-zero id a
 *     non-recording span reports, so a deploy with OTLP export off omits the
 *     field and the error page degrades to `errorCode` + `timestamp`.
 *  3. **One log record**, at the level the status implies (5xx → ERROR,
 *     4xx → WARN). This hook is the rendering layer, so call sites must not log
 *     the same failure first.
 *
 * SvelteKit calls this only for *unexpected* errors — an `error(404, …)` thrown
 * by a load function bypasses it — and it must never throw.
 *
 * It runs inside `createObservabilityHandle`'s active server span, so the trace
 * id here is the one the response and the logs are correlated by.
 */
export const handleError: HandleServerError = ({ error: cause, status }) => {
	const failureClass = failureClassForStatus(status);
	const errorCode = errorCodeForClass(failureClass);
	const userMessage = resolveUserMessage(errorCode, ERROR_CATALOGUE);

	const activeTraceId = trace.getActiveSpan()?.spanContext().traceId;
	const traceId = isTraceId(activeTraceId) ? activeTraceId : undefined;

	const pageError = {
		message: userMessage,
		errorCode,
		timestamp: new Date().toISOString(),
		userMessage,
		isRetryable: resolveIsRetryable({ errorCode, failureClass, catalogue: ERROR_CATALOGUE }),
		...(traceId === undefined ? {} : { trace_id: traceId })
	};

	try {
		const fields = {
			err: cause,
			status,
			errorCode,
			...(traceId === undefined ? {} : { trace_id: traceId })
		};
		if (status >= 500) {
			log.error(fields, 'unhandled server error');
		} else {
			log.warn(fields, 'unhandled server error');
		}
	} catch {
		// A broken logger must not turn a handled 500 into an unrenderable one.
	}

	return pageError;
};

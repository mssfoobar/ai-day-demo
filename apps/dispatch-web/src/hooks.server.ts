import { env as envPrivate } from '$env/dynamic/private';
import { env } from '$env/dynamic/public';
import {
	type Configuration,
	discovery,
	type DiscoveryRequestOptions,
	allowInsecureRequests
} from 'openid-client';
import {
	authenticate,
	LOGIN_API,
	type AuthResult,
	getSdsClient
} from '$lib/aoh/core/provider/auth/auth';
import { log } from '$lib/aoh/core/logger/Logger';
import { error, type Handle, type HandleServerError } from '@sveltejs/kit';
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
/*                               AUTHENTICATION                               */
/* -------------------------------------------------------------------------- */

/*
 * Restored as one piece from the `aoh-web-init` scaffold, reversing
 * `baseline-dispatch-console` D1. Auth here is all-or-nothing and cannot be left inert:
 * `discovery()` below is a TOP-LEVEL await, so it runs at server startup before any
 * routing. With no reachable identity provider *every* route — including the health
 * probes — fails with 500. That is why the baseline deleted the layer rather than
 * disabling it, and why it returns whole.
 *
 * Over plain http, OIDC_ALLOW_INSECURE_REQUESTS=1 is not optional for the same reason:
 * without it this discovery throws at startup and the app never serves anything.
 *
 * Tokens live in SDS. The browser gets only `web_auth_session_id`.
 */

const discoveryRequestOptions: DiscoveryRequestOptions = {};
if (envPrivate.OIDC_ALLOW_INSECURE_REQUESTS === '1') {
	discoveryRequestOptions.execute = [allowInsecureRequests];
}

const oidc_config: Configuration = await discovery(
	new URL(envPrivate.IAM_URL!),
	envPrivate.IAM_CLIENT_ID ?? '',
	envPrivate.IAM_CLIENT_SECRET || undefined,
	undefined,
	discoveryRequestOptions
);

/* -------------------------------------------------------------------------- */
/*                           VALIDATE ENV VARIABLES                           */
/* -------------------------------------------------------------------------- */

const missingEnvVars: string[] = [];

// Public (browser-exposed) variables.
if (!env.PUBLIC_DOMAIN) {
	log.warn(
		'PUBLIC_DOMAIN is not set, cookie behaviour might not be as expected! This is desired only in local development using localhost.'
	);
}
if (!env.PUBLIC_COOKIE_SAMESITE) {
	log.warn("PUBLIC_COOKIE_SAMESITE is not set, defaulting to 'lax'.");
}
// The cookie prefix is what makes the session cookie `web_auth_session_id`, which is the
// name rtus-seh's seeded `rtus.session-id.cookienames` already carries. Change it and the
// map's SSE subscription is rejected with nothing in either log to explain it.
if (!env.PUBLIC_COOKIE_PREFIX) missingEnvVars.push('PUBLIC_COOKIE_PREFIX');

// Private variables.
if (!envPrivate.IAM_CLIENT_ID) missingEnvVars.push('IAM_CLIENT_ID');
if (!envPrivate.IAM_URL) missingEnvVars.push('IAM_URL');
if (!envPrivate.ORIGIN) missingEnvVars.push('ORIGIN');
if (!envPrivate.LOGIN_DESTINATION) missingEnvVars.push('LOGIN_DESTINATION');
if (!envPrivate.LOGIN_PAGE) {
	log.warn(`LOGIN_PAGE is not set, defaulting to ${LOGIN_API}`);
}

// SDS is not optional here, whatever the scaffold's fallback allows: the cookie-only flow
// puts a usable bearer in the browser, and rtus-seh authorises the map's SSE stream by
// resolving the SDS session cookie.
if (!envPrivate.SDS_URL) {
	log.warn(
		'SDS_URL is not set — falling back to cookie-held tokens. That is a diagnostic mode: it puts a bearer in the browser and the map will not be able to subscribe to rtus-seh.'
	);
}

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
/*                             TOKENS AND COOKIES                             */
/* -------------------------------------------------------------------------- */

/*
 * This handle only *resolves* the session; it never redirects. The sign-in redirect is
 * the `(private)` layout's job, which is what keeps `/livez` and `/readyz` — and the
 * `(public)/aoh/api/auth/*` routes themselves — reachable with no session.
 */
const authHandle: Handle = async ({ event, resolve }) => {
	let sdsClient = undefined;
	if (envPrivate.SDS_URL) {
		sdsClient = await getSdsClient();
	}

	try {
		const authResult: AuthResult = await authenticate(
			oidc_config,
			event.cookies,
			event.url,
			sdsClient
		);
		event.locals.clients = {
			oidc_config
		};
		event.locals.authResult = authResult;

		// Preserve the operator's intended destination across the sign-in round trip.
		if (!authResult.success && !event.url.pathname.startsWith('/aoh/api/auth/')) {
			event.locals.originalUrl = event.url.pathname + event.url.search;
		}
	} catch (err) {
		log.error({ err }, 'critical authentication failure');

		if (envPrivate.NODE_ENV === 'development') {
			error(500, {
				message: (err as Error).message
			});
		}
	} finally {
		if (sdsClient) {
			await sdsClient.close();
		}
	}

	const response = await resolve(event);

	// 🛡️ Security headers (only when the environment sets them).
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
export const handle: Handle = sequence(createObservabilityHandle(), authHandle);

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

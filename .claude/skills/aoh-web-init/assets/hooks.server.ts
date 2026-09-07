import { env as envPrivate } from '$env/dynamic/private';
import { env } from '$env/dynamic/public';
import {
	type Configuration,
	discovery,
	type DiscoveryRequestOptions,
	allowInsecureRequests
} from 'openid-client';
import dayjs from 'dayjs';
import Duration from 'dayjs/plugin/duration';
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

dayjs.extend(Duration);

log.info('Environment Mode (NODE_ENV): ' + envPrivate.NODE_ENV);

/* -------------------------------------------------------------------------- */
/*                               AUTHENTICATION                               */
/* -------------------------------------------------------------------------- */

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

//Public env variables
if (!env.PUBLIC_DOMAIN) {
	log.warn(
		'PUBLIC_DOMAIN is not set, cookie behaviour might not be as expected! This is desired only in local development using localhost.'
	);
}
if (!env.PUBLIC_COOKIE_SAMESITE) {
	log.warn("PUBLIC_COOKIE_SAMESITE is not set, defaulting to 'lax'.");
}
if (!env.PUBLIC_COOKIE_PREFIX) missingEnvVars.push('PUBLIC_COOKIE_PREFIX');

//Private variables
if (!envPrivate.IAM_CLIENT_ID) missingEnvVars.push('IAM_CLIENT_ID');
if (!envPrivate.IAM_URL) missingEnvVars.push('IAM_URL');
if (!envPrivate.ORIGIN) missingEnvVars.push('ORIGIN');
if (!envPrivate.LOGIN_DESTINATION) missingEnvVars.push('LOGIN_DESTINATION');
if (!envPrivate.LOGIN_PAGE) {
	log.warn(`LOGIN_PAGE is not set, defaulting to ${LOGIN_API}`);
}

if (!envPrivate.SDS_URL) {
	log.warn(`SDS_URL is not set, defaulting to basic authentication flow`);
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

		// Store the original URL for potential redirect after authentication
		// This helps preserve the user's intended destination
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

	// 🛡️ Set Security Headers (only if environment variables are set)
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
// matched SvelteKit route (low cardinality) — it runs outermost so the rename
// lands before auth. To tag spans/logs + downstream calls with the tenant,
// pass `teamID`, e.g. once auth has populated locals:
//   sequence(authHandle, createObservabilityHandle({ teamID: (e) => e.locals.authResult?.activeTenant?.id }))
// (no-op when OTEL is disabled — there is simply no active span).
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

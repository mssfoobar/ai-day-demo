import { env as envPrivate } from '$env/dynamic/private';
import {
	randomPKCECodeVerifier,
	calculatePKCECodeChallenge,
	type Configuration,
	buildAuthorizationUrl
} from 'openid-client';
import { StatusCodes } from 'http-status-codes';
import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import {
	COOKIES_TYPE_ENUM,
	DEFAULT_COOKIE_OPTIONS,
	getSdsClient,
	SESSION_DATA_ENUM
} from '$lib/aoh/core/provider/auth/auth';
import { log } from '$lib/aoh/core/logger/Logger';

export const GET: RequestHandler = async (event) => {
	if (!event.locals.clients?.oidc_config) {
		throw new Error(
			'oidc_config missing from locals - required for authentication, please check your configuration for authentication'
		);
	}

	const oidc_config: Configuration = event.locals.clients.oidc_config;

	// Generate PKCE - Proof Key for Code Exchange
	const code_verifier = randomPKCECodeVerifier();
	const code_challenge = await calculatePKCECodeChallenge(code_verifier);

	// Always redirect OAuth callbacks to our safe callback handler
	// This prevents redirect loops by ensuring authentication is fully established
	// before redirecting to private routes
	const finalDestination = envPrivate.ORIGIN + '/aoh/api/auth/callback';

	// Create OIDC Auth URL - where we will redirect the client to for authorization
	const redirectUrl: URL = buildAuthorizationUrl(oidc_config, {
		scope: 'openid',
		resource: finalDestination,
		redirect_uri: finalDestination,
		code_challenge,
		code_challenge_method: 'S256'
	});

	// If SDS is configured, caches code verifier at the SDS server.
	// Otherwise, caches inside the browser cookie
	if (envPrivate.SDS_URL) {
		const sdsClient = await getSdsClient();
		try {
			const tempSid = await sdsClient.tempSessionNew();
			await sdsClient.tempSessionSet(tempSid, SESSION_DATA_ENUM.CODE_VERIFIER, code_verifier);
			event.cookies.set(COOKIES_TYPE_ENUM.TEMP_SESSION_ID, tempSid, DEFAULT_COOKIE_OPTIONS);
		} catch (err) {
			log.error({ err }, 'creating temporary session error');
		} finally {
			await sdsClient.close();
		}
	} else {
		event.cookies.set(COOKIES_TYPE_ENUM.CODE_VERIFIER, code_verifier, DEFAULT_COOKIE_OPTIONS);
	}

	event.setHeaders({
		Location: redirectUrl.href ?? ''
	});

	return json(null, {
		status: StatusCodes.TEMPORARY_REDIRECT
	});
};

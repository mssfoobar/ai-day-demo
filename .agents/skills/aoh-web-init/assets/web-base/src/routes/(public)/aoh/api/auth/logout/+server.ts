import { env } from '$env/dynamic/private';
import { json } from '@sveltejs/kit';
import { StatusCodes } from 'http-status-codes';
import type { RequestHandler } from './$types';
import {
	type Configuration,
	type TokenEndpointResponse,
	refreshTokenGrant,
	buildEndSessionUrl
} from 'openid-client';
import {
	COOKIES_TYPE_ENUM,
	expiryCookieOpts,
	getSdsClient,
	LOGIN_API
} from '$lib/aoh/core/provider/auth/auth';
import { CONTEXT_COOKIE_NAME } from '$lib/aoh/core/constants';
import { log } from '$lib/aoh/core/logger/Logger';

function buildLoginRedirectUrl(): string {
	const origin = env.ORIGIN;
	const loginPage: string = env.LOGIN_PAGE ? env.LOGIN_PAGE : LOGIN_API;
	return origin + ('/' + loginPage).replace('//', '/');
}

async function handleSdsLogout(cookies: Parameters<RequestHandler>[0]['cookies']): Promise<void> {
	const authSessionId: string | undefined = cookies.get(COOKIES_TYPE_ENUM.AUTH_SESSION_ID);
	cookies.delete(COOKIES_TYPE_ENUM.AUTH_SESSION_ID, expiryCookieOpts(0));

	if (authSessionId) {
		const sdsClient = await getSdsClient();

		await sdsClient
			.authSessionDestroy(authSessionId)
			.catch((err: Error) => {
				log.error(err.message);
			})
			.finally(() => {
				sdsClient.close();
			});
	}
}

async function handleOidcLogout(
	cookies: Parameters<RequestHandler>[0]['cookies'],
	locals: Parameters<RequestHandler>[0]['locals'],
	redirectUrl: string
): Promise<Response | null> {
	if (!locals.clients?.oidc_config) {
		throw new Error(
			'oidc_config missing from locals - required for authentication, please check your configuration for authentication'
		);
	}

	const oidc_config: Configuration = locals.clients?.oidc_config;
	const refresh_token: string | undefined = cookies.get(COOKIES_TYPE_ENUM.REFRESH_TOKEN);

	cookies.delete(COOKIES_TYPE_ENUM.REFRESH_TOKEN, expiryCookieOpts(0));
	cookies.delete(COOKIES_TYPE_ENUM.ACCESS_TOKEN, expiryCookieOpts(0));
	cookies.delete(CONTEXT_COOKIE_NAME, expiryCookieOpts(0));

	if (!refresh_token || !oidc_config) {
		return null;
	}

	try {
		const tokens: TokenEndpointResponse = await refreshTokenGrant(oidc_config, refresh_token);

		if (!tokens.id_token) {
			return json(null, { status: StatusCodes.UNAUTHORIZED });
		}

		const endSessionUrl: URL = buildEndSessionUrl(oidc_config, {
			post_logout_redirect_uri: redirectUrl,
			id_token_hint: tokens.id_token
		});

		return json(null, {
			status: StatusCodes.TEMPORARY_REDIRECT,
			headers: { Location: endSessionUrl.toString() }
		});
	} catch (err) {
		log.error(err, 'error refreshing tokens during logout');
	}

	return null;
}

/** Log the user out by deleting all token cookies */
export const GET: RequestHandler = async ({ cookies, locals, setHeaders }) => {
	const redirectUrl = buildLoginRedirectUrl();

	if (env.SDS_URL) {
		await handleSdsLogout(cookies);
	} else {
		const oidcResponse = await handleOidcLogout(cookies, locals, redirectUrl);
		if (oidcResponse) {
			return oidcResponse;
		}
	}

	setHeaders({ Location: redirectUrl });

	return json(null, {
		status: StatusCodes.TEMPORARY_REDIRECT
	});
};

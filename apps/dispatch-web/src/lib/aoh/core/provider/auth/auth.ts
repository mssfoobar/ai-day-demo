import { env as envPublic } from '$env/dynamic/public';
import { env as envPrivate } from '$env/dynamic/private';
import {
	type Configuration,
	authorizationCodeGrant,
	refreshTokenGrant,
	fetchUserInfo,
	type TokenEndpointResponse,
	type TokenEndpointResponseHelpers,
	type IDToken,
	type ResponseBodyError
} from 'openid-client';
import { type Cookies } from '@sveltejs/kit';
import dayjs from 'dayjs';
import Duration from 'dayjs/plugin/duration';
import { log as logger } from '$lib/aoh/core/logger/Logger';
import { jwtDecode } from 'jwt-decode';
import SdsClient from '@mssfoobar/sds-client';

dayjs.extend(Duration);
const log = logger.child({ src: new URL(import.meta.url).pathname });
export const refreshExpiresIn: number = dayjs.duration({ years: 1 }).asSeconds();
export const LOGIN_API: string = '/aoh/api/auth/login';

export type Auth = {
	claims?: AuthClaims;
	authenticated?: boolean;
};

export type AuthClaims = IDToken & {
	all_tenants: Array<Tenant>;
	active_tenant: Tenant;
	realm_access: {
		roles: Array<string>;
	};
};

export type Tenant = {
	tenant_name?: string;
	tenant_id?: string;
	roles?: string[];
};

/**
 * Cookie options, typed against SvelteKit's own `Cookies.set` rather than the `cookie`
 * package.
 *
 * Deliberate: the scaffold imported `SerializeOptions` from `cookie`, which pins this app
 * to whichever major `cookie` it happens to resolve — the name only exists from 0.7, and
 * SvelteKit itself depends on 0.6, where it is `CookieSerializeOptions`. Deriving the type
 * from the function these values are actually passed to cannot skew.
 */
export type CookieOptions = Parameters<Cookies['set']>[2];

export const DEFAULT_COOKIE_OPTIONS: CookieOptions = {
	domain: envPublic.PUBLIC_DOMAIN,
	path: '/',
	secure: (envPrivate.ORIGIN ?? '').toLowerCase().startsWith('https://'),
	httpOnly: true,
	sameSite: (envPublic.PUBLIC_COOKIE_SAMESITE as CookieOptions['sameSite']) ?? 'lax'
};

export const COOKIES_TYPE_ENUM = {
	ACCESS_TOKEN: `${envPublic.PUBLIC_COOKIE_PREFIX}_access_token`,
	REFRESH_TOKEN: `${envPublic.PUBLIC_COOKIE_PREFIX}_refresh_token`,
	CODE_VERIFIER: `${envPublic.PUBLIC_COOKIE_PREFIX}_code_verifier`,
	TEMP_SESSION_ID: `${envPublic.PUBLIC_COOKIE_PREFIX}_temp_session_id`,
	AUTH_SESSION_ID: `${envPublic.PUBLIC_COOKIE_PREFIX}_auth_session_id`
};

export const SESSION_DATA_ENUM = {
	CODE_VERIFIER: 'code_verifier'
};

export const getSdsClient = async (): Promise<SdsClient> => {
	const client = new SdsClient(envPrivate.SDS_URL);
	client.on('error', (err) => {
		log.error({ err }, 'SDS Client error');
	});
	await client.connect();

	return client;
};

/**
 * Returns cookie options with the specified expiry time.
 *
 * @param {number} expiry - The maximum age of the cookie in seconds.
 * @return {CookieOptions} Cookie options with the specified expiry time.
 */
export const expiryCookieOpts = (expiry: number = 0): CookieOptions => {
	return { ...DEFAULT_COOKIE_OPTIONS, maxAge: expiry };
};

/**
 * Checks if the user is redirected from the issuer after an authorization request.
 *
 * @param {URL} url - The URL of the current page
 * @return {boolean} - True if the user was redirected from the issuer, false otherwise
 */
export function isUserRedirectedFromIssuer(url: URL): boolean {
	// Check if the request was redirected and the URL contains a session state and a code
	// This indicates that the user was redirected from the issuer after an authorization request
	return Boolean(url.searchParams.has('session_state') && url.searchParams.has('code'));
}

/**
 * Refresh the tokens by calling the refresh endpoint with the refresh token.
 * @param {Configuration} oidc_config - The OpenID client configuration
 * @param {string} refreshToken - The refresh token to use for refreshing the tokens
 * @returns {Promise<TokenEndpointResponse & TokenEndpointResponseHelpers | undefined>} - The newly refreshed tokens, or undefined if the refresh failed
 */
async function refresh(
	oidc_config: Configuration,
	refreshToken: string
): Promise<(TokenEndpointResponse & TokenEndpointResponseHelpers) | undefined> {
	log.debug('attempting to refresh tokens...');

	try {
		return await refreshTokenGrant(oidc_config, refreshToken);
	} catch (error) {
		log.error(error);
		// If the refresh failed, return undefined
		return undefined;
	}
}

/**
 * Validate the access token by calling the userinfo endpoint with the access token.
 * @param {Configuration} oidc_config - The OpenID client instance
 * @param {string} accessToken - The access token to validate
 * @returns {Promise<boolean>} - Whether the access token is valid
 */
export async function validateAccessToken(
	oidc_config: Configuration,
	accessToken: string
): Promise<boolean> {
	try {
		const claims = jwtDecode(accessToken);

		if (!claims.sub) {
			throw new Error('no sub in access token');
		}

		// Call the userinfo endpoint with the access token
		await fetchUserInfo(oidc_config, accessToken, claims.sub);

		return true;
	} catch (error) {
		log.error({ error }, 'invalid access token detected');
		return false;
	}
}

export type AuthResultSuccess = {
	success: true;
	claims: AuthClaims;
	access_token?: string;
};

export type AuthResultFail = {
	success: false;
};

export type AuthResult = AuthResultSuccess | AuthResultFail;

/**
 * Read JWT claims off an AuthResult without forcing every caller to re-narrow
 * the discriminated union. Returns null on the fail branch.
 */
export function getClaims(authResult: AuthResult): AuthClaims | null {
	return authResult.success ? authResult.claims : null;
}

/**
 * Read the active tenant's roles off an AuthResult. Returns [] on the fail
 * branch OR when the success branch's JWT has no active_tenant.roles claim,
 * so callers can just do `getRoles(locals.authResult).includes("field-reporter")`.
 */
export function getRoles(authResult: AuthResult): string[] {
	return authResult.success ? (authResult.claims.active_tenant?.roles ?? []) : [];
}

/**
 * @description
 * Authenticates the user by validating access token and refresh token, and getting a new access token
 * using the refresh token if the access token is invalid. If the user is redirected from the issuer
 * (e.g. Keycloak), it will also handle the callback from the issuer. If the sdsClient is provided,
 * it will store the tokens at SDS backend server instead of storing in browser cookies.
 *
 * @param {Configuration} openid_config - The OAuth 2.0 client configuration
 * @param {Cookies} cookies - The cookies object
 * @param {URL} url - The URL object
 * @param {SdsClient} [sdsClient] - The optional sdsClient object
 *
 * @returns {Promise<boolean>} - Whether the authentication is successful or not
 */
export async function authenticate(
	openid_config: Configuration,
	cookies: Cookies,
	url: URL,
	sdsClient?: SdsClient
): Promise<AuthResult> {
	// If sdsClient is provided, use the SDS authentication flow to store tokens at SDS server
	// Otherwise, use the basic authentication flow that stores tokens at cookies
	if (sdsClient) {
		return sdsAuthenticationFlow(openid_config, cookies, url, sdsClient);
	} else {
		return basicAuthenticationFlow(openid_config, cookies, url);
	}
}

async function basicAuthenticationFlow(
	openid_config: Configuration,
	cookies: Cookies,
	url: URL
): Promise<AuthResult> {
	// Get the access token and refresh token from the cookies
	const accessToken: string | undefined = cookies.get(COOKIES_TYPE_ENUM.ACCESS_TOKEN);
	const refreshToken: string | undefined = cookies.get(COOKIES_TYPE_ENUM.REFRESH_TOKEN);
	const code_verifier: string | undefined = cookies.get(COOKIES_TYPE_ENUM.CODE_VERIFIER);

	if (accessToken && refreshToken) {
		// === User has both access and refresh tokens, validate the them
		const isAccessTokenValid: boolean = await validateAccessToken(openid_config, accessToken);

		if (isAccessTokenValid) {
			// If the access token is valid, call the onAuthenticationSuccess callback
			return onAuthenticationSuccess(cookies, accessToken);
		} else {
			// If the access token is invalid, refresh the tokens and call the onAuthenticationSuccess callback
			return await refreshTokens(openid_config, refreshToken, cookies);
		}
	} else if (refreshToken) {
		// === No access token but have refresh token
		// If the user has a refresh token but no access token, refresh the tokens and call the onAuthenticationSuccess callback
		return await refreshTokens(openid_config, refreshToken, cookies);
	} else if (isUserRedirectedFromIssuer(url)) {
		// === If the user is redirected from the issuer, handle the callback

		const originatingUrl = new URL(envPrivate.ORIGIN + url.pathname + url.search);

		try {
			// either new URL(url.href) or new URL(destination)
			const tokenSet: TokenEndpointResponse & TokenEndpointResponseHelpers =
				await authorizationCodeGrant(openid_config, originatingUrl, {
					pkceCodeVerifier: code_verifier
				});

			return onAuthenticationSuccess(cookies, tokenSet.access_token, {
				refresh_token: tokenSet.refresh_token!,
				access_token_expires_in: tokenSet.expires_in!
			});
		} catch (err) {
			log.error({ err }, 'error retrieving tokens from issuer');

			if ((err as ResponseBodyError).error === 'invalid_grant') {
				log.debug(`Invalid grant. This often happens with cookies are not being forwarded.
					Please check that the code is correct.

					Other possible reasons why the code is wrong:
					- Your PUBLIC_DOMAIN env var might be wrong.
					- Access token client login session timeout too large (Keycloak overflow bug).`);
			} else if ((err as ResponseBodyError).error === 'unauthorized_client') {
				// Prevent infinite redirect-loop - misconfiguration detected
				throw new Error(
					'Invalid client. Please check that the client_id and client_secret are correct.',
					{ cause: err }
				);
			}

			return onAuthenticationFailed(cookies);
		}
	}

	return onAuthenticationFailed(cookies);
}

async function sdsAuthenticationFlow(
	openid_config: Configuration,
	cookies: Cookies,
	url: URL,
	sdsClient: SdsClient
): Promise<AuthResult> {
	const authSessionId: string | undefined = cookies.get(COOKIES_TYPE_ENUM.AUTH_SESSION_ID);

	if (authSessionId) {
		// if there is authSessionId in the cookie, retrieve accessToken from SMM
		try {
			const accessToken = await sdsClient.authSessionGetAccessToken(authSessionId);
			return onAuthenticationSuccess(cookies, accessToken, { authSessionId: authSessionId });
		} catch (err) {
			log.warn({ err }, 'invalid access token. authentication failed');
			return onAuthenticationFailed(cookies);
		}
	}

	if (isUserRedirectedFromIssuer(url)) {
		// === If the user is redirected from the issuer, handle the callback

		const originatingUrl = new URL(envPrivate.ORIGIN + url.pathname + url.search);

		const tempSessionId = cookies.get(COOKIES_TYPE_ENUM.TEMP_SESSION_ID);
		if (!tempSessionId) {
			log.warn({ err: 'no temp session id cookie found' }, 'unable to retrieve code verifier');
			return onAuthenticationFailed(cookies);
		}

		let code_verifier: string;
		try {
			code_verifier = await sdsClient.tempSessionGet(
				tempSessionId,
				SESSION_DATA_ENUM.CODE_VERIFIER
			);
		} catch (err) {
			log.warn({ err }, 'unable to retrieve code verifier');
			return onAuthenticationFailed(cookies);
		}

		try {
			// either new URL(url.href) or new URL(destination)
			const tokenSet: TokenEndpointResponse & TokenEndpointResponseHelpers =
				await authorizationCodeGrant(openid_config, originatingUrl, {
					pkceCodeVerifier: code_verifier
				});

			const newAuthSessionId = await sdsClient.authSessionNew(
				tokenSet.access_token,
				tokenSet.refresh_token!
			);

			return onAuthenticationSuccess(cookies, tokenSet.access_token, {
				authSessionId: newAuthSessionId
			});
		} catch (err) {
			log.error({ err }, 'error retrieving tokens from issuer');

			if ((err as ResponseBodyError).error === 'invalid_grant') {
				log.debug(`Invalid grant. This often happens with cookies are not being forwarded.
					Please check that the code is correct.

					Other possible reasons why the code is wrong:
					- Your PUBLIC_DOMAIN env var might be wrong.
					- Access token client login session timeout too large (Keycloak overflow bug).`);
			} else if ((err as ResponseBodyError).error === 'unauthorized_client') {
				// Prevent infinite redirect-loop - misconfiguration detected
				throw new Error(
					'Invalid client. Please check that the client_id and client_secret are correct.',
					{ cause: err }
				);
			}

			return onAuthenticationFailed(cookies);
		}
	}

	return onAuthenticationFailed(cookies);
}

async function refreshTokens(
	openid_config: Configuration,
	refresh_token: string,
	cookies: Cookies
): Promise<AuthResult> {
	const newTokenSet: (TokenEndpointResponse & TokenEndpointResponseHelpers) | undefined =
		await refresh(openid_config, refresh_token);

	if (newTokenSet) {
		log.debug('Tokens refreshed successfully');
		return onAuthenticationSuccess(cookies, newTokenSet.access_token, {
			refresh_token: newTokenSet.refresh_token,
			access_token_expires_in: newTokenSet.expires_in!
		});
	} else {
		log.debug('Tokens refresh failed');
		return onAuthenticationFailed(cookies);
	}
}

/**
 * Handles the authentication success by setting the access and refresh tokens
 * in the user's cookies. If refresh_token and expires_in are provided, those
 * cookies will also be updated.
 *
 * @param cookies - SvelteKit cookies interface to set or delete cookies
 * @param access_token - The access token
 * @param options - Optional parameters
 */
export function onAuthenticationSuccess(
	cookies: Cookies,
	access_token: string,
	options?: {
		authSessionId?: string;
		refresh_token?: string;
		access_token_expires_in?: number;
	}
): AuthResultSuccess {
	// If authSessionId is provided, set only AUTH_SESSION_ID cookie and return
	// Otherwise, set OIDC auth flow related cookies
	if (options?.authSessionId) {
		cookies.set(COOKIES_TYPE_ENUM.AUTH_SESSION_ID, options.authSessionId, DEFAULT_COOKIE_OPTIONS);
		cookies.delete(COOKIES_TYPE_ENUM.TEMP_SESSION_ID, expiryCookieOpts());
	} else {
		// Set cookies
		if (options?.access_token_expires_in) {
			cookies.set(
				COOKIES_TYPE_ENUM.ACCESS_TOKEN,
				access_token,
				expiryCookieOpts(options.access_token_expires_in)
			);
		}

		if (options?.refresh_token) {
			cookies.set(
				COOKIES_TYPE_ENUM.REFRESH_TOKEN,
				options.refresh_token,
				expiryCookieOpts(refreshExpiresIn)
			);
		}

		cookies.delete(COOKIES_TYPE_ENUM.CODE_VERIFIER, expiryCookieOpts());
	}

	return {
		success: true,
		claims: jwtDecode(access_token),
		access_token: access_token
	};
}

/**
 * Handles the authentication failure by deleting the access, refresh and code verifier
 * tokens from the user's cookies.
 *
 * @param cookies - SvelteKit cookies interface to delete cookies
 */
export function onAuthenticationFailed(cookies: Cookies): AuthResultFail {
	// Delete SDS related cookies
	cookies.delete(COOKIES_TYPE_ENUM.AUTH_SESSION_ID, expiryCookieOpts());
	cookies.delete(COOKIES_TYPE_ENUM.TEMP_SESSION_ID, expiryCookieOpts());

	// Delete the access token
	cookies.delete(COOKIES_TYPE_ENUM.ACCESS_TOKEN, expiryCookieOpts());
	// Delete the refresh token
	cookies.delete(COOKIES_TYPE_ENUM.REFRESH_TOKEN, expiryCookieOpts());
	// Delete the code verifier
	cookies.delete(COOKIES_TYPE_ENUM.CODE_VERIFIER, expiryCookieOpts());

	return {
		success: false
	};
}

/**
 * Checks if the user is a tenant admin by decoding the access token and checking
 * if the 'tenant-admin' role exists in the active tenant's roles.
 *
 * @param {string} accessToken - The JWT access token to decode
 * @returns {boolean} - True if the user has the 'tenant-admin' role in their active tenant, false otherwise
 */
export function isTenantAdmin(accessToken: string): boolean {
	try {
		const claims = jwtDecode<AuthClaims>(accessToken);
		// Check if the user has the 'tenant-admin' role in their active tenant
		return claims.active_tenant?.roles?.includes('tenant-admin') ?? false;
	} catch (error) {
		log.error({ error }, 'failed to decode access token');
		return false;
	}
}

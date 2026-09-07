import { env } from "$env/dynamic/private";
import { json } from "@sveltejs/kit";
import { StatusCodes } from "http-status-codes";
import type { RequestHandler } from "./$types";
import { type Configuration, type TokenEndpointResponse, refreshTokenGrant } from "openid-client";
import { COOKIES_TYPE_ENUM, expiryCookieOpts, refreshExpiresIn, getSdsClient } from "$lib/aoh/core/provider/auth/auth";
import { log } from "$lib/aoh/core/logger/Logger";

export const GET: RequestHandler = async ({ cookies, locals }) => {
	if (env.SDS_URL) {
		const authSessionId: string | undefined = cookies.get(COOKIES_TYPE_ENUM.AUTH_SESSION_ID);
		if (authSessionId) {
			const sdsClient = await getSdsClient();
			try {
				// accessing the token refresh it if necessary
				await sdsClient.authSessionGetAccessToken(authSessionId);
			} catch (e) {
				log.error({ err: e }, "error refreshing tokens");
				cookies.delete(COOKIES_TYPE_ENUM.AUTH_SESSION_ID, expiryCookieOpts());
				const responseBody = {
					message: "Unable to refresh tokens, deleting tokens cookies.",
					sent_at: new Date().toISOString(),
				};

				return json(responseBody, {
					status: StatusCodes.UNAUTHORIZED,
				});
			} finally {
				await sdsClient.close();
			}

			return json(null, {
				status: StatusCodes.OK,
			});
		}
	} else {
		const oidc_config: Configuration | undefined = locals.clients?.oidc_config;
		const refresh_token: string | undefined = cookies.get(COOKIES_TYPE_ENUM.REFRESH_TOKEN);

		if (refresh_token && oidc_config) {
			try {
				const tokens: TokenEndpointResponse = await refreshTokenGrant(oidc_config, refresh_token);
				cookies.set(COOKIES_TYPE_ENUM.ACCESS_TOKEN, tokens.access_token, expiryCookieOpts(tokens.expires_in));
				cookies.set(
					COOKIES_TYPE_ENUM.REFRESH_TOKEN,
					tokens.refresh_token as string,
					expiryCookieOpts(refreshExpiresIn)
				);
				cookies.delete(COOKIES_TYPE_ENUM.CODE_VERIFIER, expiryCookieOpts());
			} catch (e) {
				log.error({ err: e }, "error refreshing tokens");
				cookies.delete(COOKIES_TYPE_ENUM.ACCESS_TOKEN, expiryCookieOpts());
				cookies.delete(COOKIES_TYPE_ENUM.REFRESH_TOKEN, expiryCookieOpts());
				const responseBody = {
					message: "Unable to refresh tokens, deleting tokens cookies.",
					sent_at: new Date().toISOString(),
				};

				return json(responseBody, {
					status: StatusCodes.UNAUTHORIZED,
				});
			}

			return json(null, {
				status: StatusCodes.OK,
			});
		}
	}

	const responseBody = {
		message: "Unable to refresh tokens, deleting tokens cookies.",
		sent_at: new Date().toISOString(),
	};

	return json(responseBody, {
		status: StatusCodes.UNAUTHORIZED,
	});
};

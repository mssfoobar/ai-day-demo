import { env } from "$env/dynamic/private";
import { json } from "@sveltejs/kit";
import { StatusCodes } from "http-status-codes";
import { CONTEXT_COOKIE_MAX_AGE, CONTEXT_COOKIE_NAME } from "$lib/aoh/core/constants";
import type { RequestHandler } from "./$types";
import { expiryCookieOpts } from "$lib/aoh/core/provider/auth/auth";

export const GET: RequestHandler = async ({ cookies, setHeaders, params }) => {
	const contextValue = params.value;
	const origin = env.ORIGIN ?? "";

	cookies.set(CONTEXT_COOKIE_NAME, contextValue, expiryCookieOpts(CONTEXT_COOKIE_MAX_AGE));
	setHeaders({
		Location: origin,
	});

	return json(null, {
		status: StatusCodes.TEMPORARY_REDIRECT,
	});
};

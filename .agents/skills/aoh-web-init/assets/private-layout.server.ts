/** @type {import('./$types').LayoutLoad} */
import { log as logger } from "$lib/aoh/core/logger/Logger.js";
import { LOGIN_API } from "$lib/aoh/core/provider/auth/auth";
import { redirect } from "@sveltejs/kit";
import dayjs from "dayjs";
import Duration from "dayjs/plugin/duration.js";
import { StatusCodes } from "http-status-codes";
import { CONTEXT_COOKIE_NAME } from "$lib/aoh/core/constants";
import { env as envPrivate } from "$env/dynamic/private";
dayjs.extend(Duration);

const log = logger.child({ src: new URL(import.meta.url).pathname });

export async function load({ locals, cookies, url }) {
	//INFO: Then based on its authenticated status, decide what to do. As this route is protected, we will
	//redirect the user to the login page if they are not authenticated.

	const authResult = locals.authResult;
	const context = cookies.get(CONTEXT_COOKIE_NAME);

	if (authResult.success) {
		log.debug("User is authenticated");

		return {
			user: authResult.claims,
			context,
		};
	} else {
		log.debug("User is not authenticated");
		// Avoid redirect loops by checking if we're already dealing with auth-related URLs
		if (url.pathname.startsWith("/aoh/api/auth/")) {
			log.warn("Attempted to redirect from auth endpoint, potential redirect loop");
			throw new Error("Authentication required");
		}

		// Store the intended destination in a cookie to avoid redirect loops
		const currentPath = url.pathname + url.search;
		cookies.set("aoh_redirect_after_auth", currentPath, {
			path: "/",
			maxAge: 60 * 10, // 10 minutes
			httpOnly: true,
			sameSite: "lax",
			secure: (envPrivate.ORIGIN ?? "").toLowerCase().startsWith("https://"),
		});
		redirect(StatusCodes.TEMPORARY_REDIRECT, LOGIN_API);
	}
}

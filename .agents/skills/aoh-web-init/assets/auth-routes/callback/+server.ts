import { redirect } from "@sveltejs/kit";
import { StatusCodes } from "http-status-codes";
import type { RequestHandler } from "./$types";
import { log as logger } from "$lib/aoh/core/logger/Logger";
import { env as envPrivate } from "$env/dynamic/private";

const log = logger.child({ src: new URL(import.meta.url).pathname });

export const GET: RequestHandler = async ({ cookies }) => {
	// This route handles the final redirect after successful authentication
	// It ensures the user is redirected to their intended destination

	// Check if there's a stored redirect destination
	const redirectDestination = cookies.get("aoh_redirect_after_auth");

	if (redirectDestination) {
		log.debug(`Redirecting user to stored destination: ${redirectDestination}`);
		// Clear the redirect cookie
		cookies.delete("aoh_redirect_after_auth", { path: "/" });
		redirect(StatusCodes.TEMPORARY_REDIRECT, redirectDestination);
	}

	// If no stored destination, redirect to the default landing page
	const defaultDestination = ("/" + envPrivate.LOGIN_DESTINATION).replace("//", "/");
	log.debug(`Redirecting user to default destination: ${defaultDestination}`);
	redirect(StatusCodes.TEMPORARY_REDIRECT, defaultDestination);
};

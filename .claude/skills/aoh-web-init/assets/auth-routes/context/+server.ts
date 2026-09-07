import type { RequestHandler } from "@sveltejs/kit";
import { CONTEXT_COOKIE_NAME } from "$lib/aoh/core/constants";

export const GET: RequestHandler = async ({ cookies }) => {
	const currentContext = cookies.get(CONTEXT_COOKIE_NAME);

	return new Response(JSON.stringify({ context: currentContext }), {
		status: 200,
		headers: {
			"Content-Type": "application/json",
		},
	});
};

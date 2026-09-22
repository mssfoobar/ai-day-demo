<svelte:options runes={true} />

<script lang="ts" module>
	import { get, writable, type Writable } from 'svelte/store';
	import type { Auth } from '$lib/aoh/core/provider/auth/auth';

	const authStore: Writable<Auth> = writable({});

	/** The signed-in operator's claims, for code outside the component tree. */
	export function getAuth(): Auth {
		return get(authStore);
	}
</script>

<script lang="ts">
	import type { AuthClaims } from '$lib/aoh/core/provider/auth/auth';

	// Claims only — never a token. The access token stays in SDS and is read
	// server-side; putting it here would serialise it into the page payload, which
	// is exactly what the SDS posture exists to prevent.
	let { claims, children }: { claims: AuthClaims | undefined; children: import('svelte').Snippet } =
		$props();

	$effect(() => {
		authStore.set(claims ? { claims, authenticated: true } : {});
	});
</script>

{@render children()}

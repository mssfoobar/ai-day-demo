<script lang="ts" context="module">
	let authStore: Writable<Auth> = writable({});
	export function getAuth(): Auth {
		return get(authStore);
	}
</script>

<script lang="ts">
	import type { Auth, AuthClaims } from '$lib/aoh/core/provider/auth/auth';
	import { get, type Writable, writable } from 'svelte/store';

	export let claims: AuthClaims | undefined;
	$: if (claims) {
		authStore.set({
			claims: claims,
			authenticated: true
		});
	}
</script>

<slot />

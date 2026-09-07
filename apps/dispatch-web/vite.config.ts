import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';

export default defineConfig({
	// No `allowedHosts`: the scaffold allowlisted `127.0.0.1.nip.io` so Keycloak cookie
	// domains and OIDC redirect URIs would resolve against a real hostname. This app has
	// no authentication, so it is served on plain localhost and the allowlist is moot.
	plugins: [tailwindcss(), sveltekit()],
	build: {
		outDir: 'build'
	}
});

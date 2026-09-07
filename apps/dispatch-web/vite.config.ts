import { defineConfig, configDefaults } from 'vitest/config';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';

export default defineConfig({
	// No `allowedHosts`: the scaffold allowlisted `127.0.0.1.nip.io` so Keycloak cookie
	// domains and OIDC redirect URIs would resolve against a real hostname. This app has
	// no authentication, so it is served on plain localhost and the allowlist is moot.
	plugins: [tailwindcss(), sveltekit()],
	build: {
		outDir: 'build'
	},
	test: {
		expect: { requireAssertions: true },
		include: ['src/**/*.{test,spec}.{js,ts}'],
		exclude: [...configDefaults.exclude],
		passWithNoTests: true,
		coverage: {
			// An explicit `include` is what makes untested files appear in the report. Without
			// it, coverage covers only the modules a test happened to import, so an untested
			// file reads as "100%" by being absent. (vitest 4 dropped the old `all` flag —
			// `include` now carries that meaning.)
			include: ['src/lib/**/*.{ts,svelte}'],
			exclude: ['src/lib/**/*.{test,spec}.{js,ts}']
		}
	}
});

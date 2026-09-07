import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter(),
		alias: {
			$root: '.'
		},
		// Enables src/instrumentation.server.ts (OpenTelemetry bootstrap).
		// Requires SvelteKit >= 2.31 + @sveltejs/adapter-node >= 5.3.0.
		experimental: {
			instrumentation: {
				server: true
			}
		}
	}
};

export default config;

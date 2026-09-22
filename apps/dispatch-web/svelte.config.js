import path from 'node:path';
import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { transformWithOxc } from 'vite';

/**
 * TypeScript stripping for `<script lang="ts">`, including in dependencies.
 *
 * Two things make this necessary rather than `vitePreprocess({ script: true })`:
 *
 *  1. **Svelte's own type stripping is not enough.** It handles plain annotations but not
 *     optional parameters, so a dependency shipping `.svelte` with `fn(a?: T)` — which
 *     `@mssfoobar/gis-web-sdk` does — reaches the bundler as unparseable JS and the build
 *     fails deep inside node_modules with no obvious cause.
 *
 *  2. **`vitePreprocess({ script: true })` then breaks those same files a second way.** It
 *     transforms each `<script>` block as if it were a standalone module, with no
 *     `onlyRemoveTypeImports`, so oxc drops any import the block does not itself use. The
 *     SDK's `GisProvider` imports `setContext` in its `<script module>` and uses it in the
 *     instance script — a perfectly legal Svelte 5 arrangement — and the import is
 *     silently deleted. The failure surfaces at runtime, during SSR, as
 *     `ReferenceError: setContext is not defined` while rendering an error page.
 *     `vite-plugin-svelte` carries a TODO about exactly this gap.
 *
 * `onlyRemoveTypeImports: true` removes only imports explicitly marked `type`, which is
 * what both this app and the SDK are written against.
 */
const typescriptScript = {
	name: 'dispatch-typescript-script',
	/** @type {import('svelte/compiler').Preprocessor} */
	async script({ attributes, content, filename = '' }) {
		if (attributes.lang !== 'ts') return;

		const { code, map } = await transformWithOxc(content, filename, {
			lang: 'ts',
			target: 'esnext',
			typescript: { onlyRemoveTypeImports: true }
		});

		// Sourcemap sources arrive absolute; Svelte expects them relative to the file, the
		// same normalisation vitePreprocess does before handing the map back.
		if (map?.sources) {
			const dir = path.dirname(filename);
			map.sources = map.sources.map((source) =>
				source ? path.relative(dir, source.replace(/^file:\/{2,3}/, '')) || source : source
			);
		}

		return { code, map };
	}
};

const config = {
	// Order matters: the TypeScript pass runs before vitePreprocess, which here handles
	// only `<style>` (its script half is deliberately not enabled — see above).
	preprocess: [typescriptScript, vitePreprocess()],
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

import { cpSync, existsSync, mkdirSync } from 'node:fs';
import { createRequire } from 'node:module';
import { dirname, join } from 'node:path';
import { defineConfig, type Plugin } from 'vite';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';

const require = createRequire(import.meta.url);

/** Where the copied assets live, and the base URL the engine fetches them from. */
const CESIUM_STATIC_DIR = 'static/cesium';
const CESIUM_BASE_URL = '/cesium';

/**
 * Cesium fetches its Web Workers, textures and third-party shims at *runtime*, from
 * `CESIUM_BASE_URL`. They are not part of the module graph, so no bundler pulls them in:
 * without this copy the canvas renders entirely black with only 404s under `/cesium/` in
 * the console to explain it.
 *
 * It runs on both `buildStart` and `configureServer` so `pnpm build` and `pnpm dev` are
 * both covered — a plugin hooked only to the build leaves dev black.
 */
function cesiumAssets(): Plugin {
	const copy = () => {
		const engine = dirname(require.resolve('@cesium/engine/package.json'));
		const widgets = dirname(require.resolve('@cesium/widgets/package.json'));
		const sources: [string, string][] = [
			[join(engine, 'Build/Workers'), join(CESIUM_STATIC_DIR, 'Workers')],
			[join(engine, 'Build/ThirdParty'), join(CESIUM_STATIC_DIR, 'ThirdParty')],
			[join(engine, 'Source/Assets'), join(CESIUM_STATIC_DIR, 'Assets')],
			[join(engine, 'Source/ThirdParty'), join(CESIUM_STATIC_DIR, 'ThirdParty')],
			[join(widgets, 'Source'), join(CESIUM_STATIC_DIR, 'Widgets')]
		];

		mkdirSync(CESIUM_STATIC_DIR, { recursive: true });
		for (const [from, to] of sources) {
			// Tolerated rather than fatal: Cesium has moved these between majors, and a
			// missing optional directory must not break every build.
			if (existsSync(from)) cpSync(from, to, { recursive: true, force: true, dereference: true });
		}
	};

	return {
		name: 'dispatch-cesium-assets',
		buildStart: copy,
		configureServer: copy
	};
}

export default defineConfig({
	plugins: [tailwindcss(), sveltekit(), cesiumAssets()],
	define: {
		// Read by the engine at module init to resolve its runtime assets.
		CESIUM_BASE_URL: JSON.stringify(CESIUM_BASE_URL)
	},
	ssr: {
		// These ship untranspiled Svelte / ESM the Node build cannot consume as-is.
		noExternal: [/^@mssfoobar\//, '@lucide/svelte', '@cesium/engine', '@cesium/widgets']
	},
	server: {
		// Restored from the scaffold. The console is served on the platform's dev origin
		// (`http://127.0.0.1.nip.io:5173`), not plain localhost, because that is the only
		// origin whose cookie reaches `rtus-seh.${DEV_DOMAIN}` and that rtus-seh's CORS
		// list already permits. `pnpm dev` runs `vite dev --host`, so without this the dev
		// server host-checks the request and refuses it.
		allowedHosts: ['127.0.0.1.nip.io', '.127.0.0.1.nip.io']
	},
	preview: {
		// `pnpm preview` reads the same .env.development, so it needs the same allowlist.
		allowedHosts: ['127.0.0.1.nip.io', '.127.0.0.1.nip.io']
	},
	build: {
		outDir: 'build'
	}
});

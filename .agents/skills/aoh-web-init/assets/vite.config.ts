import { defineConfig, configDefaults } from 'vitest/config';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		allowedHosts: ['127.0.0.1.nip.io', '.127.0.0.1.nip.io']
	},
	preview: {
		allowedHosts: ['127.0.0.1.nip.io', '.127.0.0.1.nip.io']
	},
	build: {
		outDir: 'build'
	},
	test: {
		expect: { requireAssertions: true },
		include: ['src/**/*.{test,spec}.{js,ts}'],
		exclude: [...configDefaults.exclude],
		passWithNoTests: true
	}
});

<svelte:options runes={true} />

<script lang="ts">
	import '/src/app.css';
	import { Toaster } from '@mssfoobar/ui/toast';
	// Default export: the subpath resolves to the component's own `.svelte` file.
	import GisProvider from '@mssfoobar/gis-web-sdk/provider';
	import ThemeProvider from '$lib/aoh/core/provider/theme/ThemeProvider/index.svelte';
	import { darkModeStore } from '$lib/aoh/core/provider/theme/ThemeProvider/theme';

	let { children } = $props();
</script>

<ThemeProvider mode="dark">
	<!--
		GisProvider is the SDK's theme provider and its only prop is the host's dark-mode
		store, so the map repaints with the rest of the console. It is mounted once, at the
		root, because the SDK expects it high in the tree — not per page. `darkModeStore` is
		kept in sync by ThemeProvider's applyThemeClass.
	-->
	<GisProvider dark_mode_store={darkModeStore}>
		{@render children()}
	</GisProvider>
	<!--
		Toaster is NOT auto-mounted by the package: without this, every `toast.*()` call is
		a silent no-op and a write just looks like nothing happened (aoh-conventions/web.md).
	-->
	<Toaster richColors closeButton />
</ThemeProvider>

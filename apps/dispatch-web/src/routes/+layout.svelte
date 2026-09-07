<svelte:options runes={true} />

<script lang="ts">
	import '/src/app.css';
	import { Toaster } from '@mssfoobar/ui/toast';
	import ThemeProvider from '$lib/aoh/core/provider/theme/ThemeProvider/index.svelte';

	let { children } = $props();
</script>

<ThemeProvider mode="dark">
	{@render children()}
	<!--
		Toaster is NOT auto-mounted by the package: without this, every `toast.*()` call is
		a silent no-op and a write just looks like nothing happened (aoh-conventions/web.md).
	-->
	<Toaster richColors closeButton />
</ThemeProvider>

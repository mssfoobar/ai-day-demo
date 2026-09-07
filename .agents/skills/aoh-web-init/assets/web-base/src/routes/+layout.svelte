<svelte:options runes={true} />

<script lang="ts">
	import '/src/app.css';
	import ThemeProvider from '$lib/aoh/core/provider/theme/ThemeProvider/index.svelte';

	let { children } = $props();
</script>

<ThemeProvider mode="dark">
	{@render children()}
</ThemeProvider>

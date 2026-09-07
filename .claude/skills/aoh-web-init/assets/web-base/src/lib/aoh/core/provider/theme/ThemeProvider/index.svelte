<script lang="ts">
	import { onMount } from 'svelte';
	import { applyThemeClass, setStoredTheme, type ThemeMode } from './theme';

	let { mode = 'dark', children }: { mode?: ThemeMode; children?: import('svelte').Snippet } =
		$props();

	onMount(() => {
		applyThemeClass(mode);
		setStoredTheme(mode);
	});

	$effect(() => {
		applyThemeClass(mode);
	});
</script>

{@render children?.()}

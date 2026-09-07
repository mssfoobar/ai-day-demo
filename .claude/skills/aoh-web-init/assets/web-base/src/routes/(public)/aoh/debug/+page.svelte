<svelte:options runes={true} />

<script lang="ts">
	import { env } from '$env/dynamic/public';
</script>

<section class="mx-auto max-w-3xl space-y-4 p-8">
	<h1 class="text-2xl font-semibold">Debug</h1>
	<pre class="rounded-md border p-4 text-sm">{JSON.stringify(env, null, 2)}</pre>
</section>

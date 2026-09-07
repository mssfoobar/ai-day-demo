<script lang="ts">
	type Props = {
		data: number[];
		labels: string[];
		color?: string;
		height?: number;
	};

	let {
		data,
		labels,
		color = "var(--bg-info-strong)",
		height = 180,
	}: Props = $props();

	const max = $derived(Math.max(...data, 1));
</script>

<div
	class="relative flex items-end gap-3"
	style="height: {height}px; padding: 8px 0 24px;"
>
	{#each data as v, i (i)}
		<div class="flex flex-1 flex-col items-center gap-1.5">
			<div
				class="w-full max-w-10 self-center rounded-t-md"
				style="height: {(v / max) * (height - 36)}px; background-color: {color}; opacity: {0.85 +
					(i / data.length) * 0.15};"
			></div>
			<span class="text-(--text-muted) text-[11px] font-medium">{labels[i] ?? ""}</span>
		</div>
	{/each}
</div>

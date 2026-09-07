<script lang="ts">
	type Props = {
		data: number[];
		color?: string;
		height?: number;
		gridLines?: number;
	};

	let {
		data,
		color = "var(--bg-info-strong)",
		height = 180,
		gridLines = 4,
	}: Props = $props();

	const width = 600;
	const max = $derived(Math.max(...data, 1));
	const stepX = $derived(width / Math.max(data.length - 1, 1));
	const points = $derived(
		data.map((v, i) => [i * stepX, height - (v / max) * (height - 24) - 12] as const),
	);
	const path = $derived(
		points
			.map((p, i) => `${i ? "L" : "M"}${p[0].toFixed(1)},${p[1].toFixed(1)}`)
			.join(" "),
	);
	const area = $derived(`${path} L${width},${height} L0,${height} Z`);
	const gradientId = `ln-grad-${Math.random().toString(36).slice(2, 8)}`;
</script>

<svg
	viewBox="0 0 {width} {height}"
	width="100%"
	height={height}
	preserveAspectRatio="none"
	role="img"
	aria-label="Line chart"
>
	<defs>
		<linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
			<stop offset="0%" stop-color={color} stop-opacity="0.25" />
			<stop offset="100%" stop-color={color} stop-opacity="0" />
		</linearGradient>
	</defs>
	{#each Array.from({ length: gridLines }, (_, i) => (i + 1) / (gridLines + 1)) as p (p)}
		<line
			x1="0"
			y1={height * (1 - p)}
			x2={width}
			y2={height * (1 - p)}
			stroke="var(--border)"
			stroke-width="1"
		/>
	{/each}
	<path d={area} fill="url(#{gradientId})" />
	<path
		d={path}
		stroke={color}
		stroke-width="2"
		fill="none"
		stroke-linejoin="round"
		stroke-linecap="round"
	/>
</svg>

<script lang="ts">
	type Segment = { name: string; value: number; color: string };

	type Props = {
		data: Segment[];
		size?: number;
		total?: number;
		totalLabel?: string;
	};

	let { data, size = 160, total, totalLabel = "total" }: Props = $props();

	const r = $derived(size * 0.375);
	const cx = $derived(size / 2);
	const cy = $derived(size / 2);
	const circ = $derived(2 * Math.PI * r);
	const sum = $derived(total ?? data.reduce((a, s) => a + s.value, 0));

	type Slice = Segment & { offset: number; length: number };

	const slices = $derived(
		data.reduce<{ acc: Slice[]; offset: number }>(
			(state, d) => {
				const length = (d.value / Math.max(sum, 1)) * circ;
				state.acc.push({ ...d, offset: state.offset, length });
				state.offset += length;
				return state;
			},
			{ acc: [], offset: 0 },
		).acc,
	);
</script>

<svg
	viewBox="0 0 {size} {size}"
	width={size}
	height={size}
	role="img"
	aria-label="Donut chart"
>
	<circle
		cx={cx}
		cy={cy}
		r={r}
		stroke="var(--border)"
		stroke-width={size * 0.1}
		fill="none"
	/>
	{#each slices as s, i (i)}
		<circle
			cx={cx}
			cy={cy}
			r={r}
			stroke={s.color}
			stroke-width={size * 0.1}
			fill="none"
			stroke-dasharray="{s.length} {circ}"
			stroke-dashoffset={-s.offset}
			transform="rotate(-90 {cx} {cy})"
			stroke-linecap="butt"
		/>
	{/each}
	<text
		x={cx}
		y={cy - 4}
		text-anchor="middle"
		font-size={size * 0.175}
		font-weight="700"
		fill="var(--text)"
	>{sum}</text>
	<text
		x={cx}
		y={cy + size * 0.115}
		text-anchor="middle"
		font-size={size * 0.07}
		fill="var(--text-muted)"
	>{totalLabel}</text>
</svg>

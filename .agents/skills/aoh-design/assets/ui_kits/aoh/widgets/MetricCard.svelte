<script lang="ts">
	import { Card, CardContent, CardHeader, CardTitle } from "@mssfoobar/ui/card";
	import { TrendingDown, TrendingUp } from "@lucide/svelte";
	import type { Component, Snippet } from "svelte";

	type TrendDir = "up" | "down" | "flat";

	type Props = {
		title: string;
		value: string;
		trend?: string;
		trendDir?: TrendDir;
		icon?: Component;
		/** CSS color token, e.g. `var(--bg-info-strong)` or `var(--bg-error-strong)`. */
		accent?: string;
		children?: Snippet;
	};

	let {
		title,
		value,
		trend,
		trendDir = "up",
		icon: Icon,
		accent = "var(--bg-info-strong)",
		children,
	}: Props = $props();

	const trendClass = $derived(
		trendDir === "up"
			? "text-(--text-success)"
			: trendDir === "down"
				? "text-(--text-error)"
				: "text-(--text-muted)",
	);
</script>

<Card>
	<CardHeader>
		<CardTitle class="flex items-center gap-2 text-sm font-medium text-(--text-muted)">
			{#if Icon}
				<span
					class="inline-flex size-7 items-center justify-center rounded-md text-[color:var(--text-on-color)]"
					style="background-color: {accent}"
				>
					<Icon class="size-4" />
				</span>
			{/if}
			{title}
		</CardTitle>
	</CardHeader>
	<CardContent class="space-y-2 pb-4">
		<div class="font-(family-name:--font-display) text-3xl font-semibold tabular-nums">
			{value}
		</div>
		{#if trend}
			<div class="flex items-center gap-1 text-xs font-medium {trendClass}">
				{#if trendDir === "up"}<TrendingUp class="size-3.5" />{/if}
				{#if trendDir === "down"}<TrendingDown class="size-3.5" />{/if}
				<span>{trend}</span>
				<span class="text-(--text-muted) font-normal">vs last week</span>
			</div>
		{/if}
		{#if children}{@render children()}{/if}
	</CardContent>
</Card>

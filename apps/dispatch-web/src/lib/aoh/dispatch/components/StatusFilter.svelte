<svelte:options runes={true} />

<!--
  Fleet summary that doubles as the status filter.

  One control does both jobs: the tiles show the count per status over the FULL roster,
  and pressing one narrows the list to that status. Counts never shrink when filtering —
  the summary is about the fleet, the list is about what you are looking at.

  Kept small on purpose: the tiles are the least-used control on the screen, so they
  should not be the loudest. The list is where the eye should land.
-->

<script lang="ts">
	import { Card, CardContent } from '@mssfoobar/ui/card';
	import { ToggleGroup, ToggleGroupItem } from '@mssfoobar/ui/toggle-group';
	import { KNOWN_STATUSES, type UnitStatus } from '../types';
	import { statusRail } from '../format';

	let {
		counts,
		total,
		value = $bindable<UnitStatus[]>([])
	}: {
		counts: Record<UnitStatus, number>;
		total: number;
		value?: UnitStatus[];
	} = $props();
</script>

<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
	<Card class="justify-center">
		<CardContent class="flex items-baseline gap-2 px-3.5 py-2.5">
			<span class="text-xl leading-7 font-semibold tabular-nums">{total}</span>
			<span class="text-xs text-muted-foreground">Total units</span>
		</CardContent>
	</Card>

	<ToggleGroup
		type="multiple"
		bind:value
		variant="outline"
		class="contents"
		aria-label="Filter by status"
	>
		{#each KNOWN_STATUSES as status (status)}
			<ToggleGroupItem
				value={status}
				aria-label={`Show ${status} units`}
				class="h-auto min-w-0 items-baseline justify-start gap-2 rounded-[var(--radius-lg,12px)] bg-card px-3.5 py-2.5 text-left shadow-[var(--shadow-md)] transition-colors duration-150 hover:bg-muted/50 data-[state=on]:border-primary data-[state=on]:bg-accent data-[state=on]:shadow-[inset_0_0_0_1px_var(--ring)]"
			>
				<span
					class="size-2 shrink-0 self-center rounded-full {statusRail[status]}"
					aria-hidden="true"
				></span>
				<span class="text-xl leading-7 font-semibold tabular-nums">{counts[status]}</span>
				<span class="text-xs font-normal text-muted-foreground">{status}</span>
			</ToggleGroupItem>
		{/each}
	</ToggleGroup>
</div>

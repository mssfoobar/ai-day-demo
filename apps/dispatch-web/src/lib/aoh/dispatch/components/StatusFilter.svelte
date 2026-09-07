<svelte:options runes={true} />

<!--
  Fleet summary that doubles as the status filter.

  One control does both jobs: the tiles show the count per status over the FULL roster,
  and pressing one narrows the list to that status. Counts never shrink when filtering —
  the summary is about the fleet, the list is about what you are looking at.

  Composed from @mssfoobar/ui: the Total tile is a Card, the three status tiles are a
  ToggleGroup (type="multiple") whose items are restyled from pills into tiles.
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
		<CardContent class="px-4 py-3">
			<p class="text-2xl leading-8 font-semibold tabular-nums">{total}</p>
			<p class="text-xs text-muted-foreground">Total units</p>
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
				class="h-auto min-w-0 flex-col items-start justify-center gap-0 rounded-[var(--radius-lg,12px)] bg-card px-4 py-3 text-left shadow-[var(--shadow-md)] transition-colors data-[state=on]:border-primary data-[state=on]:bg-accent data-[state=on]:ring-1 data-[state=on]:ring-primary/40"
			>
				<span class="flex w-full items-center gap-2">
					<span class="size-2 shrink-0 rounded-full {statusRail[status]}" aria-hidden="true"></span>
					<span class="text-2xl leading-8 font-semibold tabular-nums">{counts[status]}</span>
				</span>
				<span class="text-xs font-normal text-muted-foreground">{status}</span>
			</ToggleGroupItem>
		{/each}
	</ToggleGroup>
</div>

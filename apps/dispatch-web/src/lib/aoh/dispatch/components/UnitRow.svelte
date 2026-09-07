<svelte:options runes={true} />

<!--
  One selectable row in the units list.

  Type roles are deliberately limited to three, used identically everywhere:
    primary   text-sm font-semibold            (call sign)
    secondary text-xs text-muted-foreground     (context)
    data      font-mono text-xs tabular-nums    (ids, times)

  Colour is limited to two carriers per row: the status rail (scan) and the priority chip
  (triage). The status badge is a neutral outline — the rail already says it — and status
  is still readable as text, never colour alone.
-->

<script lang="ts">
	import { Badge } from '@mssfoobar/ui/badge';
	import { Button } from '@mssfoobar/ui/button';
	import Ambulance from '@lucide/svelte/icons/ambulance';
	import Flame from '@lucide/svelte/icons/flame';
	import LifeBuoy from '@lucide/svelte/icons/life-buoy';
	import ShieldAlert from '@lucide/svelte/icons/shield-alert';
	import Truck from '@lucide/svelte/icons/truck';
	import type { FieldUnit } from '../types';
	import { priorityColor, sinceLabel, statusRail } from '../format';

	let {
		unit,
		selected = false,
		now,
		onselect
	}: {
		unit: FieldUnit;
		selected?: boolean;
		now: number;
		onselect: (unit: FieldUnit) => void;
	} = $props();

	const TYPE_ICON: Record<string, typeof Truck> = {
		Ambulance,
		'Fire engine': Flame,
		'Rescue tender': LifeBuoy,
		'Patrol car': ShieldAlert
	};
	const TypeIcon = $derived(TYPE_ICON[unit.unitType] ?? Truck);
</script>

<Button
	variant="ghost"
	data-unit-id={unit.id}
	aria-current={selected ? 'true' : undefined}
	onclick={() => onselect(unit)}
	class="group/row h-auto w-full items-stretch justify-start gap-2.5 rounded-md py-1.5 pr-2.5 pl-2 text-left transition-colors duration-150 {selected
		? 'bg-accent text-accent-foreground shadow-[inset_0_0_0_1px_var(--ring)] hover:bg-accent'
		: 'hover:bg-muted/50'}"
>
	<span class="w-1 shrink-0 rounded-full {statusRail[unit.status]}" aria-hidden="true"></span>

	<TypeIcon
		class="mt-0.5 size-4 shrink-0 {selected
			? 'text-foreground'
			: 'text-muted-foreground group-hover/row:text-foreground'}"
		aria-hidden="true"
	/>

	<span class="min-w-0 flex-1">
		<span class="flex items-baseline gap-2">
			<span class="truncate text-sm leading-5 font-semibold">{unit.callSign}</span>
			<span class="font-mono text-xs text-muted-foreground tabular-nums">{unit.id}</span>
		</span>
		<span class="flex items-center gap-1.5 text-xs leading-4 text-muted-foreground">
			{#if unit.assignment}
				<Badge
					variant="soft"
					color={priorityColor(unit.assignment.priority)}
					class="h-4 shrink-0 px-1.5 text-[10px] leading-4"
				>
					{unit.assignment.priority}
				</Badge>
				<span class="truncate">
					<span class="font-mono tabular-nums">{unit.assignment.incidentCode}</span>
					· {unit.assignment.title}
				</span>
			{:else}
				<span class="truncate">{unit.unitType} · {unit.station}</span>
			{/if}
		</span>
	</span>

	<!-- Trailing column: right-aligned data on both lines. -->
	<span class="flex w-20 shrink-0 flex-col items-end justify-between">
		<span class="font-mono text-xs leading-5 text-muted-foreground tabular-nums">
			{sinceLabel(unit.lastContact, now)}
		</span>
		<Badge variant="outline" class="h-4 px-1.5 text-[10px] leading-4">{unit.status}</Badge>
	</span>
</Button>

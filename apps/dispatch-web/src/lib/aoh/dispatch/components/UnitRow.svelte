<svelte:options runes={true} />

<!--
  One selectable row in the units list.

  A `Button variant="ghost"` (keyboard + focus ring for free) carrying: a status rail,
  a type icon, call sign + id, a context line (assignment or station), a priority chip when
  assigned, and the status badge. Selection is `aria-current` PLUS a visible treatment —
  `variant="ghost"` styles no `aria-current` state, so the attribute alone is invisible.
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
	import { priorityColor, recency, sinceLabel, statusColor, statusRail } from '../format';

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

	const age = $derived(recency(unit.lastContact, now));
</script>

<Button
	variant="ghost"
	data-unit-id={unit.id}
	aria-current={selected ? 'true' : undefined}
	onclick={() => onselect(unit)}
	class="group/row h-auto w-full items-stretch justify-start gap-3 rounded-md py-2 pr-3 pl-2 text-left transition-colors {selected
		? 'bg-accent text-accent-foreground shadow-[inset_0_0_0_1px_var(--ring)]'
		: ''}"
>
	<!-- Status rail: colour AND the badge text carry the status, never colour alone. -->
	<span class="w-1 shrink-0 rounded-full {statusRail[unit.status]}" aria-hidden="true"></span>

	<TypeIcon
		class="mt-0.5 size-5 shrink-0 {selected
			? 'text-foreground'
			: 'text-muted-foreground group-hover/row:text-foreground'}"
		aria-hidden="true"
	/>

	<span class="min-w-0 flex-1">
		<span class="flex items-baseline gap-2">
			<span class="text-sm leading-5 font-semibold">{unit.callSign}</span>
			<span class="font-mono text-xs text-muted-foreground tabular-nums">{unit.id}</span>
			<span class="ml-auto text-[11px] leading-4 text-muted-foreground tabular-nums">
				{sinceLabel(unit.lastContact, now)}
			</span>
		</span>
		<span class="mt-0.5 flex items-center gap-2">
			{#if unit.assignment}
				<Badge variant="soft" color={priorityColor(unit.assignment.priority)} class="shrink-0">
					{unit.assignment.priority}
				</Badge>
				<span class="truncate text-xs leading-4 text-muted-foreground">
					<span class="font-mono tabular-nums">{unit.assignment.incidentCode}</span>
					· {unit.assignment.title}
				</span>
			{:else}
				<span class="truncate text-xs leading-4 text-muted-foreground">
					{unit.unitType} · {unit.station}
				</span>
			{/if}
		</span>
	</span>

	<span class="flex shrink-0 flex-col items-end justify-between gap-1">
		<Badge variant="soft" color={statusColor[unit.status]}>{unit.status}</Badge>
		<!-- Recency dot: fresh / ageing / stale. Text label sits on the first line. -->
		<span
			class="size-1.5 rounded-full {age === 'fresh'
				? 'bg-success'
				: age === 'ageing'
					? 'bg-warning'
					: 'bg-muted-foreground/40'}"
			aria-hidden="true"
		></span>
	</span>
</Button>

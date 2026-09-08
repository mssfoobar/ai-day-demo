<svelte:options runes={true} />

<!--
  The right-hand pane.

  With a unit selected: an identity header (call sign, id, status, one muted context line),
  then Overview / Assignment / Crew / Capabilities. With nothing selected: the fleet at a
  glance — the same roster read a different way, so the pane is never a placeholder.
-->

<script lang="ts">
	import { fade, fly } from 'svelte/transition';
	import { Badge } from '@mssfoobar/ui/badge';
	import { Button } from '@mssfoobar/ui/button';
	import { Separator } from '@mssfoobar/ui/separator';
	import Crosshair from '@lucide/svelte/icons/crosshair';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Radio from '@lucide/svelte/icons/radio';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import type { FieldUnit } from '../types';
	import { FOCUS_CLASS, type ExerciseNumber } from '../workshop';
	import { fleetSummary } from '../filters';
	import { priorityColor, sinceLabel, statusColor } from '../format';

	let {
		unit,
		units,
		now,
		onedit,
		ondelete,
		onexercise,
		focus = null
	}: {
		unit: FieldUnit | null;
		units: FieldUnit[];
		now: number;
		/** Optional: when provided, Edit / Delete appear in the identity header. */
		onedit?: () => void;
		ondelete?: () => void;
		/** Workshop layer (WORKSHOP.md): when provided, violet pins float over the pane where each
		 *  exercise is meant to be built, and clicking one opens its brief. */
		onexercise?: (exercise: ExerciseNumber) => void;
		/** Workshop focus mode: the exercise whose pin should pulse and scroll into view. */
		focus?: ExerciseNumber | null;
	} = $props();

	const fleet = $derived(fleetSummary(units));

	// Workshop focus mode: bring the focused pin into view.
	let targets = $state<Record<ExerciseNumber, HTMLElement | null>>({ 1: null, 2: null, 3: null });
	$effect(() => {
		const el = focus === null ? null : targets[focus];
		if (el) el.scrollIntoView({ block: 'center', behavior: 'smooth' });
	});
</script>

<!-- A default, not `count?:` — Svelte strips the TS annotation but leaves the `?`, and
     the emitted JS then fails to parse. -->
{#snippet sectionTitle(label: string, count: number | undefined = undefined)}
	<h3 class="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
		{label}
		{#if count !== undefined}
			<span class="ml-1 font-normal normal-case tabular-nums">({count})</span>
		{/if}
	</h3>
{/snippet}

<!--
  Workshop pins. Absolutely positioned and deliberately styled as annotations — tilted,
  shadowed, overlapping the content — so nobody mistakes them for console controls. They take
  no layout space; delete an exercise's pin when you build it. See WORKSHOP.md.
-->
{#snippet pin(exercise: ExerciseNumber, label: string, position: string)}
	{#if onexercise}
		<span
			class="pointer-events-none absolute z-10 flex items-center gap-2 {position}"
			bind:this={targets[exercise]}
			in:fly={{ y: -8, duration: 300 }}
		>
			<Button
				variant="ghost"
				size="sm"
				class="pointer-events-auto h-7 -rotate-1 gap-1.5 rounded-md bg-(--workshop-strong) px-2 text-[11px] font-bold text-(--workshop-fg) shadow-lg ring-2 shadow-black/30 ring-background transition-transform duration-200 hover:scale-105 hover:rotate-0 hover:bg-(--workshop-strong) {focus ===
				exercise
					? `${FOCUS_CLASS} scale-110 rotate-0`
					: ''}"
				onclick={() => onexercise?.(exercise)}
			>
				<span
					class="grid size-4 place-items-center rounded-full bg-(--workshop-fg) text-[10px] text-(--workshop-text) tabular-nums"
					>{exercise}</span
				>
				{label}
			</Button>
			{#if focus === exercise}
				<span
					in:fly={{ x: -10, duration: 260 }}
					out:fade={{ duration: 220 }}
					class="inline-flex items-center gap-1 rounded-full bg-(--workshop-strong) px-2 py-0.5 text-[10px] font-bold tracking-wide text-(--workshop-fg) uppercase shadow-md"
				>
					<Crosshair class="size-3" aria-hidden="true" />
					Build here
				</span>
			{/if}
		</span>
	{/if}
{/snippet}

{#if unit}
	<!-- Identity header -->
	<header class="relative flex items-start justify-between gap-4 px-5 py-4">
		{@render pin(1, 'Dispatch goes here', '-top-3 right-40')}
		<div class="min-w-0">
			<div class="flex items-baseline gap-2">
				<h2 class="truncate text-lg leading-6 font-semibold">{unit.callSign}</h2>
				<span class="font-mono text-xs text-muted-foreground tabular-nums">{unit.id}</span>
			</div>
			<p class="mt-0.5 flex flex-wrap items-center gap-x-1.5 text-xs text-muted-foreground">
				<span>{unit.unitType}</span>
				<span aria-hidden="true">·</span>
				<span class="inline-flex items-center gap-1">
					<Radio class="size-3" aria-hidden="true" />
					<span class="font-mono">{unit.radioChannel}</span>
				</span>
				<span aria-hidden="true">·</span>
				<span>{unit.station}</span>
			</p>
		</div>
		<div class="flex shrink-0 flex-col items-end gap-1.5">
			{#if onedit || ondelete}
				<div class="flex gap-1">
					{#if onedit}
						<Button variant="ghost" size="sm" onclick={onedit} class="h-7 gap-1 px-2 text-xs">
							<Pencil class="size-3.5" aria-hidden="true" />
							Edit
						</Button>
					{/if}
					{#if ondelete}
						<Button
							variant="ghost"
							size="sm"
							onclick={ondelete}
							class="h-7 gap-1 px-2 text-xs text-destructive hover:text-destructive"
						>
							<Trash2 class="size-3.5" aria-hidden="true" />
							Delete
						</Button>
					{/if}
				</div>
			{/if}
			<Badge variant="soft" color={statusColor[unit.status]}>{unit.status}</Badge>
			<span class="font-mono text-xs text-muted-foreground tabular-nums">
				{sinceLabel(unit.lastContact, now)}
			</span>
		</div>
	</header>

	<Separator />

	<div class="space-y-5 px-5 py-4">
		<section>
			{@render sectionTitle('Overview')}
			<dl class="grid grid-cols-[120px_1fr] gap-x-4 gap-y-1.5 text-xs">
				<dt class="text-muted-foreground">Sector</dt>
				<dd class="inline-flex items-center gap-1.5">
					<MapPin class="size-3 text-muted-foreground" aria-hidden="true" />
					{unit.sector}
				</dd>
				<dt class="text-muted-foreground">Shift</dt>
				<dd>{unit.shift}</dd>
				<dt class="text-muted-foreground">Radio</dt>
				<dd class="font-mono tabular-nums">{unit.radioChannel}</dd>
				<dt class="text-muted-foreground">Last contact</dt>
				<dd class="font-mono tabular-nums">{sinceLabel(unit.lastContact, now)}</dd>
			</dl>
		</section>

		<section>
			{@render sectionTitle('Assignment')}
			{#if unit.assignment}
				<div class="rounded-md border bg-muted/60 p-3">
					<div class="flex items-baseline gap-2">
						<Badge variant="soft" color={priorityColor(unit.assignment.priority)}>
							{unit.assignment.priority}
						</Badge>
						<span class="font-mono text-xs text-muted-foreground tabular-nums">
							{unit.assignment.incidentCode}
						</span>
						<span class="truncate text-sm font-semibold">{unit.assignment.title}</span>
						<span class="ml-auto shrink-0 font-mono text-xs text-muted-foreground tabular-nums">
							{sinceLabel(unit.assignment.since, now)}
						</span>
					</div>
					<p class="mt-1.5 inline-flex items-center gap-1.5 text-xs text-muted-foreground">
						<MapPin class="size-3" aria-hidden="true" />
						{unit.assignment.location}
					</p>
				</div>
			{:else}
				<p class="text-xs text-muted-foreground">Not currently assigned.</p>
			{/if}
		</section>

		<!-- Workshop exercise 2 anchor: zero height, the pin floats over the section boundary. -->
		<div class="relative h-0">
			{@render pin(2, 'Activity timeline goes here', '-top-3 left-1/2 -translate-x-1/2')}
		</div>

		<div class="grid gap-5 sm:grid-cols-2">
			<section class="relative">
				{@render pin(3, 'Manage crew goes here', '-top-3 right-0')}
				{@render sectionTitle('Crew', unit.crew.length)}
				{#if unit.crew.length > 0}
					<ul class="divide-y divide-border">
						{#each unit.crew as member (member.name)}
							<li class="flex items-baseline justify-between gap-3 py-1.5">
								<span class="text-sm">{member.name}</span>
								<span class="text-xs text-muted-foreground">{member.role}</span>
							</li>
						{/each}
					</ul>
				{:else}
					<p class="text-xs text-muted-foreground">None recorded.</p>
				{/if}
			</section>

			<section>
				{@render sectionTitle('Capabilities')}
				{#if unit.capabilities.length > 0}
					<div class="flex flex-wrap gap-1.5">
						{#each unit.capabilities as capability (capability)}
							<Badge variant="outline">{capability}</Badge>
						{/each}
					</div>
				{:else}
					<p class="text-xs text-muted-foreground">None recorded.</p>
				{/if}
			</section>
		</div>
	</div>
{:else}
	<!-- Nothing selected: fleet at a glance. -->
	<header class="px-5 py-4">
		<h2 class="text-lg leading-6 font-semibold">Fleet at a glance</h2>
		<p class="mt-0.5 text-xs text-muted-foreground">Select a unit for its detail.</p>
	</header>

	<Separator />

	<div class="space-y-5 px-5 py-4">
		<dl class="grid grid-cols-3 gap-3">
			<div class="rounded-md border p-3">
				<dd class="text-xl leading-7 font-semibold tabular-nums">{fleet.total}</dd>
				<dt class="text-xs text-muted-foreground">Units</dt>
			</div>
			<div class="rounded-md border p-3">
				<dd class="text-xl leading-7 font-semibold tabular-nums">{fleet.assigned}</dd>
				<dt class="text-xs text-muted-foreground">Assigned</dt>
			</div>
			<div class="rounded-md border p-3">
				<dd class="text-xl leading-7 font-semibold tabular-nums">{fleet.free}</dd>
				<dt class="text-xs text-muted-foreground">Free</dt>
			</div>
		</dl>

		<section>
			{@render sectionTitle('Open P1', fleet.p1.length)}
			{#if fleet.p1.length > 0}
				<ul class="divide-y divide-border">
					{#each fleet.p1 as u (u.id)}
						<li class="flex items-baseline gap-2 py-1.5">
							<Badge variant="soft" color="destructive">P1</Badge>
							<span class="text-sm font-semibold">{u.callSign}</span>
							<span class="truncate text-xs text-muted-foreground">
								<span class="font-mono tabular-nums">{u.assignment?.incidentCode}</span>
								· {u.assignment?.title}
							</span>
							<span class="ml-auto shrink-0 font-mono text-xs text-muted-foreground tabular-nums">
								{sinceLabel(u.assignment?.since ?? '', now)}
							</span>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="text-xs text-muted-foreground">None open.</p>
			{/if}
		</section>

		{#if fleet.quietest}
			<section>
				{@render sectionTitle('Longest since contact')}
				<p class="flex items-baseline gap-2 text-sm">
					<span class="font-semibold">{fleet.quietest.callSign}</span>
					<span class="text-xs text-muted-foreground">{fleet.quietest.station}</span>
					<span class="ml-auto font-mono text-xs text-muted-foreground tabular-nums">
						{sinceLabel(fleet.quietest.lastContact, now)}
					</span>
				</p>
			</section>
		{/if}
	</div>
{/if}

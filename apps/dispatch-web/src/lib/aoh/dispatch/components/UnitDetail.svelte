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
	import { Skeleton } from '@mssfoobar/ui/skeleton';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Radio from '@lucide/svelte/icons/radio';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import type { FieldUnit } from '../types';
	import { FOCUS_BASE_CLASS, FOCUS_ON_CLASS, type ExerciseNumber } from '../workshop';
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
		/** Workshop layer (WORKSHOP.md): when provided, a dashed sketch of the missing control appears
		 *  where the focused exercise is meant to be built; clicking it opens its story. */
		onexercise?: (exercise: ExerciseNumber) => void;
		/** Workshop focus: the one exercise whose sketch is shown. Only one shows at a time. */
		focus?: ExerciseNumber | null;
	} = $props();

	const fleet = $derived(fleetSummary(units));

	// Workshop sketches: dashed, tinted, clearly not real controls. See WORKSHOP.md.
	const sketch =
		'border border-dashed border-(--workshop) bg-(--workshop-muted) text-(--workshop-text) hover:bg-(--workshop-muted-hover) transition-all duration-200 hover:-translate-y-0.5 active:scale-95';
	// Workshop focus: the sketch being pointed at scrolls into view and gets a highlighted border.
	let targets = $state<Record<ExerciseNumber, HTMLElement | null>>({ 1: null, 2: null, 3: null });
	$effect(() => {
		const el = focus === null ? null : targets[focus];
		if (el) el.scrollIntoView({ block: 'center', behavior: 'smooth' });
	});
	const focusClass = (n: ExerciseNumber) =>
		`${FOCUS_BASE_CLASS} ${focus === n ? FOCUS_ON_CLASS : ''}`;
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

{#snippet marker(exercise: ExerciseNumber)}
	<span
		class="grid size-4 shrink-0 place-items-center rounded-full bg-(--workshop-strong) text-[10px] font-bold text-(--workshop-fg) tabular-nums"
	>
		{exercise}
	</span>
{/snippet}

{#if unit}
	<!-- Identity header -->
	<header class="flex items-start justify-between gap-4 px-5 py-4">
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
			{#if onexercise || onedit || ondelete}
				<div class="flex gap-1">
					<!-- Keyed on the unit so the sketches replay their entrance when the selection changes. -->
					{#key unit.id}
						{#if onexercise && focus === 1}
							<!-- Workshop exercise 1: a sketch of the Dispatch / Stand down button. -->
							<span
								bind:this={targets[1]}
								class="inline-flex {focusClass(1)}"
								in:fly={{ y: -6, duration: 250, delay: 80 }}
								out:fade={{ duration: 150 }}
							>
								<Button
									variant="ghost"
									size="sm"
									class="h-7 gap-1.5 px-2 text-xs {sketch}"
									onclick={() => onexercise?.(1)}
								>
									{@render marker(1)}
									{unit.assignment ? 'Stand down' : 'Dispatch'}
								</Button>
							</span>
						{/if}
					{/key}
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

		<!-- Keyed on the unit so the sketches replay their entrance when the selection changes. -->
		{#key unit.id}
			{#if onexercise && focus === 2}
				<!-- Workshop exercise 2: a sketch of the Activity section — a title and three timeline rows. -->
				<section
					bind:this={targets[2]}
					class={focusClass(2)}
					in:fly={{ y: 6, duration: 250, delay: 160 }}
					out:fade={{ duration: 150 }}
				>
					<div class="flex items-center gap-2">
						{@render sectionTitle('Activity')}
						<span class="mb-2">{@render marker(2)}</span>
					</div>
					<Button
						variant="ghost"
						size="sm"
						class="h-auto w-full flex-col items-stretch gap-2 rounded-md p-3 text-left {sketch}"
						onclick={() => onexercise?.(2)}
					>
						{#each [72, 52, 64] as width, i (width)}
							<span
								class="flex items-center gap-2"
								in:fly={{ x: -8, duration: 250, delay: 240 + i * 70 }}
							>
								<span class="size-2 shrink-0 rounded-full bg-(--workshop)/50"></span>
								<Skeleton
									class="h-2.5 animate-none rounded-sm bg-(--workshop)/25"
									style="width: {width}%"
								/>
								<Skeleton class="ml-auto h-2.5 w-10 animate-none rounded-sm bg-(--workshop)/25" />
							</span>
						{/each}
					</Button>
				</section>
			{/if}
		{/key}

		<div class="grid gap-5 sm:grid-cols-2">
			<section>
				<div class="flex items-start justify-between gap-2">
					{@render sectionTitle('Crew', unit.crew.length)}
					<!-- Keyed on the unit so the sketches replay their entrance when the selection changes. -->
					{#key unit.id}
						{#if onexercise && focus === 3}
							<!-- Workshop exercise 3: a sketch of the Manage button. -->
							<span
								bind:this={targets[3]}
								class="-mt-1 inline-flex {focusClass(3)}"
								in:fly={{ y: -6, duration: 250, delay: 240 }}
								out:fade={{ duration: 150 }}
							>
								<Button
									variant="ghost"
									size="sm"
									class="h-6 gap-1.5 px-2 text-xs {sketch}"
									onclick={() => onexercise?.(3)}
								>
									{@render marker(3)}
									Manage
								</Button>
							</span>
						{/if}
					{/key}
				</div>
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

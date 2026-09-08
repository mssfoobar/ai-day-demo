<svelte:options runes={true} />

<!--
  The right-hand pane.

  With a unit selected: an identity header (call sign, id, status, one muted context line),
  then Overview / Assignment / Crew / Capabilities. With nothing selected: the fleet at a
  glance — the same roster read a different way, so the pane is never a placeholder.
-->

<script lang="ts">
	import { Badge } from '@mssfoobar/ui/badge';
	import { Button } from '@mssfoobar/ui/button';
	import { Separator } from '@mssfoobar/ui/separator';
	import Construction from '@lucide/svelte/icons/construction';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Radio from '@lucide/svelte/icons/radio';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import UserPlus from '@lucide/svelte/icons/user-plus';
	import type { FieldUnit } from '../types';
	import { EXERCISES, PLACEHOLDER_CLASS, type ExerciseNumber } from '../workshop';
	import { fleetSummary } from '../filters';
	import { priorityColor, sinceLabel, statusColor } from '../format';

	let {
		unit,
		units,
		now,
		onedit,
		ondelete,
		onexercise
	}: {
		unit: FieldUnit | null;
		units: FieldUnit[];
		now: number;
		/** Optional: when provided, Edit / Delete appear in the identity header. */
		onedit?: () => void;
		ondelete?: () => void;
		/** Workshop placeholders (WORKSHOP.md). When provided, the stubbed Dispatch / Activity /
		 *  Manage crew controls render and open the exercise brief. */
		onexercise?: (exercise: ExerciseNumber) => void;
	} = $props();

	const fleet = $derived(fleetSummary(units));
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
					{#if onexercise}
						<!-- Workshop exercise 1 — placeholder; see WORKSHOP.md. -->
						<Button
							variant="outline"
							size="sm"
							onclick={() => onexercise?.(1)}
							class="h-7 gap-1.5 px-2 text-xs font-bold {PLACEHOLDER_CLASS}"
						>
							<Construction class="size-3.5" aria-hidden="true" />
							Exercise 1 · {unit.assignment ? 'Stand down' : 'Dispatch'}
						</Button>
					{/if}
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

		<!-- Workshop exercise 2 — placeholder; see WORKSHOP.md. -->
		<section>
			{@render sectionTitle('Activity')}
			<div class="{PLACEHOLDER_CLASS} flex items-start gap-3 rounded-md p-3 text-xs">
				<Construction class="mt-0.5 size-5 shrink-0" aria-hidden="true" />
				<div>
					<p class="text-sm font-bold tracking-wide uppercase">
						Exercise 2 · Activity timeline — not built yet
					</p>
					<p class="mt-0.5">
						Show this unit's recent status and assignment changes here, newest first. The service
						has a <code class="font-mono">GET /v1/units/&#123;unit_code&#125;/events</code> stub waiting.
					</p>
				</div>
				{#if onexercise}
					<Button
						variant="outline"
						size="sm"
						class="ml-auto h-7 shrink-0 border-(--border-warning) px-2 text-xs font-semibold text-(--text-warning-strong) hover:bg-(--bg-warning-muted-hover)"
						onclick={() => onexercise?.(2)}
					>
						Details
					</Button>
				{/if}
			</div>
		</section>

		<div class="grid gap-5 sm:grid-cols-2">
			<section>
				<div class="flex items-start justify-between gap-2">
					{@render sectionTitle('Crew', unit.crew.length)}
					{#if onexercise}
						<!-- Workshop exercise 3 — placeholder; see WORKSHOP.md. -->
						<Button
							variant="ghost"
							size="sm"
							onclick={() => onexercise?.(3)}
							class="-mt-1 h-7 gap-1.5 px-2 text-xs font-bold {PLACEHOLDER_CLASS}"
						>
							<UserPlus class="size-3" aria-hidden="true" />
							Exercise 3 · Manage
						</Button>
					{/if}
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

		{#if onexercise}
			<!-- Workshop placeholders — see WORKSHOP.md. Listed here so they are visible before any unit is selected. -->
			<section>
				{@render sectionTitle('Workshop exercises', 3)}
				<ul class="{PLACEHOLDER_CLASS} divide-y divide-(--border-warning-muted) rounded-md">
					{#each Object.values(EXERCISES) as ex (ex.number)}
						<li>
							<button
								type="button"
								class="flex w-full items-center gap-3 px-3 py-2 text-left text-sm hover:bg-(--bg-warning-muted-hover)"
								onclick={() => onexercise?.(ex.number)}
							>
								<Construction class="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
								<span class="min-w-0 flex-1">
									<span class="block font-medium">{ex.title}</span>
									<span class="block truncate text-xs text-muted-foreground">{ex.where}</span>
								</span>
								<Badge variant="solid" color="warning">Exercise {ex.number}</Badge>
							</button>
						</li>
					{/each}
				</ul>
			</section>
		{/if}
	</div>
{/if}

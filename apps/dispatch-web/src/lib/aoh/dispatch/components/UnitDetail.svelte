<svelte:options runes={true} />

<!--
  The right-hand detail pane: an identity strip, then Overview / Assignment / Crew /
  Capabilities. Read-only — the baseline carries no command action.
-->

<script lang="ts">
	import { Badge } from '@mssfoobar/ui/badge';
	import { Separator } from '@mssfoobar/ui/separator';
	import MousePointerClick from '@lucide/svelte/icons/mouse-pointer-click';
	import Radio from '@lucide/svelte/icons/radio';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import Clock from '@lucide/svelte/icons/clock';
	import type { FieldUnit } from '../types';
	import { priorityColor, sinceLabel, statusColor } from '../format';

	let { unit, now }: { unit: FieldUnit | null; now: number } = $props();
</script>

{#if unit}
	<div class="space-y-5">
		<!-- Identity strip: what a dispatcher reads first. -->
		<div class="flex flex-wrap items-center gap-2">
			<Badge variant="soft" color={statusColor[unit.status]}>{unit.status}</Badge>
			<span class="text-sm text-muted-foreground">{unit.unitType}</span>
			<span class="text-sm text-muted-foreground" aria-hidden="true">·</span>
			<span class="inline-flex items-center gap-1 text-sm text-muted-foreground">
				<Radio class="size-3.5" aria-hidden="true" />
				<span class="font-mono">{unit.radioChannel}</span>
			</span>
			<span class="ml-auto inline-flex items-center gap-1 text-xs text-muted-foreground">
				<Clock class="size-3.5" aria-hidden="true" />
				Last contact {sinceLabel(unit.lastContact, now)}
			</span>
		</div>

		<Separator />

		<section>
			<h3 class="mb-3 text-xs font-medium tracking-wide text-muted-foreground uppercase">
				Overview
			</h3>
			<dl class="grid grid-cols-[130px_1fr] gap-x-5 gap-y-2.5">
				<dt class="text-sm text-muted-foreground">Call sign</dt>
				<dd class="text-sm font-medium">{unit.callSign}</dd>
				<dt class="text-sm text-muted-foreground">Unit ID</dt>
				<dd class="font-mono text-sm tabular-nums">{unit.id}</dd>
				<dt class="text-sm text-muted-foreground">Station</dt>
				<dd class="text-sm">{unit.station}</dd>
				<dt class="text-sm text-muted-foreground">Sector</dt>
				<dd class="inline-flex items-center gap-1.5 text-sm">
					<MapPin class="size-3.5 text-muted-foreground" aria-hidden="true" />
					{unit.sector}
				</dd>
				<dt class="text-sm text-muted-foreground">Shift</dt>
				<dd class="text-sm">{unit.shift}</dd>
			</dl>
		</section>

		<Separator />

		<section>
			<h3 class="mb-3 text-xs font-medium tracking-wide text-muted-foreground uppercase">
				Assignment
			</h3>
			{#if unit.assignment}
				<div class="rounded-lg border bg-muted/40 p-3">
					<div class="flex flex-wrap items-center gap-2">
						<Badge variant="soft" color={priorityColor(unit.assignment.priority)}>
							{unit.assignment.priority}
						</Badge>
						<span class="font-mono text-sm tabular-nums">{unit.assignment.incidentCode}</span>
						<span class="text-sm font-medium">{unit.assignment.title}</span>
						<span class="ml-auto text-xs text-muted-foreground">
							Committed {sinceLabel(unit.assignment.since, now)}
						</span>
					</div>
					<p class="mt-2 inline-flex items-center gap-1.5 text-sm text-muted-foreground">
						<MapPin class="size-3.5" aria-hidden="true" />
						{unit.assignment.location}
					</p>
				</div>
			{:else}
				<p class="text-sm text-muted-foreground">Not currently assigned.</p>
			{/if}
		</section>

		<Separator />

		<div class="grid gap-5 sm:grid-cols-2">
			<section>
				<h3 class="mb-3 text-xs font-medium tracking-wide text-muted-foreground uppercase">
					Crew
					<span class="ml-1 font-normal text-muted-foreground/70 normal-case tabular-nums"
						>({unit.crew.length})</span
					>
				</h3>
				{#if unit.crew.length > 0}
					<ul class="divide-y divide-border">
						{#each unit.crew as member (member.name)}
							<li class="flex items-baseline justify-between gap-4 py-1.5 text-sm">
								<span>{member.name}</span>
								<span class="text-xs text-muted-foreground">{member.role}</span>
							</li>
						{/each}
					</ul>
				{:else}
					<p class="text-sm text-muted-foreground">None recorded.</p>
				{/if}
			</section>

			<section>
				<h3 class="mb-3 text-xs font-medium tracking-wide text-muted-foreground uppercase">
					Capabilities
				</h3>
				{#if unit.capabilities.length > 0}
					<div class="flex flex-wrap gap-2">
						{#each unit.capabilities as capability (capability)}
							<Badge variant="outline">{capability}</Badge>
						{/each}
					</div>
				{:else}
					<p class="text-sm text-muted-foreground">None recorded.</p>
				{/if}
			</section>
		</div>
	</div>
{:else}
	<div
		class="flex h-full flex-col items-center justify-center px-6 py-14 text-center text-muted-foreground"
	>
		<MousePointerClick class="mb-3 size-8" aria-hidden="true" />
		<p class="text-sm">Select a unit to see its details.</p>
		<p class="mt-1 text-xs">
			Use <kbd class="kbd">↑</kbd> <kbd class="kbd">↓</kbd> and <kbd class="kbd">Enter</kbd>, or
			click a row.
		</p>
	</div>
{/if}

<style>
	/* A keycap has no @mssfoobar/ui primitive; this is typography on semantic tokens, not
	   a hand-rolled component. */
	.kbd {
		display: inline-block;
		min-width: 1.5rem;
		padding: 0 0.35rem;
		border: 1px solid var(--border);
		border-bottom-width: 2px;
		border-radius: 4px;
		font-family: var(--font-mono, ui-monospace, monospace);
		font-size: 0.7rem;
		line-height: 1.1rem;
		text-align: center;
		color: var(--foreground);
		background: var(--muted);
	}
</style>

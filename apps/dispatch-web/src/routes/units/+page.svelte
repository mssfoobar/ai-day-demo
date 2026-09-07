<svelte:options runes={true} />

<!--
  Dispatch console — a left Units list and a right unit detail pane.

  Units are fetched on the SvelteKit server (see +page.server.ts) from dispatch-svc, so
  the browser only ever talks to its own origin. This app has NO authentication; the page
  sits at a plain `/units` route. See openspec/changes/dispatch-units-service.

  Read-only on purpose: the detail pane carries no command action yet.
-->

<script lang="ts">
	import { Badge } from '@mssfoobar/ui/badge';
	import { Button } from '@mssfoobar/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '@mssfoobar/ui/card';
	import { Separator } from '@mssfoobar/ui/separator';
	import Ambulance from '@lucide/svelte/icons/ambulance';
	import CirclePause from '@lucide/svelte/icons/circle-pause';
	import Flame from '@lucide/svelte/icons/flame';
	import LifeBuoy from '@lucide/svelte/icons/life-buoy';
	import MousePointerClick from '@lucide/svelte/icons/mouse-pointer-click';
	import Navigation from '@lucide/svelte/icons/navigation';
	import Radio from '@lucide/svelte/icons/radio';
	import ShieldAlert from '@lucide/svelte/icons/shield-alert';
	import Truck from '@lucide/svelte/icons/truck';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';

	import type { FieldUnit, UnitStatus } from '$lib/aoh/dispatch/units.server';
	import type { PageData } from './$types';

	// PageData is generated from +page.server.ts's return, so `units` is typed
	// FieldUnit[] here without restating the shape.
	let { data }: { data: PageData } = $props();

	const units = $derived(data.units);

	// Client-side only: selecting a unit must not fetch and must not navigate.
	let selectedId = $state<string | null>(null);
	const selected = $derived(units.find((unit) => unit.id === selectedId) ?? null);

	const STATUSES: UnitStatus[] = ['Available', 'En route', 'Idle'];
	const countOf = (status: UnitStatus) => units.filter((unit) => unit.status === status).length;

	// Badge splits shape (variant) from palette (color) — see aoh-conventions/web.md.
	// Idle is `default`, not `warning`: an idle unit is neutral, not faulted.
	const statusColor: Record<UnitStatus, 'success' | 'info' | 'default'> = {
		Available: 'success',
		'En route': 'info',
		Idle: 'default'
	};

	// Priority is the dispatcher's triage signal, so P1 reads destructive.
	function priorityColor(priority: string): 'destructive' | 'warning' | 'default' {
		if (priority === 'P1') return 'destructive';
		if (priority === 'P2') return 'warning';
		return 'default';
	}

	// Icon by unit type, falling back to the status glyph for a type we do not know.
	function typeIcon(unit: FieldUnit) {
		switch (unit.unitType) {
			case 'Ambulance':
				return Ambulance;
			case 'Fire engine':
				return Flame;
			case 'Rescue tender':
				return LifeBuoy;
			case 'Patrol car':
				return ShieldAlert;
			default:
				return unit.status === 'En route'
					? Navigation
					: unit.status === 'Idle'
						? CirclePause
						: Truck;
		}
	}

	/** "3 min ago" — a dispatcher reads recency, not a wall-clock timestamp. */
	function sinceLabel(iso: string): string {
		const then = Date.parse(iso);
		if (Number.isNaN(then)) return '—';
		const minutes = Math.max(0, Math.round((Date.now() - then) / 60000));
		if (minutes < 1) return 'just now';
		if (minutes === 1) return '1 min ago';
		if (minutes < 60) return `${minutes} min ago`;
		const hours = Math.round(minutes / 60);
		return hours === 1 ? '1 hr ago' : `${hours} hrs ago`;
	}

	function select(unit: FieldUnit) {
		selectedId = unit.id;
	}
</script>

<svelte:head>
	<title>Dispatch console</title>
</svelte:head>

<section class="mx-auto max-w-7xl space-y-6 p-8">
	<header class="space-y-1">
		<h1 class="text-xl font-semibold tracking-tight">Dispatch console</h1>
		<p class="text-sm text-muted-foreground">Field units and their current status.</p>
	</header>

	{#if data.unavailable}
		<!-- @mssfoobar/ui ships no inline Alert primitive; the documented substitute is a
		     Card with a destructive border and title. -->
		<Card class="border-destructive">
			<CardHeader>
				<CardTitle class="flex items-center gap-2 text-destructive">
					<TriangleAlert class="size-5" aria-hidden="true" />
					Units unavailable
				</CardTitle>
			</CardHeader>
			<CardContent>
				<p class="text-sm text-muted-foreground" role="alert">
					Could not reach the dispatch service. Check that it is running, then reload.
				</p>
			</CardContent>
		</Card>
	{:else}
		<!-- Inline metrics on a list/detail page — not a DASH dashboard surface. -->
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<Card>
				<CardContent class="p-4">
					<p class="text-2xl leading-8 font-semibold tabular-nums">{units.length}</p>
					<p class="text-xs text-muted-foreground">Total units</p>
				</CardContent>
			</Card>
			{#each STATUSES as status (status)}
				<Card>
					<CardContent class="p-4">
						<p class="text-2xl leading-8 font-semibold tabular-nums">{countOf(status)}</p>
						<p class="text-xs text-muted-foreground">{status}</p>
					</CardContent>
				</Card>
			{/each}
		</div>

		<div class="grid items-start gap-5 lg:grid-cols-[380px_1fr]">
			<!-- Units pane — Main List archetype, narrowed to a master list. -->
			<Card>
				<CardHeader>
					<CardTitle>Units</CardTitle>
				</CardHeader>
				<CardContent class="p-2">
					<ul>
						{#each units as unit, i (unit.id)}
							{@const TypeIcon = typeIcon(unit)}
							{@const isSelected = unit.id === selectedId}
							{#if i > 0}
								<li aria-hidden="true"><Separator /></li>
							{/if}
							<li>
								<Button
									variant="ghost"
									class="h-auto w-full justify-start gap-3 px-3 py-2.5 text-left {isSelected
										? 'bg-accent text-accent-foreground'
										: ''}"
									aria-current={isSelected ? 'true' : undefined}
									onclick={() => select(unit)}
								>
									<TypeIcon
										class="size-5 shrink-0 {isSelected
											? 'text-foreground'
											: 'text-muted-foreground'}"
										aria-hidden="true"
									/>
									<span class="min-w-0 flex-1">
										<span class="flex items-baseline gap-2">
											<span class="text-sm leading-5 font-semibold">{unit.callSign}</span>
											<span class="font-mono text-xs text-muted-foreground tabular-nums"
												>{unit.id}</span
											>
										</span>
										<!-- Second line: what the unit is doing, or where it lives. -->
										<span class="block truncate text-xs leading-4 text-muted-foreground">
											{#if unit.assignment}
												{unit.assignment.incidentCode} · {unit.assignment.title}
											{:else}
												{unit.station}
											{/if}
										</span>
									</span>
									<Badge variant="soft" color={statusColor[unit.status]}>{unit.status}</Badge>
								</Button>
							</li>
						{/each}
					</ul>
				</CardContent>
			</Card>

			<!-- Detail pane — Details archetype. -->
			<Card>
				<CardHeader>
					<CardTitle>{selected ? selected.callSign : 'Unit detail'}</CardTitle>
				</CardHeader>
				<CardContent>
					<div aria-live="polite">
						{#if selected}
							<div class="space-y-5">
								<section>
									<h3
										class="mb-3 text-xs font-medium tracking-wide text-muted-foreground uppercase"
									>
										Overview
									</h3>
									<dl class="grid grid-cols-[130px_1fr] gap-x-5 gap-y-3">
										<dt class="text-sm text-muted-foreground">Call sign</dt>
										<dd class="text-sm">{selected.callSign}</dd>

										<dt class="text-sm text-muted-foreground">Unit ID</dt>
										<dd class="font-mono text-sm tabular-nums">{selected.id}</dd>

										<dt class="text-sm text-muted-foreground">Status</dt>
										<dd class="text-sm">
											<Badge variant="soft" color={statusColor[selected.status]}>
												{selected.status}
											</Badge>
										</dd>

										<dt class="text-sm text-muted-foreground">Type</dt>
										<dd class="text-sm">{selected.unitType}</dd>

										<dt class="text-sm text-muted-foreground">Station</dt>
										<dd class="text-sm">{selected.station}</dd>

										<dt class="text-sm text-muted-foreground">Sector</dt>
										<dd class="text-sm">{selected.sector}</dd>

										<dt class="text-sm text-muted-foreground">Radio</dt>
										<dd class="flex items-center gap-1.5 font-mono text-sm">
											<Radio class="size-4 text-muted-foreground" aria-hidden="true" />
											{selected.radioChannel}
										</dd>

										<dt class="text-sm text-muted-foreground">Shift</dt>
										<dd class="text-sm">{selected.shift}</dd>

										<dt class="text-sm text-muted-foreground">Last contact</dt>
										<dd class="text-sm">{sinceLabel(selected.lastContact)}</dd>
									</dl>
								</section>

								<Separator />

								<section>
									<h3
										class="mb-3 text-xs font-medium tracking-wide text-muted-foreground uppercase"
									>
										Assignment
									</h3>
									{#if selected.assignment}
										<dl class="grid grid-cols-[130px_1fr] gap-x-5 gap-y-3">
											<dt class="text-sm text-muted-foreground">Incident</dt>
											<dd class="text-sm">
												<span class="font-mono tabular-nums"
													>{selected.assignment.incidentCode}</span
												>
												· {selected.assignment.title}
											</dd>

											<dt class="text-sm text-muted-foreground">Priority</dt>
											<dd class="text-sm">
												<Badge variant="soft" color={priorityColor(selected.assignment.priority)}>
													{selected.assignment.priority}
												</Badge>
											</dd>

											<dt class="text-sm text-muted-foreground">Location</dt>
											<dd class="text-sm">{selected.assignment.location}</dd>

											<dt class="text-sm text-muted-foreground">Committed</dt>
											<dd class="text-sm">{sinceLabel(selected.assignment.since)}</dd>
										</dl>
									{:else}
										<p class="text-sm text-muted-foreground">Not currently assigned.</p>
									{/if}
								</section>

								<Separator />

								<section>
									<h3
										class="mb-3 text-xs font-medium tracking-wide text-muted-foreground uppercase"
									>
										Crew
									</h3>
									{#if selected.crew.length > 0}
										<ul class="space-y-2">
											{#each selected.crew as member (member.name)}
												<li class="flex items-baseline justify-between gap-4 text-sm">
													<span>{member.name}</span>
													<span class="text-xs text-muted-foreground">{member.role}</span>
												</li>
											{/each}
										</ul>
									{:else}
										<p class="text-sm text-muted-foreground">None recorded.</p>
									{/if}
								</section>

								<Separator />

								<section>
									<h3
										class="mb-3 text-xs font-medium tracking-wide text-muted-foreground uppercase"
									>
										Capabilities
									</h3>
									{#if selected.capabilities.length > 0}
										<div class="flex flex-wrap gap-2">
											{#each selected.capabilities as capability (capability)}
												<Badge variant="outline">{capability}</Badge>
											{/each}
										</div>
									{:else}
										<p class="text-sm text-muted-foreground">None recorded.</p>
									{/if}
								</section>
							</div>
						{:else}
							<div class="px-6 py-14 text-center text-muted-foreground">
								<MousePointerClick class="mx-auto mb-3 size-8" aria-hidden="true" />
								<p class="text-sm">Select a unit to see its details.</p>
							</div>
						{/if}
					</div>
				</CardContent>
			</Card>
		</div>
	{/if}
</section>

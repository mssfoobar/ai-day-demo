<svelte:options runes={true} />

<!--
  Baseline dispatch console — a left Units list and a right unit detail pane.

  This app has NO authentication, so the page sits at a plain `/units` route and renders
  with `pnpm dev` and no containers. See the openspec change `baseline-dispatch-console`
  (design.md D1/D2) and its clickable mock at
  openspec/changes/baseline-dispatch-console/design/units-console-mock.html.

  Read-only on purpose: the detail pane carries no command action yet.
-->

<script lang="ts">
	import { Badge } from '@mssfoobar/ui/badge';
	import { Button } from '@mssfoobar/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '@mssfoobar/ui/card';
	import { Separator } from '@mssfoobar/ui/separator';
	import MousePointerClick from '@lucide/svelte/icons/mouse-pointer-click';
	import Navigation from '@lucide/svelte/icons/navigation';
	import CirclePause from '@lucide/svelte/icons/circle-pause';
	import Radio from '@lucide/svelte/icons/radio';

	import { listUnits, type FieldUnit, type UnitStatus } from '$lib/aoh/dispatch/roster';

	const units = listUnits();

	// Client-side only: selecting a unit must not fetch and must not navigate.
	let selectedId = $state<string | null>(null);
	const selected = $derived(units.find((unit) => unit.id === selectedId) ?? null);

	// Badge splits shape (variant) from palette (color) — see aoh-conventions/web.md.
	// Idle is `default`, not `warning`: an idle unit is neutral, not faulted.
	const statusColor: Record<UnitStatus, 'success' | 'info' | 'default'> = {
		Available: 'success',
		'En route': 'info',
		Idle: 'default'
	};

	const statusIcon: Record<UnitStatus, typeof Radio> = {
		Available: Radio,
		'En route': Navigation,
		Idle: CirclePause
	};

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

	<div class="grid items-start gap-5 lg:grid-cols-[360px_1fr]">
		<!-- Units pane — Main List archetype, narrowed to a master list. -->
		<Card>
			<CardHeader>
				<CardTitle>Units</CardTitle>
			</CardHeader>
			<CardContent class="p-2">
				<ul>
					{#each units as unit, i (unit.id)}
						{@const StatusIcon = statusIcon[unit.status]}
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
								<StatusIcon
									class="size-5 shrink-0 {isSelected ? 'text-foreground' : 'text-muted-foreground'}"
									aria-hidden="true"
								/>
								<span class="min-w-0 flex-1">
									<span class="block text-sm leading-5 font-semibold">
										{unit.callSign}
									</span>
									<span
										class="block font-mono text-xs leading-4 text-muted-foreground tabular-nums"
									>
										{unit.id}
									</span>
								</span>
								<Badge variant="soft" color={statusColor[unit.status]}>
									{unit.status}
								</Badge>
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
						<dl class="grid grid-cols-[120px_1fr] gap-x-5 gap-y-3">
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
						</dl>
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
</section>

<svelte:options runes={true} />

<!--
  Dispatch console — search, filter, sort and inspect field units.

  Units are fetched on the SvelteKit server (see +page.server.ts) from dispatch-svc, so
  the browser only ever talks to its own origin. This app has NO authentication.
  See openspec/changes/dispatch-units-service.

  Read-only on purpose: nothing here dispatches, reassigns or changes a status.
-->

<script lang="ts">
	import { Button } from '@mssfoobar/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '@mssfoobar/ui/card';
	import { Input } from '@mssfoobar/ui/input';
	import { ScrollArea } from '@mssfoobar/ui/scroll-area';
	import { Select, SelectContent, SelectItem, SelectTrigger } from '@mssfoobar/ui/select';
	import { Separator } from '@mssfoobar/ui/separator';
	import ArrowUpDown from '@lucide/svelte/icons/arrow-up-down';
	import Search from '@lucide/svelte/icons/search';
	import SearchX from '@lucide/svelte/icons/search-x';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import X from '@lucide/svelte/icons/x';

	import StatusFilter from '$lib/aoh/dispatch/components/StatusFilter.svelte';
	import UnitDetail from '$lib/aoh/dispatch/components/UnitDetail.svelte';
	import UnitRow from '$lib/aoh/dispatch/components/UnitRow.svelte';
	import {
		countByStatus,
		filterUnits,
		SORT_OPTIONS,
		sortUnits,
		type SortKey
	} from '$lib/aoh/dispatch/filters';
	import type { FieldUnit, UnitStatus } from '$lib/aoh/dispatch/types';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
	const units = $derived(data.units);

	// --- interaction state (all client-side; nothing here fetches or navigates) ---------
	let query = $state('');
	let statuses = $state<UnitStatus[]>([]);
	let sortKey = $state<SortKey>('status');
	let selectedId = $state<string | null>(null);
	let searchRef = $state<HTMLInputElement | null>(null);

	const visible = $derived(sortUnits(filterUnits(units, query, statuses), sortKey));
	const counts = $derived(countByStatus(units));
	const selected = $derived(units.find((u) => u.id === selectedId) ?? null);
	const filtering = $derived(query.trim() !== '' || statuses.length > 0);
	const sortLabel = $derived(SORT_OPTIONS.find((o) => o.value === sortKey)?.label ?? 'Sort');

	// Recency labels ("3 min ago") re-render every 30s without any data refetch.
	let now = $state(Date.now());
	$effect(() => {
		const id = setInterval(() => (now = Date.now()), 30_000);
		return () => clearInterval(id);
	});

	function select(unit: FieldUnit) {
		selectedId = unit.id;
	}

	function clearFilters() {
		query = '';
		statuses = [];
	}

	/** Arrow keys move the selection through the VISIBLE list and keep focus on the row. */
	function onListKeydown(event: KeyboardEvent) {
		if (visible.length === 0) return;
		const keys: Record<string, (i: number) => number> = {
			ArrowDown: (i) => Math.min(i + 1, visible.length - 1),
			ArrowUp: (i) => Math.max(i - 1, 0),
			Home: () => 0,
			End: () => visible.length - 1
		};
		const step = keys[event.key];
		if (!step) return;
		event.preventDefault();

		const current = visible.findIndex((u) => u.id === selectedId);
		const next =
			visible[step(current === -1 ? (event.key === 'ArrowUp' ? visible.length : -1) : current)];
		if (!next) return;
		selectedId = next.id;
		(event.currentTarget as HTMLElement)
			.querySelector<HTMLElement>(`[data-unit-id="${next.id}"]`)
			?.focus();
	}

	/** `/` jumps to search from anywhere; Escape in the search box clears it. */
	function onWindowKeydown(event: KeyboardEvent) {
		const target = event.target as HTMLElement | null;
		const typing = target?.tagName === 'INPUT' || target?.tagName === 'TEXTAREA';
		if (event.key === '/' && !typing) {
			event.preventDefault();
			searchRef?.focus();
		}
	}
</script>

<svelte:head>
	<title>Dispatch console</title>
</svelte:head>

<svelte:window onkeydown={onWindowKeydown} />

<div class="mx-auto flex h-dvh max-w-[1400px] flex-col gap-4 p-6">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<h1 class="text-xl font-semibold tracking-tight">Dispatch console</h1>
			<p class="text-sm text-muted-foreground">Field units and their current status.</p>
		</div>

		{#if !data.unavailable}
			<div class="flex flex-wrap items-center gap-2">
				<div class="relative">
					<Search
						class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground"
						aria-hidden="true"
					/>
					<Input
						bind:ref={searchRef}
						bind:value={query}
						type="search"
						size="sm"
						placeholder="Search units, incidents, crew…  ( / )"
						aria-label="Search units"
						class="w-80 pl-9"
						onkeydown={(e) => {
							if (e.key === 'Escape') {
								query = '';
								(e.currentTarget as HTMLInputElement).blur();
							}
						}}
					/>
				</div>

				<Select type="single" bind:value={sortKey}>
					<SelectTrigger size="sm" class="w-44" aria-label="Sort units">
						<ArrowUpDown class="size-4" aria-hidden="true" />
						<span>{sortLabel}</span>
					</SelectTrigger>
					<SelectContent>
						{#each SORT_OPTIONS as option (option.value)}
							<SelectItem value={option.value} label={option.label} />
						{/each}
					</SelectContent>
				</Select>
			</div>
		{/if}
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
		<StatusFilter {counts} total={units.length} bind:value={statuses} />

		<div class="grid min-h-0 flex-1 items-stretch gap-4 lg:grid-cols-[420px_1fr]">
			<!-- Units pane — Main List archetype, narrowed to a master list. -->
			<Card class="flex min-h-0 flex-col">
				<CardHeader class="flex-row items-center justify-between gap-2 space-y-0 py-3">
					<CardTitle class="flex items-baseline gap-2">
						Units
						<span class="text-xs font-normal text-muted-foreground tabular-nums">
							{visible.length} of {units.length}
						</span>
					</CardTitle>
					{#if filtering}
						<Button
							variant="ghost"
							size="sm"
							onclick={clearFilters}
							class="-mr-2 h-7 gap-1 px-2 text-xs"
						>
							<X class="size-3.5" aria-hidden="true" />
							Clear
						</Button>
					{/if}
				</CardHeader>
				<Separator />
				<ScrollArea class="min-h-0 flex-1">
					{#if visible.length === 0}
						<div class="flex flex-col items-center px-6 py-14 text-center text-muted-foreground">
							<SearchX class="mb-3 size-8" aria-hidden="true" />
							<p class="text-sm">No units match.</p>
							<Button variant="outline" size="sm" onclick={clearFilters} class="mt-3">
								Clear filters
							</Button>
						</div>
					{:else}
						<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
						<ul class="p-1.5" onkeydown={onListKeydown} aria-label="Units">
							{#each visible as unit (unit.id)}
								<li>
									<UnitRow {unit} {now} selected={unit.id === selectedId} onselect={select} />
								</li>
							{/each}
						</ul>
					{/if}
				</ScrollArea>
				<Separator />
				<p class="flex flex-wrap gap-x-3 px-4 py-2 text-[11px] leading-4 text-muted-foreground">
					<span><kbd class="kbd">↑</kbd><kbd class="kbd">↓</kbd> move</span>
					<span><kbd class="kbd">Enter</kbd> select</span>
					<span><kbd class="kbd">/</kbd> search</span>
					<span><kbd class="kbd">Esc</kbd> clear</span>
				</p>
			</Card>

			<!-- Detail pane — Details archetype. -->
			<Card class="flex min-h-0 flex-col">
				<CardHeader class="py-3">
					<CardTitle class="flex items-baseline gap-2">
						{selected ? selected.callSign : 'Unit detail'}
						{#if selected}
							<span class="font-mono text-xs font-normal text-muted-foreground tabular-nums">
								{selected.id}
							</span>
						{/if}
					</CardTitle>
				</CardHeader>
				<Separator />
				<ScrollArea class="min-h-0 flex-1">
					<div class="h-full p-5" aria-live="polite">
						<UnitDetail unit={selected} {now} />
					</div>
				</ScrollArea>
			</Card>
		</div>
	{/if}
</div>

<style>
	/* Keycap styling on semantic tokens — there is no @mssfoobar/ui kbd primitive. */
	.kbd {
		display: inline-block;
		min-width: 1.25rem;
		margin-right: 0.15rem;
		padding: 0 0.3rem;
		border: 1px solid var(--border);
		border-bottom-width: 2px;
		border-radius: 4px;
		font-family: var(--font-mono, ui-monospace, monospace);
		font-size: 0.65rem;
		line-height: 1rem;
		text-align: center;
		color: var(--foreground);
		background: var(--muted);
	}
</style>

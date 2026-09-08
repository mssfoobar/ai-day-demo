<svelte:options runes={true} />

<!--
  Dispatch console — search, filter, sort, inspect, and (basic) add / edit / delete field
  units.

  Reads happen in +page.server.ts's load; writes go through its form actions. Either way
  the browser only ever talks to its own origin and the server talks to dispatch-svc. This
  app has NO authentication. See openspec/changes/dispatch-units-service and
  openspec/changes/dispatch-units-crud.
-->

<script lang="ts">
	import { enhance } from '$app/forms';
	import { fly } from 'svelte/transition';
	import {
		AlertDialog,
		AlertDialogCancel,
		AlertDialogContent,
		AlertDialogDescription,
		AlertDialogFooter,
		AlertDialogHeader,
		AlertDialogTitle
	} from '@mssfoobar/ui/alert-dialog';
	import { Button } from '@mssfoobar/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '@mssfoobar/ui/card';
	import { Input } from '@mssfoobar/ui/input';
	import { ScrollArea } from '@mssfoobar/ui/scroll-area';
	import { Select, SelectContent, SelectItem, SelectTrigger } from '@mssfoobar/ui/select';
	import { Separator } from '@mssfoobar/ui/separator';
	import { toast } from '@mssfoobar/ui/toast';
	import ArrowUpDown from '@lucide/svelte/icons/arrow-up-down';
	import Construction from '@lucide/svelte/icons/construction';
	import Plus from '@lucide/svelte/icons/plus';
	import Search from '@lucide/svelte/icons/search';
	import SearchX from '@lucide/svelte/icons/search-x';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import X from '@lucide/svelte/icons/x';

	import StatusFilter from '$lib/aoh/dispatch/components/StatusFilter.svelte';
	import ExerciseDialog from '$lib/aoh/dispatch/components/ExerciseDialog.svelte';
	import UnitDetail from '$lib/aoh/dispatch/components/UnitDetail.svelte';
	import UnitForm from '$lib/aoh/dispatch/components/UnitForm.svelte';
	import UnitRow from '$lib/aoh/dispatch/components/UnitRow.svelte';
	import {
		countByStatus,
		filterUnits,
		fleetSummary,
		SORT_OPTIONS,
		sortUnits,
		type SortKey
	} from '$lib/aoh/dispatch/filters';
	import { sinceLabel } from '$lib/aoh/dispatch/format';
	import type { FieldUnit, UnitStatus } from '$lib/aoh/dispatch/types';
	import type { UnitFormErrors } from '$lib/aoh/dispatch/forms';
	import { incompleteExercises, type ExerciseNumber } from '$lib/aoh/dispatch/workshop';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
	const units = $derived(data.units);

	// --- interaction state (client-side; only the forms below talk to the server) --------
	let query = $state('');
	let statuses = $state<UnitStatus[]>([]);
	let sortKey = $state<SortKey>('status');
	let selectedId = $state<string | null>(null);
	let searchRef = $state<HTMLInputElement | null>(null);

	// Write UI state.
	let formOpen = $state(false);
	// Workshop layer — see WORKSHOP.md.
	let exerciseOpen = $state(false);
	let focused = $state<ExerciseNumber | null>(null);
	const incomplete = incompleteExercises();
	let formMode = $state<'create' | 'edit'>('create');
	let deleteOpen = $state(false);
	let deleting = $state(false);

	const visible = $derived(sortUnits(filterUnits(units, query, statuses), sortKey));
	const counts = $derived(countByStatus(units));
	const fleet = $derived(fleetSummary(units));
	// Re-resolved against the freshly loaded roster after every write, so a deleted unit
	// deselects itself and an edited one shows its new values.
	const selected = $derived(units.find((u) => u.id === selectedId) ?? null);
	const filtering = $derived(query.trim() !== '' || statuses.length > 0);
	const sortLabel = $derived(SORT_OPTIONS.find((o) => o.value === sortKey)?.label ?? 'Sort');
	const overlayOpen = $derived(formOpen || deleteOpen || exerciseOpen);

	// Recency labels ("3 min ago") re-render every 30s without any data refetch. The
	// header's "updated …" is anchored to the last load.
	let loadedAt = $state(new Date().toISOString());
	let now = $state(Date.now());
	$effect(() => {
		const id = setInterval(() => (now = Date.now()), 30_000);
		return () => clearInterval(id);
	});
	$effect(() => {
		// Any change to the roster counts as an update for the header line.
		void units;
		loadedAt = new Date().toISOString();
	});

	function select(unit: FieldUnit) {
		selectedId = unit.id;
	}

	function clearFilters() {
		query = '';
		statuses = [];
	}

	function openCreate() {
		formMode = 'create';
		formOpen = true;
	}

	function openEdit() {
		if (!selected) return;
		formMode = 'edit';
		formOpen = true;
	}

	/** Workshop layer — see WORKSHOP.md. */
	function openExercises() {
		exerciseOpen = true;
	}

	/**
	 * Focus mode: close the panel, make sure a unit is selected so the detail pane is showing,
	 * then let UnitDetail show the sketch for that exercise — and only that one — and scroll to it.
	 * It stays until another exercise is chosen.
	 */
	function focusExercise(n: ExerciseNumber) {
		exerciseOpen = false;
		if (!selected) {
			const pick = n === 1 ? (units.find((u) => !u.assignment) ?? units[0]) : units[0];
			if (pick) selectedId = pick.id;
		}
		focused = n;
	}

	function onFormOutcome(
		outcome:
			| { ok: true; intent: 'create' | 'update'; unitCode: string; callSign: string }
			| { ok: false; intent: 'create' | 'update'; errors: UnitFormErrors }
	) {
		if (outcome.ok) {
			toast.success(`${outcome.callSign} ${outcome.intent === 'create' ? 'added' : 'saved'}`);
			selectedId = outcome.unitCode;
			return;
		}
		// Field errors render inside the form; only a form-level failure needs a toast.
		if (outcome.errors.form) toast.error(outcome.errors.form);
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

	/** `/` jumps to search from anywhere (unless an overlay is open); Escape in the box clears it. */
	function onWindowKeydown(event: KeyboardEvent) {
		if (overlayOpen) return;
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

<div class="mx-auto flex h-dvh max-w-[1400px] flex-col gap-3 p-5">
	<header class="flex flex-wrap items-end justify-between gap-3">
		<div>
			<h1 class="text-lg leading-6 font-semibold tracking-tight">Dispatch console</h1>
			{#if data.unavailable}
				<p class="text-xs text-muted-foreground">Field units and their current status.</p>
			{:else}
				<p class="text-xs text-muted-foreground tabular-nums">
					{fleet.total} units · {fleet.assigned} assigned · {fleet.free} free · updated
					{sinceLabel(loadedAt, now)}
				</p>
			{/if}
		</div>

		<div class="flex flex-wrap items-center gap-2">
			{#if !data.unavailable}
				<div class="relative">
					<Search
						class="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground"
						aria-hidden="true"
					/>
					<Input
						bind:ref={searchRef}
						bind:value={query}
						type="search"
						size="sm"
						placeholder="Search units, incidents, crew…  /"
						aria-label="Search units"
						class="w-72 pl-8 text-xs"
						onkeydown={(e) => {
							if (e.key === 'Escape') {
								query = '';
								(e.currentTarget as HTMLInputElement).blur();
							}
						}}
					/>
				</div>

				<Select type="single" bind:value={sortKey}>
					<SelectTrigger size="sm" class="w-40 text-xs" aria-label="Sort units">
						<ArrowUpDown class="size-3.5" aria-hidden="true" />
						<span>{sortLabel}</span>
					</SelectTrigger>
					<SelectContent>
						{#each SORT_OPTIONS as option (option.value)}
							<SelectItem value={option.value} label={option.label} />
						{/each}
					</SelectContent>
				</Select>

				<Button size="sm" onclick={openCreate} class="gap-1.5 text-xs">
					<Plus class="size-3.5" aria-hidden="true" />
					Add unit
				</Button>
			{/if}
		</div>
	</header>

	{#if data.unavailable}
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

		<div class="grid min-h-0 flex-1 items-stretch gap-3 lg:grid-cols-[400px_1fr]">
			<!-- Units pane — Main List archetype, narrowed to a master list. -->
			<Card class="flex min-h-0 flex-col">
				<CardHeader class="flex-row items-center justify-between gap-2 space-y-0 px-4 py-2.5">
					<CardTitle class="flex items-baseline gap-2 text-sm">
						Units
						<span class="font-mono text-xs font-normal text-muted-foreground tabular-nums">
							{visible.length}/{units.length}
						</span>
					</CardTitle>
					{#if filtering}
						<Button
							variant="ghost"
							size="sm"
							onclick={clearFilters}
							class="-mr-2 h-6 gap-1 px-2 text-xs"
						>
							<X class="size-3" aria-hidden="true" />
							Clear
						</Button>
					{/if}
				</CardHeader>
				<Separator />
				<ScrollArea class="min-h-0 flex-1">
					{#if visible.length === 0}
						<div class="flex flex-col items-center px-6 py-14 text-center text-muted-foreground">
							<SearchX class="mb-3 size-8" aria-hidden="true" />
							<p class="text-sm">{units.length === 0 ? 'No units yet.' : 'No units match.'}</p>
							{#if units.length === 0}
								<Button variant="outline" size="sm" onclick={openCreate} class="mt-3">
									Add the first unit
								</Button>
							{:else}
								<Button variant="outline" size="sm" onclick={clearFilters} class="mt-3">
									Clear filters
								</Button>
							{/if}
						</div>
					{:else}
						<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
						<ul class="space-y-px p-1.5" onkeydown={onListKeydown} aria-label="Units">
							{#each visible as unit (unit.id)}
								<li>
									<UnitRow {unit} {now} selected={unit.id === selectedId} onselect={select} />
								</li>
							{/each}
						</ul>
					{/if}
				</ScrollArea>
				<Separator />
				<p class="flex flex-wrap gap-x-3 px-4 py-1.5 text-xs leading-4 text-muted-foreground">
					<span><kbd class="kbd">↑</kbd><kbd class="kbd">↓</kbd> move</span>
					<span><kbd class="kbd">Enter</kbd> select</span>
					<span><kbd class="kbd">/</kbd> search</span>
					<span><kbd class="kbd">Esc</kbd> clear</span>
				</p>
			</Card>

			<!-- Detail pane — Details archetype. UnitDetail owns its own header. -->
			<Card class="flex min-h-0 flex-col">
				<ScrollArea class="min-h-0 flex-1">
					<div aria-live="polite">
						<UnitDetail
							unit={selected}
							{units}
							{now}
							onedit={openEdit}
							onexercise={() => openExercises()}
							focus={focused}
							ondelete={() => (deleteOpen = true)}
						/>
					</div>
				</ScrollArea>
			</Card>
		</div>

		<!-- Add / edit -->
		<UnitForm bind:open={formOpen} mode={formMode} unit={selected} onoutcome={onFormOutcome} />
		<ExerciseDialog bind:open={exerciseOpen} onfocus={focusExercise} />

		{#if incomplete.length > 0}
			<!-- Workshop layer — floating on purpose: it is an annotation over the console, not part of
			     it. Opens the exercise dialog. See WORKSHOP.md. -->
			<div class="fixed right-6 bottom-6 z-40" in:fly={{ y: 24, duration: 350, delay: 200 }}>
				<Button
					class="h-11 gap-2 rounded-full bg-(--workshop-strong) px-4 text-sm font-bold text-(--workshop-fg) shadow-lg ring-2 ring-background transition-transform duration-200 hover:scale-105 hover:bg-(--workshop-strong)/90 active:scale-95"
					onclick={() => openExercises()}
				>
					<Construction class="size-4" aria-hidden="true" />
					{incomplete.length === 1
						? 'One thing left to build'
						: `${incomplete.length} things left to build`}
				</Button>
			</div>
		{/if}

		<!-- Delete — confirmation names the unit; the action is a real form post. -->
		<AlertDialog bind:open={deleteOpen}>
			<AlertDialogContent>
				{#if selected}
					<form
						method="POST"
						action="?/delete"
						use:enhance={() => {
							deleting = true;
							const gone = selected;
							return async ({ result, update }) => {
								deleting = false;
								deleteOpen = false;
								if (result.type === 'success') {
									toast.success(`${gone?.callSign ?? 'Unit'} deleted`);
									selectedId = null;
								} else if (result.type === 'failure') {
									const d = result.data as { errors?: UnitFormErrors } | undefined;
									toast.error(d?.errors?.form ?? 'The unit could not be deleted.');
								}
								await update({ reset: false });
							};
						}}
					>
						<input type="hidden" name="unitCode" value={selected.id} />
						<input type="hidden" name="callSign" value={selected.callSign} />
						<input type="hidden" name="occLock" value={selected.occLock} />
						<AlertDialogHeader>
							<AlertDialogTitle>Delete {selected.callSign}?</AlertDialogTitle>
							<AlertDialogDescription>
								This removes <span class="font-mono">{selected.id}</span> and its crew. It cannot be undone.
							</AlertDialogDescription>
						</AlertDialogHeader>
						<AlertDialogFooter>
							<AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
							<Button type="submit" variant="destructive" disabled={deleting}>
								{deleting ? 'Deleting…' : 'Delete'}
							</Button>
						</AlertDialogFooter>
					</form>
				{/if}
			</AlertDialogContent>
		</AlertDialog>
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

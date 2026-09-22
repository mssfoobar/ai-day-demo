<svelte:options runes={true} />

<!--
  Add / edit form for a field unit, in a side Sheet.

  Submits to a SvelteKit form action (`?/create` or `?/update`) with `use:enhance`, so the
  browser posts to its own origin and the server talks to the service. Field errors come
  back from the action and render beside the field; @mssfoobar/ui ships no FormMessage, so
  the error line is the documented sibling-<p> composition.

  Crew and assignment are not editable here — deliberately, see design.md D7.
-->

<script lang="ts">
	import { enhance } from '$app/forms';
	import { Button } from '@mssfoobar/ui/button';
	import { Input } from '@mssfoobar/ui/input';
	import { Label } from '@mssfoobar/ui/label';
	import { Select, SelectContent, SelectItem, SelectTrigger } from '@mssfoobar/ui/select';
	import { ScrollArea } from '@mssfoobar/ui/scroll-area';
	import {
		Sheet,
		SheetContent,
		SheetDescription,
		SheetFooter,
		SheetHeader,
		SheetTitle
	} from '@mssfoobar/ui/sheet';
	import { EMPTY_VALUES, fieldLabel, type UnitFormErrors, type UnitFormValues } from '../forms';
	import { KNOWN_STATUSES, type FieldUnit, type UnitField } from '../types';

	type Outcome =
		| { ok: true; unitCode: string; callSign: string }
		| { ok: false; errors: UnitFormErrors; values?: UnitFormValues };

	let {
		open = $bindable(false),
		mode,
		unit = null,
		onoutcome
	}: {
		open?: boolean;
		mode: 'create' | 'edit';
		/** The unit being edited; ignored for create. */
		unit?: FieldUnit | null;
		onoutcome: (outcome: Outcome & { intent: 'create' | 'update' }) => void;
	} = $props();

	function valuesOf(u: FieldUnit | null): UnitFormValues {
		if (!u) return { ...EMPTY_VALUES };
		return {
			unitCode: u.id,
			callSign: u.callSign,
			status: u.status,
			unitType: u.unitType,
			station: u.station,
			sector: u.sector,
			radioChannel: u.radioChannel,
			shift: u.shift,
			capabilities: u.capabilities.join(', ')
		};
	}

	// Local copy of the values so a failed submit can re-render what was typed, and so
	// reopening for a different unit starts fresh.
	let values = $state<UnitFormValues>({ ...EMPTY_VALUES });
	let errors = $state<UnitFormErrors>({});
	let submitting = $state(false);

	$effect(() => {
		if (open) {
			values = valuesOf(mode === 'edit' ? unit : null);
			errors = {};
		}
	});

	const title = $derived(mode === 'create' ? 'Add unit' : `Edit ${unit?.callSign ?? 'unit'}`);
	const action = $derived(mode === 'create' ? '?/create' : '?/update');

	const FIELDS: { key: UnitField; placeholder: string; mono?: boolean }[] = [
		{ key: 'callSign', placeholder: 'Delta-1' },
		{ key: 'unitType', placeholder: 'Ambulance' },
		{ key: 'station', placeholder: 'Bedok Station 3' },
		{ key: 'sector', placeholder: 'Sector 9 · Bedok' },
		{ key: 'radioChannel', placeholder: 'TAC-3', mono: true },
		{ key: 'shift', placeholder: 'Day · 07:00-19:00' }
	];
</script>

<Sheet bind:open>
	<SheetContent side="right" class="flex w-full flex-col gap-0 p-0 sm:max-w-md">
		<form
			method="POST"
			{action}
			class="flex min-h-0 flex-1 flex-col"
			use:enhance={() => {
				submitting = true;
				return async ({ result, update }) => {
					submitting = false;
					if (result.type === 'success' && result.data) {
						const d = result.data as { unitCode: string; callSign: string };
						open = false;
						onoutcome({
							ok: true,
							intent: mode === 'create' ? 'create' : 'update',
							unitCode: d.unitCode,
							callSign: d.callSign
						});
						// Reload the roster; keep the form's own state untouched.
						await update({ reset: false });
					} else if (result.type === 'failure' && result.data) {
						const d = result.data as { errors?: UnitFormErrors; values?: UnitFormValues };
						errors = d.errors ?? {};
						if (d.values) values = d.values;
						onoutcome({
							ok: false,
							intent: mode === 'create' ? 'create' : 'update',
							errors,
							values: d.values
						});
					} else {
						await update();
					}
				};
			}}
		>
			<SheetHeader class="px-5 pt-5 pb-3">
				<SheetTitle>{title}</SheetTitle>
				<SheetDescription>
					{#if mode === 'create'}
						Give the unit an ID and a call sign; everything else describes where it works.
					{:else}
						Changes save to the dispatch service. Crew and assignment are not editable here yet.
					{/if}
				</SheetDescription>
			</SheetHeader>

			<!-- ScrollArea, not overflow-y-auto: the native scrollbar clashes with the theme. -->
			<ScrollArea class="min-h-0 flex-1">
				<div class="space-y-4 px-5 py-2">
					{#if errors.form}
						<p class="text-sm text-destructive" role="alert">{errors.form}</p>
					{/if}

					<div class="grid gap-1.5">
						<Label for="unitCode">{fieldLabel('unitCode')}</Label>
						<Input
							id="unitCode"
							name="unitCode"
							bind:value={values.unitCode}
							placeholder="FU-401"
							class="font-mono"
							readonly={mode === 'edit'}
							aria-invalid={errors.unitCode ? 'true' : undefined}
							aria-describedby={errors.unitCode ? 'unitCode-error' : undefined}
						/>
						{#if errors.unitCode}
							<p id="unitCode-error" class="text-xs text-destructive">{errors.unitCode}</p>
						{:else if mode === 'create'}
							<p class="text-xs text-muted-foreground">Unique. Dispatchers say this out loud.</p>
						{/if}
					</div>

					{#each FIELDS as f (f.key)}
						<div class="grid gap-1.5">
							<Label for={f.key}>{fieldLabel(f.key)}</Label>
							<Input
								id={f.key}
								name={f.key}
								bind:value={values[f.key]}
								placeholder={f.placeholder}
								class={f.mono ? 'font-mono' : ''}
								aria-invalid={errors[f.key] ? 'true' : undefined}
								aria-describedby={errors[f.key] ? `${f.key}-error` : undefined}
							/>
							{#if errors[f.key]}
								<p id="{f.key}-error" class="text-xs text-destructive">{errors[f.key]}</p>
							{/if}
						</div>
					{/each}

					<div class="grid gap-1.5">
						<Label for="status">{fieldLabel('status')}</Label>
						<!-- `name` makes Select render a hidden input, so the value posts with the form. -->
						<Select type="single" name="status" bind:value={values.status}>
							<SelectTrigger
								id="status"
								class="w-full"
								aria-invalid={errors.status ? 'true' : undefined}
							>
								{values.status || 'Choose a status'}
							</SelectTrigger>
							<SelectContent>
								{#each KNOWN_STATUSES as s (s)}
									<SelectItem value={s} label={s} />
								{/each}
							</SelectContent>
						</Select>
						{#if errors.status}
							<p class="text-xs text-destructive">{errors.status}</p>
						{/if}
					</div>

					<div class="grid gap-1.5">
						<Label for="capabilities">{fieldLabel('capabilities')}</Label>
						<Input
							id="capabilities"
							name="capabilities"
							bind:value={values.capabilities}
							placeholder="ALS, Water rescue"
						/>
						<p class="text-xs text-muted-foreground">Comma-separated. Leave empty for none.</p>
					</div>

					{#if mode === 'edit' && unit}
						<!-- Optimistic concurrency: the version this edit is based on. -->
						<input type="hidden" name="occLock" value={unit.occLock} />
						{#if unit.position}
							<!--
								The unit's existing position, carried through untouched. Positions are
								not authored here, but a write is a REPLACE: without these the edit
								would clear the fix and the unit's marker would vanish from the map.
								`at` rides along too, so an edit that reported no new location leaves
								the fix time where it was instead of looking like a fresh GPS report.
							-->
							<input type="hidden" name="positionLon" value={unit.position.lon} />
							<input type="hidden" name="positionLat" value={unit.position.lat} />
							<input type="hidden" name="positionAt" value={unit.position.at} />
						{/if}
					{/if}
				</div>
			</ScrollArea>

			<SheetFooter class="flex-row justify-end gap-2 border-t px-5 py-3">
				<Button
					type="button"
					variant="outline"
					onclick={() => (open = false)}
					disabled={submitting}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={submitting}>
					{submitting ? 'Saving…' : mode === 'create' ? 'Add unit' : 'Save'}
				</Button>
			</SheetFooter>
		</form>
	</SheetContent>
</Sheet>

<svelte:options runes={true} />

<!--
  Operator map — the tenant's field units on a geospatial canvas.

  The map itself is SDK-owned: `@mssfoobar/gis-web-sdk` supplies the engine, the map, the
  base layer, the entity layer and the layer manager. Nothing here hand-rolls a renderer,
  a tile client or a marker layer. The chrome around it is `@mssfoobar/ui`.

  ENTITY STATE HAS ONE SOURCE (design.md D9): the SDK's RTUS subscription. This page
  issues no geo-entity request of its own, so there is no fetch/SSE race to dedup. The
  roster it loads is *unit* data — a different thing — and is used only for the counts,
  because deriving "not shown" from the entities on the map would make a projection that
  is still draining look like a smaller roster.
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { Badge } from '@mssfoobar/ui/badge';
	import { Button } from '@mssfoobar/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '@mssfoobar/ui/card';
	import { Separator } from '@mssfoobar/ui/separator';
	import Layers from '@lucide/svelte/icons/layers';
	import Lock from '@lucide/svelte/icons/lock';
	import MapPinOff from '@lucide/svelte/icons/map-pin-off';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import Truck from '@lucide/svelte/icons/truck';

	// Subpath imports, not the barrel: each subpath resolves to one `.svelte` component,
	// so these are DEFAULT exports. Going through the barrel would also pull the Cesium
	// engine into every module that touches the SDK — including the root layout, which
	// SSRs — and Cesium touches browser globals at module init.
	import CesiumMapEngineProvider from '@mssfoobar/gis-web-sdk/engines/cesium';
	import GisMapComponent from '@mssfoobar/gis-web-sdk/map';
	import MapBaseLayerProvider from '@mssfoobar/gis-web-sdk/base-layer-provider';
	import MapXyzSourceProvider from '@mssfoobar/gis-web-sdk/xyz-source-provider';
	import MapEntityLayerProvider from '@mssfoobar/gis-web-sdk/entity-layer-provider';
	import MapEntityProvider from '@mssfoobar/gis-web-sdk/entity-provider';
	import MapLayerManager from '@mssfoobar/gis-web-sdk/layer-manager';
	// Types come from the barrel by name — `Gis.*` only resolves inside the SDK's own
	// build, so a consumer writing `Gis.MapEntity` gets "Cannot find namespace 'Gis'".
	import type {
		CameraView,
		GisMap,
		MapEntity,
		MapEventSubscriptions
	} from '@mssfoobar/gis-web-sdk';

	import { sinceLabel } from '$lib/aoh/dispatch/format';
	import { FOCUS_BASE_CLASS, FOCUS_ON_CLASS } from '$lib/aoh/dispatch/workshop';
	import type { FieldUnit } from '$lib/aoh/dispatch/types';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const units = $derived(data.units);
	const positioned = $derived(units.filter((u) => u.position));
	const notShown = $derived(units.length - positioned.length);
	const canWrite = $derived(data.canWrite);

	// --- the live feed ------------------------------------------------------------- //

	/*
	 * The SDK subscribes only when all four RTUS fields are present; any missing and it
	 * renders with no feed, no error and no log. That is the "not configured" half of the
	 * degraded state, and it is knowable up front.
	 *
	 * "Configured but unreachable" is not directly observable — the SDK exposes no
	 * connection state — so it is inferred from its only honest signal: entities that
	 * never arrive. Once the grace period has passed with an empty entity store while the
	 * roster says units *do* have positions, the feed is not delivering, whatever the
	 * reason. A tenant with no positioned units gets the empty state instead, not this.
	 */
	const rtusConfigured = $derived(
		Boolean(data.rtus.sehUrl && data.rtus.mapName && data.rtus.userId && data.rtus.tenantId)
	);
	const CONNECT_GRACE_MS = 8000;
	let graceElapsed = $state(false);
	onMount(() => {
		const id = setTimeout(() => (graceElapsed = true), CONNECT_GRACE_MS);
		return () => clearTimeout(id);
	});

	let gisMap = $state<GisMap | undefined>(undefined);
	const entityCount = $derived(gisMap?.state?.entities?.size ?? 0);
	const feedUnavailable = $derived(
		!rtusConfigured || (graceElapsed && entityCount === 0 && positioned.length > 0)
	);

	// --- camera -------------------------------------------------------------------- //

	/** Singapore, where the seeded roster works. Used when nothing else is known. */
	const FALLBACK_VIEW: CameraView = { position: [103.8519, 1.2903], zoom: 12 };

	// Read once at init, like every other <Map> prop.
	const initialCameraView: CameraView = (() => {
		const withPosition = data.units.filter((u) => u.position);
		if (withPosition.length === 0) return FALLBACK_VIEW;
		const lon = withPosition.reduce((sum, u) => sum + u.position!.lon, 0) / withPosition.length;
		const lat = withPosition.reduce((sum, u) => sum + u.position!.lat, 0) / withPosition.length;
		return { position: [lon, lat], zoom: 13 };
	})();

	function flyTo(unit: FieldUnit | undefined) {
		// An un-positioned unit leaves the camera exactly where it is — moving it would
		// imply a location the unit has not reported.
		if (!unit?.position) return;
		gisMap?.engine?.fly_to?.({ position: [unit.position.lon, unit.position.lat], zoom: 15 });
	}

	// --- selection ------------------------------------------------------------------ //

	// The console hands a selection over as `?unit=` — a SvelteKit navigation, not a fetch.
	let selectedCode = $state<string | undefined>(page.url.searchParams.get('unit') ?? undefined);
	// Workshop exercise 3 is built here; the units page points at it with ?focus=3.
	const workshopFocus = $derived(page.url.searchParams.get('focus') === '3');
	const incidents = $derived(units.filter((u) => u.assignment));
	const selected = $derived(units.find((u) => u.id === selectedCode));

	/*
	 * Resolve a clicked id to a unit code. Our projection writes `entity_id = unit_code`,
	 * so a direct hit on the roster is the common case; the entity-store lookup is the
	 * fallback for when the engine hands back the GIS uuid instead. No fetch either way —
	 * the entity store is already the single source of entity state.
	 */
	function resolveUnitCode(id: string | undefined): string | undefined {
		if (!id) return undefined;
		if (units.some((u) => u.id === id)) return id;
		return gisMap?.state?.entities?.get(id)?.entity_id;
	}

	// Read once at init, so it must be a stable object.
	const eventSubscriptions: MapEventSubscriptions = {
		on_click: ({ map_entity_ids }) => {
			// Both id arrays are empty for a click on empty space, which clears the panel.
			selectedCode = resolveUnitCode(map_entity_ids?.[0]);
		}
	};

	// Fly to the unit the console handed over, once the engine exists. Guarded so it
	// happens on arrival rather than on every camera change afterwards.
	let cameraHandled = $state(false);
	$effect(() => {
		if (cameraHandled || !gisMap?.engine?.fly_to) return;
		const fromUrl = page.url.searchParams.get('unit');
		if (!fromUrl) {
			cameraHandled = true;
			return;
		}
		const unit = units.find((u) => u.id === fromUrl);
		if (!unit) return;
		cameraHandled = true;
		flyTo(unit);
	});

	const OSM_URL = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png';
</script>

<svelte:head>
	<title>Dispatch map</title>
</svelte:head>

<div class="flex h-dvh flex-col gap-3 p-5">
	<header class="flex flex-wrap items-end justify-between gap-3">
		<div>
			<h1 class="text-lg leading-6 font-semibold tracking-tight">Dispatch map</h1>
			{#if data.state === 'ok'}
				<p class="text-xs text-muted-foreground tabular-nums">
					{positioned.length} of {units.length} units on the map
					{#if notShown > 0}
						· {notShown} not shown
					{/if}
				</p>
			{:else}
				<p class="text-xs text-muted-foreground">Field units on a map.</p>
			{/if}
		</div>

		<div class="flex flex-wrap items-center gap-2">
			{#if data.state === 'ok'}
				<!-- Derived from entities actually arriving, not from configuration alone. -->
				{#if feedUnavailable}
					<Badge variant="soft" color="warning">Not live</Badge>
				{:else}
					<Badge variant="soft" color="success">Live</Badge>
				{/if}
			{/if}
			<Button variant="outline" size="sm" href="/aoh/dispatch/units" class="gap-1.5 text-xs">
				<Truck class="size-3.5" aria-hidden="true" />
				Console
			</Button>
		</div>
	</header>

	{#if data.state === 'denied'}
		<!-- A refusal, not an outage. No retry: it will never succeed for these roles. -->
		<Card>
			<CardHeader>
				<CardTitle class="flex items-center gap-2">
					<Lock class="size-5" aria-hidden="true" />
					Access restricted
				</CardTitle>
			</CardHeader>
			<CardContent>
				<p class="text-sm text-muted-foreground" role="alert">
					Access to this action is restricted. For assistance with access, please contact your
					administrator.
				</p>
			</CardContent>
		</Card>
	{:else if data.state === 'unavailable'}
		<Card class="border-destructive">
			<CardHeader>
				<CardTitle class="flex items-center gap-2 text-destructive">
					<TriangleAlert class="size-5" aria-hidden="true" />
					Units unavailable
				</CardTitle>
			</CardHeader>
			<CardContent>
				<p class="text-sm text-muted-foreground" role="alert">
					Could not reach the dispatch service. The map cannot say how many units are missing from
					it.
				</p>
			</CardContent>
		</Card>
	{:else}
		{#if feedUnavailable}
			<!-- Operational register, no host name and no exception text. -->
			<Card class="border-warning">
				<CardContent class="flex items-center gap-2 py-3">
					<TriangleAlert class="size-4 shrink-0 text-warning" aria-hidden="true" />
					<p class="text-sm" role="status">
						Live positions are unavailable. The map shows its base layers only.
					</p>
				</CardContent>
			</Card>
		{/if}

		<div class="relative min-h-0 flex-1 overflow-hidden rounded-md border">
			<CesiumMapEngineProvider>
				<GisMapComponent
					bind:map={gisMap}
					initial_camera_view={initialCameraView}
					rtus_seh_url={data.rtus.sehUrl ? new URL(data.rtus.sehUrl) : undefined}
					rtus_map_name={data.rtus.mapName}
					user_id={data.rtus.userId}
					tenant_id={data.rtus.tenantId}
					batchUpdateInterval={500}
					event_subscriptions={eventSubscriptions}
				>
					<MapBaseLayerProvider layer_name="OpenStreetMap">
						<MapXyzSourceProvider url={OSM_URL} />
					</MapBaseLayerProvider>

					<MapEntityLayerProvider layer_name="Field units">
						<MapEntityProvider kind="field-unit">
							{#snippet children(entity: MapEntity<{ call_sign?: string; status?: string }>)}
								<!--
									Render-pure: the snippet runs inside a reactive render context, so it
									must not accumulate or mutate state (that throws
									`state_unsafe_mutation` and takes the whole layer down). An event
									handler is fine — it runs on click, not during render.
								-->
								{#if entity.geojson.type === 'Feature'}
									<button
										type="button"
										class="pointer-events-auto rounded-md bg-primary px-1.5 py-0.5 font-mono text-[11px] leading-4 font-semibold text-primary-foreground shadow ring-2 ring-background"
										aria-label={`Field unit ${entity.geojson.properties.call_sign ?? entity.entity_id}`}
										onclick={() => (selectedCode = entity.entity_id)}
									>
										{entity.geojson.properties.call_sign ?? entity.entity_id}
									</button>
								{/if}
							{/snippet}
						</MapEntityProvider>
					</MapEntityLayerProvider>

					<MapLayerManager>
						<Button variant="outline" size="sm" class="gap-1.5 text-xs">
							<Layers class="size-3.5" aria-hidden="true" />
							Layers
						</Button>
					</MapLayerManager>
				</GisMapComponent>
			</CesiumMapEngineProvider>

			{#if workshopFocus}
				<!-- Workshop exercise 3: a sketch of the incidents layer and its panel. -->
				<div class="pointer-events-none absolute top-3 right-3 z-10 max-w-xs">
					<div
						class="workshop-sketch pointer-events-auto rounded-md border border-dashed border-(--workshop) bg-(--workshop-muted) p-3 text-(--workshop-text) {FOCUS_BASE_CLASS} {FOCUS_ON_CLASS}"
					>
						<p class="text-xs font-semibold tracking-wide uppercase">
							Incidents ({incidents.length})
						</p>
						<div
							class="mt-2 h-16 w-full rounded-sm border border-dashed border-(--workshop) bg-(--workshop)/10"
						></div>
						<p class="mt-2 text-xs leading-5">
							Markers for each incident, and a panel with a placeholder picture when one is
							highlighted. Exercise 3.
						</p>
					</div>
				</div>
			{/if}

			{#if units.length === 0}
				<!--
					No units at all — never seeded, or emptied by deletion. Deliberately distinct
					from "none of them has reported a position": an operator must be able to tell
					an empty tenant from an unlocated fleet.
				-->
				<div class="pointer-events-none absolute inset-0 grid place-items-center p-6">
					<Card class="pointer-events-auto max-w-sm">
						<CardContent class="flex flex-col items-center gap-2 py-6 text-center">
							<MapPinOff class="size-7 text-muted-foreground" aria-hidden="true" />
							<p class="text-sm">No units in this tenant.</p>
							<p class="text-xs text-muted-foreground">There is nothing to place on the map yet.</p>
						</CardContent>
					</Card>
				</div>
			{:else if positioned.length === 0}
				<div class="pointer-events-none absolute inset-0 grid place-items-center p-6">
					<Card class="pointer-events-auto max-w-sm">
						<CardContent class="flex flex-col items-center gap-2 py-6 text-center">
							<MapPinOff class="size-7 text-muted-foreground" aria-hidden="true" />
							<p class="text-sm">No unit has reported a position.</p>
							<p class="text-xs text-muted-foreground">
								All {units.length} units are in the roster; none of them can be drawn.
							</p>
						</CardContent>
					</Card>
				</div>
			{/if}

			{#if selected}
				<!-- Selection card. Read-only for everyone: the map offers no control that
				     would change a unit, whatever the operator's roles. -->
				<div class="absolute top-3 left-3 w-72 max-w-[calc(100%-1.5rem)]">
					<Card>
						<CardHeader class="px-4 py-3">
							<CardTitle class="flex items-baseline gap-2 text-sm">
								{selected.callSign}
								<span class="font-mono text-xs font-normal text-muted-foreground tabular-nums">
									{selected.id}
								</span>
							</CardTitle>
						</CardHeader>
						<Separator />
						<CardContent class="space-y-2 px-4 py-3">
							<p class="text-xs text-muted-foreground">
								{selected.unitType} · {selected.status}
							</p>
							{#if selected.position}
								<p class="font-mono text-xs tabular-nums">
									{selected.position.lat.toFixed(5)}, {selected.position.lon.toFixed(5)}
								</p>
								<p class="text-xs text-muted-foreground">
									Fix {sinceLabel(selected.position.at, Date.now())}
								</p>
							{:else}
								<p class="text-xs text-muted-foreground">
									This unit has no position, so it is not on the map.
								</p>
							{/if}
							<Button
								variant="outline"
								size="sm"
								href={`/aoh/dispatch/units?unit=${encodeURIComponent(selected.id)}`}
								class="w-full text-xs"
							>
								Open in console
							</Button>
							{#if !canWrite}
								<p class="text-[11px] text-muted-foreground">Read-only.</p>
							{/if}
						</CardContent>
					</Card>
				</div>
			{/if}
		</div>
	{/if}
</div>

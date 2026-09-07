<script lang="ts">
	import { Button } from "@mssfoobar/ui/button";
	import { Input } from "@mssfoobar/ui/input";
	import {
		AlertDialog,
		AlertDialogContent,
		AlertDialogHeader,
		AlertDialogTitle,
		AlertDialogDescription,
		AlertDialogFooter,
		AlertDialogCancel,
		AlertDialogAction,
	} from "@mssfoobar/ui/alert-dialog";
	import {
		DataTable,
		createCheckboxColumn,
		createActionsColumn,
		type AppColumnDef,
	} from "@mssfoobar/ui/table";
	import { Download, Filter, Plus, Search, Trash2 } from "@lucide/svelte";

	type DistributionList = {
		id: string;
		name: string;
		modified: string;
		subgroups: number;
		owner: string;
	};

	type Props = {
		onToast?: (msg: string) => void;
	};

	let { onToast }: Props = $props();

	let rows = $state<DistributionList[]>([
		{ id: "1", name: "TTSH · Ward escalations", modified: "12/12/2024 12:00:00", subgroups: 4, owner: "J. Tan" },
		{ id: "2", name: "TTSH · Pharmacy on-call", modified: "12/11/2024 09:23:14", subgroups: 2, owner: "M. Lim" },
		{ id: "3", name: "NUH · ICU triage", modified: "12/10/2024 17:45:02", subgroups: 6, owner: "S. Wong" },
		{ id: "4", name: "NUH · Cleaning critical", modified: "12/09/2024 11:12:55", subgroups: 1, owner: "K. Chua" },
		{ id: "5", name: "SGH · Security perimeter", modified: "12/08/2024 22:01:43", subgroups: 3, owner: "R. Kumar" },
		{ id: "6", name: "Cross-tenant · Major incident", modified: "12/07/2024 14:28:30", subgroups: 12, owner: "Ops Lead" },
		{ id: "7", name: "TTSH · Code blue (test)", modified: "12/05/2024 08:00:00", subgroups: 5, owner: "J. Tan" },
	]);

	let selected = $state<string[]>([]);
	let query = $state("");
	let createOpen = $state(false);
	let confirmDelOpen = $state(false);
	let newName = $state("");

	const filtered = $derived(
		rows.filter((r) => r.name.toLowerCase().includes(query.toLowerCase())),
	);

	const columns: AppColumnDef<DistributionList>[] = [
		createCheckboxColumn(),
		{ accessorKey: "name", header: "Name", sortable: true },
		{ accessorKey: "modified", header: "Last modified", sortable: true, width: 200 },
		{ accessorKey: "subgroups", header: "Subgroups", align: "right", width: 120 },
		{ accessorKey: "owner", header: "Owner", width: 160 },
		createActionsColumn<DistributionList>({
			onClickEdit: (row) => onToast?.(`Edit ${row.name}`),
			onClickDelete: (row) => {
				rows = rows.filter((r) => r.id !== row.id);
				onToast?.("Distribution list deleted");
			},
		}),
	];

	function create() {
		const name = newName.trim();
		if (!name) return;
		const id = String(
			Math.max(0, ...rows.map((r) => Number.parseInt(r.id, 10) || 0)) + 1,
		);
		rows = [
			{ id, name, modified: new Date().toLocaleString("en-US"), subgroups: 0, owner: "J. Tan" },
			...rows,
		];
		newName = "";
		createOpen = false;
		onToast?.("Distribution list created");
	}

	function deleteSelected() {
		const count = selected.length;
		rows = rows.filter((r) => !selected.includes(r.id));
		onToast?.(`${count} list${count === 1 ? "" : "s"} deleted`);
		selected = [];
		confirmDelOpen = false;
	}
</script>

<div class="space-y-6 p-6">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div class="space-y-1">
			<h1 class="font-(family-name:--font-display) text-3xl font-semibold tracking-tight">
				Distribution lists
			</h1>
			<p class="text-(--text-muted) text-sm">
				Configure distribution lists for sending message notifications.
			</p>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			{#if selected.length > 0}
				<Button variant="destructive" onclick={() => (confirmDelOpen = true)}>
					<Trash2 />Delete ({selected.length})
				</Button>
			{/if}
			<Button variant="outline"><Download />Export CSV</Button>
			<Button onclick={() => (createOpen = true)}><Plus />Create new</Button>
		</div>
	</header>

	<div class="flex flex-wrap items-center gap-3">
		<div class="relative w-80">
			<Search
				class="text-(--text-muted) pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2"
			/>
			<Input bind:value={query} placeholder="Search lists…" class="pl-9" />
		</div>
		<Button variant="outline" size="sm"><Filter />Filter</Button>
		<span class="text-(--text-muted) ml-auto text-sm">
			{selected.length} of {filtered.length} row(s) selected.
		</span>
	</div>

	<DataTable
		{columns}
		data={filtered}
		bind:selectedValues={selected}
		emptyState={{
			icon: Search,
			title: "No matches",
			content: query
				? `Nothing matches "${query}". Try a shorter search.`
				: "No distribution lists yet. Create one to start routing notifications.",
		}}
	/>
</div>

<!-- Create dialog -->
<AlertDialog bind:open={createOpen}>
	<AlertDialogContent>
		<AlertDialogHeader>
			<AlertDialogTitle>Create distribution list</AlertDialogTitle>
			<AlertDialogDescription>
				Lists group recipients across channels and tenants.
			</AlertDialogDescription>
		</AlertDialogHeader>
		<div class="space-y-2">
			<label for="list-name" class="text-sm font-medium">Name</label>
			<Input
				id="list-name"
				bind:value={newName}
				placeholder="e.g. TTSH · After-hours triage"
			/>
			<p class="text-(--text-muted) text-xs">
				Used in operator dropdowns and audit logs.
			</p>
		</div>
		<AlertDialogFooter>
			<AlertDialogCancel>Cancel</AlertDialogCancel>
			<AlertDialogAction onclick={create}>Create</AlertDialogAction>
		</AlertDialogFooter>
	</AlertDialogContent>
</AlertDialog>

<!-- Destructive confirm -->
<AlertDialog bind:open={confirmDelOpen}>
	<AlertDialogContent>
		<AlertDialogHeader>
			<AlertDialogTitle>
				Delete {selected.length} list{selected.length === 1 ? "" : "s"}?
			</AlertDialogTitle>
			<AlertDialogDescription>
				This cannot be undone. Subgroups and routing rules attached to the lists
				will be unlinked.
			</AlertDialogDescription>
		</AlertDialogHeader>
		<AlertDialogFooter>
			<AlertDialogCancel>Cancel</AlertDialogCancel>
			<AlertDialogAction
				class="bg-(--bg-error-strong) hover:bg-(--bg-error-strong)/90"
				onclick={deleteSelected}
			>
				Delete
			</AlertDialogAction>
		</AlertDialogFooter>
	</AlertDialogContent>
</AlertDialog>

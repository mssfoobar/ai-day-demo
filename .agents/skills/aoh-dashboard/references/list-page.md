# List page — dashboard management

The list route is where users browse, search, sort, favourite, bulk-delete, and link off to existing dashboards. This page **keeps SSR enabled** (unlike create/edit, which need `ssr = false` because the dashboard renderer mounts to the DOM). The server load handles auth and the first page; everything else is client-side.

## Responsibilities

| Concern | Owner |
|---|---|
| Initial page of dashboards | Server load |
| Search by name | Client (debounced refetch) |
| Sort / pagination | Client (refetch on change) |
| Favourite toggle | Client (`setFavourite`) |
| Single & bulk delete (with confirmation) | Client (`deleteDashboard`, parallel) |
| Row → detail page | Client (`goto(\`/.../\${id}\`)`) |
| "Create Dashboard" button | Link to `/.../create` |

## Server load (`+page.server.ts`)

```ts
import type { PageServerLoad } from "./$types";
import { DashboardClient } from "@mssfoobar/dash-web-sdk";

export const load: PageServerLoad = async ({ fetch, locals }) => {
  const client = new DashboardClient({ fetch });
  const sort = "name";
  const asc = true;

  const result = await client
    .listDashboards({ sort, asc, page: 1, size: 5 })
    .catch(() => null);

  // Optional: pull a tenant-admin flag from your auth provider's locals so the
  // page can conditionally show admin actions without decoding tokens client-side.
  const isAdmin = false;

  return {
    dashboards: result?.ok ? result.data : [],
    page: result?.ok ? result.page : undefined,
    sort,
    asc,
    isAdmin,
  };
};
```

Notes:
- **Pass SvelteKit's `fetch`** to the client constructor so cookies and relative URLs work correctly during SSR.
- Always **wrap in `.catch()` returning `null`** so a backend hiccup doesn't crash the page.
- Don't fetch more than you'll display initially — page size 5 is fine for the first paint. The client refetches with larger sizes after mount.
- Pull tenant-admin / role flags here on the server so the client never has to decode tokens itself.

## Client component (`+page.svelte`)

The skeleton — wire it up against whatever data table / button components your design system provides (`@mssfoobar/ui`, shadcn-svelte, your own, etc.):

```svelte
<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { toast } from "svelte-sonner";
  import { DashboardClient, type Dashboard } from "@mssfoobar/dash-web-sdk";

  let { data } = $props();
  const client = new DashboardClient();

  let dashboards: Dashboard[] = $state(data.dashboards ?? []);
  let totalDashboardCount: number = $state(data.page?.total_records ?? 0);
  let pageSize: number = $state(10);
  let currentPage: number = $state(1);
  let sort: string = $state(data.sort ?? "name");
  let isDashboardSortedAsc: boolean = $state(data.asc ?? true);
  let searchValue: string = $state("");
  let selectedDashboards: string[] = $state([]);

  async function loadDashboards(opt: { sort: string; asc: boolean; page: number; size: number }) {
    const result = await client.listDashboards({
      sort: opt.sort, asc: opt.asc, page: opt.page, size: opt.size,
      name: searchValue || undefined,
    });
    if (!result.ok) {
      toast.error("There was an error retrieving dashboards.");
      return;
    }
    dashboards = (result.data as Dashboard[]) ?? [];
    totalDashboardCount = result.page?.total_records ?? 0;
  }

  // ...favourite, delete, search, columns...

  onMount(async () => {
    await loadDashboards({ sort, asc: isDashboardSortedAsc, page: currentPage, size: pageSize });
  });
</script>
```

## Columns

If you use TanStack-table-backed data table (most shadcn-svelte ports do), columns are `ColumnDef<Dashboard>[]`. The typical layout:

```ts
import type { ColumnDef } from "@tanstack/table-core";

const columns: ColumnDef<Dashboard>[] = [
  // multi-select checkbox — see your DataTable's helper or write a small column def
  createCheckboxColumn<Dashboard>(),
  { accessorKey: "name",        header: "Name",        cell: ({ row }) => row.original.name },
  { accessorKey: "description", header: "Description", cell: ({ row }) => row.original.description },
  { id: "tags",
    header: "Tags",
    cell: ({ row }) => /* render selectedTags via a Svelte component */ row.original.tags ?? [] },
  { id: "actions",
    header: "",
    cell: ({ row }) => /* a row action menu component: favourite / edit / delete */ null },
];
```

Notes:
- Use `?edit=true` query param on the row's "Edit" target to tell the detail page to start in edit mode (e.g. `goto(\`/.../\${id}?edit=true\`)`).
- A multi-select column gives you `selectedDashboards: string[]` via two-way binding — used by the bulk delete button.

## Favourite toggle

```ts
async function onFavourite(id: string, favourite: boolean) {
  const result = await client.setFavourite({ dashboard_id: id, favourite });
  if (!result.ok) {
    toast.error(`There was an error ${favourite ? "favouriting" : "un-favouriting"} the dashboard`);
    return;
  }
  const idx = dashboards.findIndex((d) => d.id === id);
  if (idx !== -1) {
    dashboards[idx].favourite = favourite;
    dashboards = [...dashboards];  // trigger array reactivity
  }
  toast.success(`Dashboard has been ${favourite ? "added to" : "removed from"} favorite dashboards.`);
}
```

Reassign `dashboards` with a shallow spread to nudge Svelte's reactivity — mutating an element in-place isn't enough when the array is `$state`.

## Bulk delete

Pattern: open a confirmation dialog, then fire all deletes in parallel.

```ts
let isDeleteDialogOpen: boolean = $state(false);
let toBeDeletedDashboards: string[] = [];
let toBeDeletedDashboardName: string = $state("");

function onDeleteConfirm(ids: string[], name = "dashboards") {
  isDeleteDialogOpen = true;
  toBeDeletedDashboards = ids;
  toBeDeletedDashboardName = name;
}

async function onDeleteConfirmed() {
  const results = await Promise.all(
    toBeDeletedDashboards.map((id) => client.deleteDashboard({ dashboard_id: id }))
  );
  const failed = results.filter((r) => !r.ok);
  if (failed.length) {
    toast.error(`There was an error deleting the dashboard ${toBeDeletedDashboardName}`);
    return;
  }
  await loadDashboards({ sort, asc: isDashboardSortedAsc, page: currentPage, size: pageSize });
  toBeDeletedDashboards = [];
  isDeleteDialogOpen = false;
  selectedDashboards = [];
  toast.success("Dashboard(s) has been deleted successfully.", { duration: 2000 });
}
```

Show the dashboard's actual name when deleting a single one; show `"dashboards"` for bulk.

## Search

Inline search input — fire on every input or debounce it depending on traffic:

```svelte
<input
  type="search"
  placeholder="Search..."
  oninput={onSearch}
/>
```

```ts
function onSearch(e: Event) {
  searchValue = (e.target as HTMLInputElement).value;
  currentPage = 1;            // reset to first page
  selectedDashboards = [];    // clear selection
  loadDashboards({ sort, asc: isDashboardSortedAsc, page: 1, size: pageSize });
}
```

For high-traffic search, wrap with `debounce(fn, 300)` from `@mssfoobar/dash-web-sdk/renderer`.

## Pagination & sort

```ts
function onPaginationChange(page: number) {
  currentPage = page;
  loadDashboards({ sort, asc: isDashboardSortedAsc, page, size: pageSize });
  selectedDashboards = [];
}

function onPageSizeChange(size: number) {
  pageSize = size;
  currentPage = 1;
  loadDashboards({ sort, asc: isDashboardSortedAsc, page: 1, size });
}
```

Pass `manualPagination={true}` (or your data table's equivalent) so it doesn't try to slice client-side — the backend already returned the correct page.

## Favourites-only variant

A second route at `favourite/+page.svelte` shows just dashboards where `favourite === true`. It's nearly identical — pass a filter or use a dedicated endpoint, otherwise the page structure is the same.

## Things to avoid

- **Don't sort/paginate client-side.** The backend supports it via `client.listDashboards({ sort, asc, page, size })`; offloading is fast and consistent.
- **Don't fetch widget types here.** They're not needed for the list — only for create/edit.
- **Don't pre-fetch all dashboards.** The list is paginated for a reason.
- **Don't expose `client.deleteDashboard` directly to a row click.** Always go through a confirmation dialog — deletes are destructive.

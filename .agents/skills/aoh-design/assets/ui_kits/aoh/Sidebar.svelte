<script lang="ts">
	import { Sidebar as AohSidebar } from "@mssfoobar/ui/sidebar";
	import type { SidebarGroup } from "@mssfoobar/ui/sidebar";
	import {
		LayoutDashboard,
		AlertTriangle,
		Radio,
		Users,
		Workflow,
		Server,
		Building2,
		Settings,
	} from "@lucide/svelte";

	type Props = {
		/** Current route key, used to mark the matching item active. */
		active?: string;
		/** Optional override for the default AOH groups. */
		groups?: SidebarGroup[];
	};

	let { active = "dashboard", groups }: Props = $props();

	const platform: SidebarGroup = {
		label: "Platform",
		items: [
			{ title: "Dashboards", icon: LayoutDashboard, url: "/dashboard", isActive: active === "dashboard" },
			{ title: "Incidents", icon: AlertTriangle, url: "/incidents", isActive: active === "incidents" },
			{ title: "Channels", icon: Radio, url: "/channels", isActive: active === "channels" },
			{ title: "Distribution lists", icon: Users, url: "/lists", isActive: active === "lists" },
			{ title: "Workflows", icon: Workflow, url: "/workflows", isActive: active === "workflows" },
			{ title: "Assets", icon: Server, url: "/assets", isActive: active === "assets" },
		],
	};

	const tenants: SidebarGroup = {
		label: "Tenants",
		items: [
			{ title: "Tan Tock Seng", icon: Building2, url: "/t/ttsh", isActive: active === "ttsh" },
			{ title: "National University Hospital", icon: Building2, url: "/t/nuh", isActive: active === "nuh" },
			{ title: "Singapore General", icon: Building2, url: "/t/sgh", isActive: active === "sgh" },
		],
	};

	const settingsGroup: SidebarGroup = {
		label: "Workspace",
		items: [
			{ title: "Settings", icon: Settings, url: "/settings", isActive: active === "settings" },
		],
	};

	const defaultGroups: SidebarGroup[] = [platform, tenants, settingsGroup];
	const resolvedGroups = $derived(groups ?? defaultGroups);
</script>

<!--
  Sidebar deliberately does NOT include the SidebarProvider — the Provider has
  to wrap both the sidebar AND the main content so flex layout works. Use
  `AppShell.svelte` for the canonical composition, or render `<SidebarProvider>`
  yourself in `+layout.svelte`.
-->
<AohSidebar
	title="AGIL Ops Hub"
	description="Enterprise"
	groups={resolvedGroups}
	user={{ name: "J. Tan", email: "ops@agil.example", initials: "JT" }}
/>

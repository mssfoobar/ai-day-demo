<script lang="ts">
	import { Button } from "@mssfoobar/ui/button";
	import { Badge } from "@mssfoobar/ui/badge";
	import { Card, CardHeader, CardTitle, CardContent } from "@mssfoobar/ui/card";
	import { Tabs, TabsList, TabsTrigger, TabsContent } from "@mssfoobar/ui/tabs";
	import {
		AlertTriangle,
		Clock,
		Download,
		Inbox,
		Plus,
		Radio,
	} from "@lucide/svelte";

	import MetricCard from "./widgets/MetricCard.svelte";
	import LineChart from "./widgets/LineChart.svelte";
	import BarChart from "./widgets/BarChart.svelte";
	import DonutChart from "./widgets/DonutChart.svelte";

	type Incident = {
		id: string;
		title: string;
		severity: "P1" | "P2" | "P3" | "P4";
		status: "Investigating" | "Acknowledged" | "Resolved";
		owner: string;
		opened: string;
	};

	type Props = {
		onOpenIncident?: (incident: Incident) => void;
	};

	let { onOpenIncident }: Props = $props();
	let tab = $state("overview");

	const incidentSeries = [22, 25, 30, 28, 35, 40, 38, 42, 45, 41, 38, 44];
	const responseTimes = [3.2, 2.9, 3.4, 2.6, 2.4, 2.1, 2.0, 1.8, 1.7, 1.9, 1.6, 1.5];
	const responseLabels = ["M", "T", "W", "T", "F", "S", "S", "M", "T", "W", "T", "F"];

	const channelData = [
		{ name: "Email", value: 142, color: "var(--bg-info-strong)" },
		{ name: "SMS", value: 88, color: "var(--bg-success-strong)" },
		{ name: "Push", value: 64, color: "var(--bg-warning-strong)" },
		{ name: "Voice", value: 24, color: "var(--bg-error-strong)" },
	];

	const recentIncidents: Incident[] = [
		{ id: "INC-2841", title: "DB replica lag on cluster-2", severity: "P1", status: "Investigating", owner: "J. Tan", opened: "4 min ago" },
		{ id: "INC-2840", title: "Login MFA timeout — APAC POP", severity: "P2", status: "Investigating", owner: "M. Lim", opened: "18 min ago" },
		{ id: "INC-2839", title: "Webhook retries exceeded", severity: "P3", status: "Acknowledged", owner: "S. Wong", opened: "1h 12m ago" },
		{ id: "INC-2838", title: "TLS cert expires in 7 days", severity: "P4", status: "Acknowledged", owner: "Auto-routing", opened: "2h 30m ago" },
		{ id: "INC-2837", title: "Storage hot-spot on s3-archive", severity: "P3", status: "Resolved", owner: "K. Chua", opened: "Yesterday" },
	];

	const severityColor = {
		P1: "destructive",
		P2: "warning",
		P3: "info",
		P4: "default",
	} as const;

	const statusColor = {
		Investigating: "warning",
		Acknowledged: "info",
		Resolved: "success",
	} as const;
</script>

<div class="space-y-6 p-6">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div class="space-y-1">
			<h1 class="font-(family-name:--font-display) text-3xl font-semibold tracking-tight">
				Incident Management Dashboard
			</h1>
			<p class="text-(--text-muted) text-sm">
				Operational overview across all tenants · refreshed 14 s ago
			</p>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			<Button variant="secondary"><Clock />Last 24 h</Button>
			<Button variant="secondary"><Download />Export</Button>
			<Button><Plus />Create incident</Button>
		</div>
	</header>

	<Tabs bind:value={tab}>
		<TabsList>
			<TabsTrigger value="overview">Overview</TabsTrigger>
			<TabsTrigger value="incidents">Incidents</TabsTrigger>
			<TabsTrigger value="channels">Channels</TabsTrigger>
			<TabsTrigger value="audit">Audit log</TabsTrigger>
		</TabsList>

		<TabsContent value="overview" class="space-y-4 pt-4">
			<!-- KPI row -->
			<div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
				<MetricCard
					title="Open incidents"
					value="38"
					trend="+18%"
					trendDir="up"
					icon={AlertTriangle}
					accent="var(--bg-error-strong)"
				/>
				<MetricCard
					title="Mean time to ack"
					value="1m 42s"
					trend="−12%"
					trendDir="down"
					icon={Clock}
					accent="var(--bg-info-strong)"
				/>
				<MetricCard
					title="Channels operational"
					value="14/14"
					trend="100%"
					trendDir="flat"
					icon={Radio}
					accent="var(--bg-success-strong)"
				/>
				<MetricCard
					title="Notifications sent"
					value="318"
					trend="+6%"
					trendDir="up"
					icon={Inbox}
					accent="var(--bg-info-strong)"
				/>
			</div>

			<!-- Charts row -->
			<div class="grid grid-cols-1 gap-4 lg:grid-cols-12">
				<Card class="lg:col-span-8">
					<CardHeader>
						<CardTitle class="flex items-center justify-between gap-2 text-base">
							Incident volume
							<span class="text-(--text-muted) text-xs font-normal">last 12 hours</span>
						</CardTitle>
					</CardHeader>
					<CardContent class="space-y-1 pb-4">
						<LineChart data={incidentSeries} height={200} />
						<div class="text-(--text-muted) flex justify-between text-[11px]">
							<span>06:00</span>
							<span>09:00</span>
							<span>12:00</span>
							<span>15:00</span>
							<span>18:00</span>
						</div>
					</CardContent>
				</Card>

				<Card class="lg:col-span-4">
					<CardHeader>
						<CardTitle class="text-base">Notifications by channel</CardTitle>
					</CardHeader>
					<CardContent class="flex items-center gap-4 pb-4">
						<DonutChart data={channelData} />
						<div class="flex-1 space-y-2">
							{#each channelData as c (c.name)}
								<div class="flex items-center justify-between text-sm">
									<span class="flex items-center gap-2">
										<span
											class="inline-block size-2.5 rounded-sm"
											style="background-color: {c.color}"
										></span>
										{c.name}
									</span>
									<span class="text-(--text-muted) tabular-nums">{c.value}</span>
								</div>
							{/each}
						</div>
					</CardContent>
				</Card>
			</div>

			<!-- Second charts row -->
			<div class="grid grid-cols-1 gap-4 lg:grid-cols-12">
				<Card class="lg:col-span-4">
					<CardHeader>
						<CardTitle class="text-base">Mean time to ack (min)</CardTitle>
					</CardHeader>
					<CardContent class="pb-4">
						<BarChart
							data={responseTimes}
							labels={responseLabels.slice(0, responseTimes.length)}
							color="var(--bg-success-strong)"
						/>
					</CardContent>
				</Card>

				<Card class="lg:col-span-8">
					<CardHeader>
						<CardTitle class="flex items-center justify-between text-base">
							Active incidents
							<Button variant="ghost" size="sm">View all</Button>
						</CardTitle>
					</CardHeader>
					<CardContent class="px-0 pb-0">
						<table class="w-full text-sm">
							<thead class="border-(--border) text-(--text-muted) border-b text-xs uppercase tracking-wide">
								<tr>
									<th class="px-6 py-2 text-left font-semibold">ID</th>
									<th class="px-6 py-2 text-left font-semibold">Title</th>
									<th class="px-6 py-2 text-left font-semibold">Severity</th>
									<th class="px-6 py-2 text-left font-semibold">Status</th>
									<th class="px-6 py-2 text-left font-semibold">Owner</th>
									<th class="px-6 py-2 text-right font-semibold">Opened</th>
								</tr>
							</thead>
							<tbody>
								{#each recentIncidents as r (r.id)}
									<tr
										class="hover:bg-(--bg-secondary) cursor-pointer border-b border-(--border) last:border-0"
										onclick={() => onOpenIncident?.(r)}
									>
										<td class="px-6 py-3 font-mono">{r.id}</td>
										<td class="px-6 py-3">{r.title}</td>
										<td class="px-6 py-3">
											<Badge color={severityColor[r.severity]} variant="soft">
												{r.severity}
											</Badge>
										</td>
										<td class="px-6 py-3">
											<Badge color={statusColor[r.status]} variant="soft">
												{r.status}
											</Badge>
										</td>
										<td class="px-6 py-3">{r.owner}</td>
										<td class="text-(--text-muted) px-6 py-3 text-right tabular-nums">
											{r.opened}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</CardContent>
				</Card>
			</div>
		</TabsContent>

		<TabsContent value="incidents"><p class="text-(--text-muted) py-12 text-center text-sm">Incident archetype lives in MainListPage.svelte.</p></TabsContent>
		<TabsContent value="channels"><p class="text-(--text-muted) py-12 text-center text-sm">Channels surface lives in the Channels module.</p></TabsContent>
		<TabsContent value="audit"><p class="text-(--text-muted) py-12 text-center text-sm">Audit log surface is in the Audit module.</p></TabsContent>
	</Tabs>
</div>

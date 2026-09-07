<script lang="ts">
	import { Button } from "@mssfoobar/ui/button";
	import { Badge } from "@mssfoobar/ui/badge";
	import { Card, CardHeader, CardTitle, CardContent } from "@mssfoobar/ui/card";
	import { Input } from "@mssfoobar/ui/input";
	import { Avatar, AvatarFallback } from "@mssfoobar/ui/avatar";
	import { ArrowLeft, Check, Plus, Send } from "@lucide/svelte";

	type Incident = {
		id?: string;
		title?: string;
		owner?: string;
	};

	type TimelineEntry = { t: string; who: string; evt: string };

	type Props = {
		incident?: Incident;
		onBack?: () => void;
		onToast?: (msg: string) => void;
	};

	let { incident, onBack, onToast }: Props = $props();

	let status = $state<"Investigating" | "Acknowledged" | "Resolved">("Investigating");
	let comment = $state("");
	let timeline = $state<TimelineEntry[]>([
		{ t: "14:32:11", who: "Auto-routing", evt: `Incident opened from monitor alert ${incident?.id ?? "INC-2841"}.` },
		{ t: "14:32:14", who: "PagerDuty", evt: "Notified on-call: J. Tan (primary), M. Lim (secondary)." },
		{ t: "14:33:48", who: "J. Tan", evt: "Acknowledged. Investigating replica lag on cluster-2." },
		{ t: "14:36:02", who: "J. Tan", evt: "Failover to read-replica cluster-2b in progress." },
	]);

	function post() {
		const text = comment.trim();
		if (!text) return;
		const now = new Date();
		const t =
			`${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}:${String(now.getSeconds()).padStart(2, "0")}`;
		timeline = [...timeline, { t, who: "J. Tan", evt: text }];
		comment = "";
		onToast?.("Comment posted to incident timeline");
	}

	function acknowledge() {
		status = "Acknowledged";
		onToast?.("Incident acknowledged");
	}

	function resolve() {
		status = "Resolved";
		onToast?.("Incident marked resolved");
	}

	const statusColor = {
		Investigating: "warning",
		Acknowledged: "info",
		Resolved: "success",
	} as const;

	const responders = [
		{ name: "J. Tan", role: "Primary on-call", online: true, initials: "JT" },
		{ name: "M. Lim", role: "Secondary", online: true, initials: "ML" },
		{ name: "S. Wong", role: "Subject expert", online: false, initials: "SW" },
	];

	const services = [
		{ name: "database/cluster-2", health: "Degraded", color: "warning" },
		{ name: "api/identity", health: "Operational", color: "success" },
		{ name: "api/notifications", health: "Operational", color: "success" },
		{ name: "monitor/cluster-2-replica", health: "Firing", color: "destructive" },
	] as const;
</script>

<div class="space-y-6 p-6">
	<div class="flex items-center gap-3 text-sm">
		<Button variant="ghost" size="sm" onclick={onBack}>
			<ArrowLeft />Back to dashboard
		</Button>
		<span class="text-(--text-muted)">·</span>
		<span class="text-(--text-muted) font-mono">{incident?.id ?? "INC-2841"}</span>
	</div>

	<header class="flex flex-wrap items-start justify-between gap-4">
		<div class="space-y-2">
			<div class="flex flex-wrap items-center gap-3">
				<h1 class="font-(family-name:--font-display) text-3xl font-semibold tracking-tight">
					{incident?.title ?? "DB replica lag on cluster-2"}
				</h1>
				<Badge color="destructive" variant="soft">P1 · Critical</Badge>
				<Badge color={statusColor[status]} variant="soft">{status}</Badge>
			</div>
			<p class="text-(--text-muted) text-sm">
				Opened 4 min ago · Owner {incident?.owner ?? "J. Tan"} · Tenant TTSH · Service database/cluster-2
			</p>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			{#if status === "Investigating"}
				<Button variant="outline" onclick={acknowledge}>Acknowledge</Button>
			{/if}
			{#if status !== "Resolved"}
				<Button onclick={resolve}><Check />Mark resolved</Button>
			{/if}
		</div>
	</header>

	<div class="grid grid-cols-1 gap-4 lg:grid-cols-12">
		<!-- Main column -->
		<div class="space-y-4 lg:col-span-8">
			<Card>
				<CardHeader>
					<CardTitle class="text-base">Impact</CardTitle>
				</CardHeader>
				<CardContent class="grid grid-cols-1 gap-4 pb-4 sm:grid-cols-3">
					<div class="space-y-1">
						<span class="text-(--text-muted) text-xs font-semibold uppercase tracking-wide">
							Users affected
						</span>
						<div class="font-(family-name:--font-display) text-2xl font-semibold tabular-nums">
							~ 1,840
						</div>
					</div>
					<div class="space-y-1">
						<span class="text-(--text-muted) text-xs font-semibold uppercase tracking-wide">
							Services degraded
						</span>
						<div class="font-(family-name:--font-display) text-2xl font-semibold tabular-nums">
							3
						</div>
					</div>
					<div class="space-y-1">
						<span class="text-(--text-muted) text-xs font-semibold uppercase tracking-wide">
							Replica lag
						</span>
						<div class="font-(family-name:--font-display) text-2xl font-semibold tabular-nums">
							42 s
						</div>
					</div>
				</CardContent>
			</Card>

			<Card>
				<CardHeader>
					<CardTitle class="flex items-center justify-between gap-2 text-base">
						Timeline
						<span class="text-(--text-muted) text-xs font-normal">{timeline.length} events</span>
					</CardTitle>
				</CardHeader>
				<CardContent class="space-y-4 pb-4">
					<ol class="space-y-3">
						{#each timeline as e, i (i)}
							<li class="grid grid-cols-[84px_14px_1fr] gap-3">
								<span class="text-(--text-muted) pt-px font-mono text-xs">{e.t}</span>
								<span class="relative">
									<span
										class="bg-(--border) mt-1.5 inline-block size-2 rounded-full"
									></span>
									{#if i < timeline.length - 1}
										<span
											class="bg-(--border) absolute left-[3px] top-3.5 w-[2px]"
											style="bottom: -12px;"
										></span>
									{/if}
								</span>
								<div class="space-y-0.5">
									<p class="text-sm">{e.evt}</p>
									<p class="text-(--text-muted) text-xs">{e.who}</p>
								</div>
							</li>
						{/each}
					</ol>

					<form
						class="border-(--border) flex items-center gap-2 border-t pt-3"
						onsubmit={(e) => { e.preventDefault(); post(); }}
					>
						<Input
							bind:value={comment}
							placeholder="Post an update to the timeline…"
						/>
						<Button type="submit"><Send />Post</Button>
					</form>
				</CardContent>
			</Card>
		</div>

		<!-- Side column -->
		<div class="space-y-4 lg:col-span-4">
			<Card>
				<CardHeader>
					<CardTitle class="text-base">Responders</CardTitle>
				</CardHeader>
				<CardContent class="space-y-3 pb-4">
					{#each responders as r (r.name)}
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-2">
								<Avatar size="sm">
									<AvatarFallback>{r.initials}</AvatarFallback>
								</Avatar>
								<div class="leading-tight">
									<p class="text-sm font-medium">{r.name}</p>
									<p class="text-(--text-muted) text-xs">{r.role}</p>
								</div>
							</div>
							<Badge color={r.online ? "success" : "default"} variant="soft">
								{r.online ? "Online" : "Offline"}
							</Badge>
						</div>
					{/each}
					<Button variant="outline" size="sm" class="w-full">
						<Plus />Add responder
					</Button>
				</CardContent>
			</Card>

			<Card>
				<CardHeader>
					<CardTitle class="text-base">Linked services</CardTitle>
				</CardHeader>
				<CardContent class="space-y-2 pb-4">
					{#each services as s (s.name)}
						<div class="flex items-center justify-between">
							<span class="font-mono text-sm">{s.name}</span>
							<Badge color={s.color} variant="soft">{s.health}</Badge>
						</div>
					{/each}
				</CardContent>
			</Card>
		</div>
	</div>
</div>

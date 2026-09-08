<svelte:options runes={true} />

<!--
  The workshop layer: a side panel listing the exercises that are not complete.

  Opened from the amber **Exercises** button in the console header. Each card can expand
  its brief inline, and **Focus in console** hands the exercise back to the page, which
  selects a unit and pulses the placeholder where the feature is meant to be built.

  Delete this component once all three exercises are done; WORKSHOP.md is the full brief.
-->

<script lang="ts">
	import { Badge } from '@mssfoobar/ui/badge';
	import { Button } from '@mssfoobar/ui/button';
	import { ScrollArea } from '@mssfoobar/ui/scroll-area';
	import {
		Sheet,
		SheetContent,
		SheetDescription,
		SheetFooter,
		SheetHeader,
		SheetTitle
	} from '@mssfoobar/ui/sheet';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import Construction from '@lucide/svelte/icons/construction';
	import Crosshair from '@lucide/svelte/icons/crosshair';
	import {
		completedExercises,
		incompleteExercises,
		PLACEHOLDER_CLASS,
		type ExerciseNumber
	} from '../workshop';

	let {
		open = $bindable(false),
		detail = $bindable(null),
		onfocus
	}: {
		open?: boolean;
		/** The exercise whose brief is expanded, if any. */
		detail?: ExerciseNumber | null;
		/** Called when the user asks the console to focus on an exercise's placeholder. */
		onfocus: (exercise: ExerciseNumber) => void;
	} = $props();

	const todo = incompleteExercises();
	const done = completedExercises();
	const total = todo.length + done.length;

	function toggle(n: ExerciseNumber) {
		detail = detail === n ? null : n;
	}
</script>

<Sheet bind:open>
	<SheetContent side="right" class="flex w-full flex-col gap-0 p-0 sm:max-w-md">
		<SheetHeader class="px-5 pt-5 pb-3">
			<SheetTitle class="flex items-center gap-2">
				<Construction class="size-4 text-(--text-warning-strong)" aria-hidden="true" />
				Workshop exercises
			</SheetTitle>
			<SheetDescription>
				{todo.length} of {total} not complete. Each one is a feature that is stubbed in the console and
				the service, waiting to be built.
			</SheetDescription>
		</SheetHeader>

		<ScrollArea class="min-h-0 flex-1">
			<div class="space-y-3 px-5 py-2">
				{#each todo as ex (ex.number)}
					<article class="{PLACEHOLDER_CLASS} rounded-md p-3">
						<div class="flex items-start gap-3">
							<span
								class="grid size-6 shrink-0 place-items-center rounded-full bg-(--bg-warning-strong) text-xs font-bold text-(--text-on-color) tabular-nums"
							>
								{ex.number}
							</span>
							<div class="min-w-0 flex-1">
								<h3 class="text-sm font-bold">{ex.title}</h3>
								<p class="text-xs">{ex.where}</p>
							</div>
							<Badge variant="outline" color="warning">Not complete</Badge>
						</div>

						<p class="mt-2.5 text-xs">{ex.story}</p>

						<div class="mt-3 flex flex-wrap items-center gap-2">
							<Button
								size="sm"
								class="h-8 gap-1.5 bg-(--bg-warning-strong) text-xs font-semibold text-(--text-on-color) hover:bg-(--bg-warning-strong)/90"
								onclick={() => onfocus(ex.number)}
							>
								<Crosshair class="size-3.5" aria-hidden="true" />
								Focus in console
							</Button>
							<Button
								variant="ghost"
								size="sm"
								class="h-8 gap-1 text-xs text-(--text-warning-strong) hover:bg-(--bg-warning-muted-hover)"
								aria-expanded={detail === ex.number}
								onclick={() => toggle(ex.number)}
							>
								{detail === ex.number ? 'Hide brief' : 'Brief'}
								<ChevronDown
									class="size-3.5 transition-transform {detail === ex.number ? 'rotate-180' : ''}"
									aria-hidden="true"
								/>
							</Button>
						</div>

						{#if detail === ex.number}
							<div class="mt-3 space-y-3 border-t border-(--border-warning-muted) pt-3 text-xs">
								<section>
									<h4 class="mb-1 font-semibold tracking-wide uppercase">Acceptance criteria</h4>
									<ol class="list-decimal space-y-1 pl-5">
										{#each ex.criteria as criterion (criterion)}
											<li>{criterion}</li>
										{/each}
									</ol>
								</section>
								<section>
									<h4 class="mb-1 font-semibold tracking-wide uppercase">
										Service stub — answers 501 today
									</h4>
									<ul class="space-y-0.5">
										{#each ex.routes as route (route)}
											<li><code class="font-mono">{route}</code></li>
										{/each}
									</ul>
								</section>
								<p>
									Full brief, code to copy, and what is out of scope:
									<code class="font-mono">WORKSHOP.md</code> at the repo root.
								</p>
							</div>
						{/if}
					</article>
				{/each}

				{#if todo.length === 0}
					<p class="py-8 text-center text-sm text-muted-foreground">
						Everything is built. Delete the workshop layer.
					</p>
				{/if}

				{#if done.length > 0}
					<section class="pt-2">
						<h3 class="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
							Completed ({done.length})
						</h3>
						<ul class="divide-y divide-border">
							{#each done as ex (ex.number)}
								<li class="flex items-center gap-2 py-1.5 text-sm text-muted-foreground">
									<CircleCheck class="size-4 text-success" aria-hidden="true" />
									<span class="line-through">{ex.number}. {ex.title}</span>
								</li>
							{/each}
						</ul>
					</section>
				{/if}
			</div>
		</ScrollArea>

		<SheetFooter class="flex-row justify-end gap-2 border-t px-5 py-3">
			<Button type="button" variant="outline" onclick={() => (open = false)}>Close</Button>
		</SheetFooter>
	</SheetContent>
</Sheet>

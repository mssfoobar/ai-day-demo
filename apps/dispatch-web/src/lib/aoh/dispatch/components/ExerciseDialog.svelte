<svelte:options runes={true} />

<!--
  The workshop layer: a dialog listing the exercises that are not complete.

  Opened from the floating **Exercises** button at the bottom right, or by clicking any
  exercise pin. Each card can expand its brief inline, and **Focus in console** hands the
  exercise back to the page, which selects a unit and pulses the placeholder where the
  feature is meant to be built.

  Built on AlertDialog because @mssfoobar/ui ships no plain Dialog primitive; it is the
  design system's centred modal. Delete this component once all three exercises are done;
  WORKSHOP.md is the full brief.
-->

<script lang="ts">
	import { fly, slide } from 'svelte/transition';
	import {
		AlertDialog,
		AlertDialogCancel,
		AlertDialogContent,
		AlertDialogDescription,
		AlertDialogFooter,
		AlertDialogHeader,
		AlertDialogTitle
	} from '@mssfoobar/ui/alert-dialog';
	import { Badge } from '@mssfoobar/ui/badge';
	import { Button } from '@mssfoobar/ui/button';
	import { ScrollArea } from '@mssfoobar/ui/scroll-area';
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

<AlertDialog bind:open>
	<AlertDialogContent
		class="flex max-h-[85dvh] w-full flex-col gap-0 p-0 duration-200 data-[size=default]:max-w-xl data-[size=default]:sm:max-w-xl"
	>
		<AlertDialogHeader class="px-6 pt-6 pb-3 text-left">
			<AlertDialogTitle class="flex items-center gap-2">
				<Construction class="size-4 text-(--workshop-text)" aria-hidden="true" />
				Workshop exercises
			</AlertDialogTitle>
			<AlertDialogDescription>
				{todo.length} of {total} not complete. Each one is a feature that is stubbed in the console and
				the service, waiting to be built.
			</AlertDialogDescription>
		</AlertDialogHeader>

		<ScrollArea class="min-h-0 flex-1">
			<div class="space-y-3 px-6 py-2">
				{#each todo as ex, i (ex.number)}
					<!-- Staggered entrance: the cards arrive one after another when the dialog opens. -->
					<article
						in:fly={{ y: 16, duration: 280, delay: 60 + i * 70 }}
						class="{PLACEHOLDER_CLASS} rounded-md p-3 transition-transform duration-200 hover:-translate-y-0.5"
					>
						<div class="flex items-start gap-3">
							<span
								class="grid size-6 shrink-0 place-items-center rounded-full bg-(--workshop-strong) text-xs font-bold text-(--workshop-fg) tabular-nums"
							>
								{ex.number}
							</span>
							<div class="min-w-0 flex-1">
								<h3 class="text-sm font-bold">{ex.title}</h3>
								<p class="text-xs">{ex.where}</p>
							</div>
							<Badge variant="outline" class="border-(--workshop-border) text-(--workshop-text)"
								>Not complete</Badge
							>
						</div>

						<p class="mt-2.5 text-xs">{ex.story}</p>

						<div class="mt-3 flex flex-wrap items-center gap-2">
							<Button
								size="sm"
								class="h-8 gap-1.5 bg-(--workshop-strong) text-xs font-semibold text-(--workshop-fg) transition-transform hover:bg-(--workshop-strong)/90 active:scale-95"
								onclick={() => onfocus(ex.number)}
							>
								<Crosshair class="size-3.5" aria-hidden="true" />
								Focus in console
							</Button>
							<Button
								variant="ghost"
								size="sm"
								class="h-8 gap-1 text-xs text-(--workshop-text) hover:bg-(--workshop-muted-hover)"
								aria-expanded={detail === ex.number}
								onclick={() => toggle(ex.number)}
							>
								{detail === ex.number ? 'Hide brief' : 'Brief'}
								<ChevronDown
									class="size-3.5 transition-transform duration-200 {detail === ex.number
										? 'rotate-180'
										: ''}"
									aria-hidden="true"
								/>
							</Button>
						</div>

						{#if detail === ex.number}
							<div
								transition:slide={{ duration: 220 }}
								class="mt-3 space-y-3 border-t border-(--workshop-border) pt-3 text-xs"
							>
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
					<section class="pt-2" in:fly={{ y: 16, duration: 280, delay: 60 + todo.length * 70 }}>
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

		<AlertDialogFooter class="flex-row justify-end gap-2 border-t px-6 py-3">
			<AlertDialogCancel>Close</AlertDialogCancel>
		</AlertDialogFooter>
	</AlertDialogContent>
</AlertDialog>

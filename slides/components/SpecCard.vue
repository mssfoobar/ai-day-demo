<script setup lang="ts">
import { computed, inject, useId } from 'vue'
import SpecBody from './SpecBody.vue'
import { SpecFocusKey } from './specFocus'

const props = withDefaults(
  defineProps<{ name: string; caption: string; active?: boolean; lines: string[] }>(),
  { active: false },
)

const focus = inject(SpecFocusKey, null)
const bodyId = useId()

// A click anywhere in the stack takes over from the slide's own click-through.
const open = computed(() =>
  focus && focus.focused.value !== null ? focus.focused.value === props.name : props.active,
)
</script>

<template>
  <div
    class="rounded-lg overflow-hidden transition-all duration-700 ease-in-out"
    :style="
      open
        ? 'background:#ffffff;border:1px solid #e4e4e7;box-shadow:0 2px 8px rgba(0,0,0,0.08)'
        : 'background:#fafafa;border:1px solid #f4f4f5;box-shadow:none'
    "
  >
    <button
      type="button"
      class="spec-card-header flex w-full items-center justify-between px-5 py-2.5 text-left"
      :class="focus ? 'cursor-pointer' : 'cursor-default'"
      :aria-expanded="open"
      :aria-controls="bodyId"
      :disabled="!focus"
      @click="focus?.toggle(name)"
    >
      <span
        class="font-mono text-sm font-bold transition-all duration-700 ease-in-out"
        :class="open ? 'text-brand' : 'text-brand-pale'"
      >
        {{ name }}
      </span>
      <span class="text-[11px] text-faint">{{ caption }}</span>
    </button>
    <div
      :id="bodyId"
      role="region"
      class="overflow-hidden transition-all duration-700 ease-in-out"
      :style="open ? 'max-height:9.5rem;opacity:1' : 'max-height:0;opacity:0'"
    >
      <div class="px-5 pb-4 h-[9.5rem]">
        <SpecBody :lines="lines" />
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Keyboard focus needs a visible ring; a mouse click should not draw one. */
.spec-card-header:focus-visible {
  outline: 2px solid var(--brand, #d33);
  outline-offset: -2px;
  border-radius: 0.5rem;
}
</style>

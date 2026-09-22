<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{ kind?: 'people' | 'agent'; size?: number }>(), {
  kind: 'people',
  size: 26,
})

// Height follows the glyph's own viewBox aspect.
const height = computed(() =>
  Math.round(props.size * (props.kind === 'people' ? 16 / 24 : 18 / 24)),
)
</script>

<template>
  <svg
    v-if="kind === 'people'"
    :width="size"
    :height="height"
    viewBox="0 0 24 16"
    fill="currentColor"
  >
    <circle cx="8.5" cy="4.2" r="3.3" />
    <path d="M2.4 15.6c0-3.4 2.7-5.4 6.1-5.4s6.1 2 6.1 5.4z" />
    <circle cx="17.4" cy="5.4" r="2.6" />
    <path d="M15.2 15.6c0-2.7 1.7-4.3 3.7-4.3 2.1 0 4 1.4 4 4.3z" />
  </svg>

  <svg
    v-else
    :width="size"
    :height="height"
    viewBox="0 0 24 18"
    fill="none"
    stroke="currentColor"
    stroke-width="1.6"
  >
    <circle cx="12" cy="1.4" r="1.3" fill="currentColor" stroke="none" />
    <line x1="12" y1="2.4" x2="12" y2="4.8" />
    <rect x="3.8" y="4.8" width="16.4" height="11.4" rx="3.4" />
    <circle cx="9.2" cy="10.6" r="1.5" fill="currentColor" stroke="none" />
    <circle cx="14.8" cy="10.6" r="1.5" fill="currentColor" stroke="none" />
  </svg>
</template>

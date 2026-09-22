<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ lines: string[] }>()

const PREFIX = /^(#{2,4} |- \[[ x]\] |- )/
const BOLD = new Set(['## ', '### '])

const parsed = computed(() =>
  props.lines.map((line) => {
    const prefix = line.match(PREFIX)?.[1] ?? ''
    return { prefix, text: line.slice(prefix.length), bold: BOLD.has(prefix) }
  }),
)
</script>

<template>
  <div v-for="(line, i) in parsed" :key="i" class="deck-mono">
    <span v-if="line.prefix" class="text-syntax">{{ line.prefix }}</span>
    <span class="text-code" :class="{ 'font-bold': line.bold }">{{ line.text }}</span>
  </div>
</template>

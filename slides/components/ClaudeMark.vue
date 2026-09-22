<script setup lang="ts">
const props = withDefaults(defineProps<{ px?: string }>(), { px: '0.26rem' })

// '#' body, 'o' eye, '.' gap
const ART = [
  '.######.',
  '########',
  '#o####o#',
  '#o####o#',
  '########',
  '########',
  '#.#..#.#',
]

const FILL: Record<string, string> = {
  '#': '#d97757',
  o: '#000000',
  '.': 'transparent',
}

const rows = ART.map(row => [...row].map(ch => FILL[ch]))
</script>

<template>
  <div class="flex flex-col">
    <div v-for="(row, y) in rows" :key="y" class="flex">
      <div
        v-for="(fill, x) in row"
        :key="x"
        :style="{ width: props.px, height: props.px, background: fill }"
      />
    </div>
  </div>
</template>

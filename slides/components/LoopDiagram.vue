<script setup lang="ts">
import { computed } from 'vue'
import OwnerIcon from './OwnerIcon.vue'

const props = withDefaults(
  defineProps<{
    label: string
    nodes: string[]
    accent?: number | number[]
    active?: number
    variant?: 'plain' | 'brand'
    startAngle?: number
    size?: number
    radius?: number
  }>(),
  { accent: -1, active: -1, variant: 'plain', startAngle: -90, size: 300, radius: 106 },
)

const HIGHLIGHT = 'border-2 border-brand bg-white text-brand font-bold'

const TONE = {
  plain: {
    node: 'border-rule bg-paper text-muted',
    accent: 'border-2 border-ink bg-ink text-white font-bold',
    icon: 'text-ink',
    ring: '#e4e4e7',
  },
  brand: {
    node: 'border-steel bg-paper text-ink',
    accent: 'border-2 border-brand bg-brand text-white font-bold',
    icon: 'text-brand',
    ring: 'rgba(220,38,38,0.35)',
  },
}

const RAD = Math.PI / 180
const tone = computed(() => TONE[props.variant])
const mid = computed(() => props.size / 2)
const step = computed(() => 360 / props.nodes.length)
const markerId = computed(() => `arrow-${props.label.replace(/\W+/g, '-').toLowerCase()}`)

const angleAt = (i: number) => props.startAngle + i * step.value
const pointAt = (deg: number, r = props.radius) => ({
  x: mid.value + r * Math.cos(deg * RAD),
  y: mid.value + r * Math.sin(deg * RAD),
})

const accents = computed(() =>
  (Array.isArray(props.accent) ? props.accent : [props.accent]).filter(i => i >= 0),
)

const placed = computed(() =>
  props.nodes.map((label, i) => ({
    label,
    accent: accents.value.includes(i),
    highlight: i === props.active,
    icon: accents.value.includes(i),
    ...pointAt(angleAt(i)),
  })),
)

// Clear the pills at both ends of every arc.
const gap = computed(() => Math.min(step.value * 0.34, 30))

const arcs = computed(() =>
  props.nodes.map((_, i) => {
    const from = pointAt(angleAt(i) + gap.value)
    const to = pointAt(angleAt(i + 1) - gap.value)
    return `M ${from.x.toFixed(2)} ${from.y.toFixed(2)} A ${props.radius} ${props.radius} 0 0 1 ${to.x.toFixed(2)} ${to.y.toFixed(2)}`
  }),
)
</script>

<template>
  <div class="flex flex-col items-center">
    <div class="dgm-wide text-xs tracking-[0.08em] text-faint mb-5">{{ label }}</div>

    <div class="relative" :style="{ width: `${size}px`, height: `${size}px` }">
      <svg class="absolute inset-0" :viewBox="`0 0 ${size} ${size}`" :width="size" :height="size">
        <defs>
          <marker :id="markerId" markerWidth="7" markerHeight="7" refX="5.5" refY="3" orient="auto">
            <path d="M 0 0 L 6 3 L 0 6 z" :fill="tone.ring" />
          </marker>
        </defs>
        <path
          v-for="(d, i) in arcs"
          :key="i"
          :d="d"
          fill="none"
          :stroke="tone.ring"
          stroke-width="2"
          :marker-end="`url(#${markerId})`"
        />
      </svg>

      <!-- Marks every stage a person has to pass. -->
      <template v-for="node in placed" :key="`icon-${node.label}`">
        <OwnerIcon
          v-if="node.icon"
          kind="people"
          class="absolute -translate-x-1/2 -translate-y-1/2"
          :class="tone.icon"
          :style="{ left: `${node.x}px`, top: `${node.y - 34}px` }"
        />
      </template>

      <div
        v-for="node in placed"
        :key="node.label"
        class="absolute rounded-lg text-sm px-3 py-1.5 text-center whitespace-nowrap -translate-x-1/2 -translate-y-1/2"
        :class="node.highlight ? HIGHLIGHT : node.accent ? tone.accent : `border ${tone.node} font-medium`"
        :style="{ left: `${node.x}px`, top: `${node.y}px` }"
      >
        {{ node.label }}
      </div>
    </div>
  </div>
</template>

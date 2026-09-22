<script setup lang="ts">
import { computed } from 'vue'
import OwnerIcon from './OwnerIcon.vue'

interface Box {
  label: string
  w: number
  accent?: boolean
  caption?: string
  icon?: 'people' | 'agent'
}

const props = withDefaults(
  defineProps<{
    label: string
    boxes: Box[]
    loop?: { from: number; to: number; label: string }
    variant?: 'plain' | 'brand'
    labelClass?: string
  }>(),
  { variant: 'plain', labelClass: 'mt-8 mb-6' },
)

const ARROW_W = 1

const TONE = {
  plain: { box: 'border-rule bg-paper text-muted', arrow: 'text-rule', loop: '#e4e4e7', loopText: 'text-faint' },
  brand: { box: 'border-steel bg-paper text-ink', arrow: 'text-steel', loop: 'rgba(220,38,38,0.35)', loopText: 'text-brand-mute' },
}

const tone = computed(() => TONE[props.variant])
const total = computed(
  () => props.boxes.reduce((sum, b) => sum + b.w, 0) + (props.boxes.length - 1) * ARROW_W,
)

// Horizontal centre of each box, used to anchor the loop bracket.
const centres = computed(() => {
  let x = 0
  return props.boxes.map((b) => {
    const centre = x + b.w / 2
    x += b.w + ARROW_W
    return centre
  })
})

const bracket = computed(() => {
  if (!props.loop) return null
  const start = centres.value[props.loop.from]
  return { start, width: centres.value[props.loop.to] - start }
})

const hasCaptions = computed(() => props.boxes.some(b => b.caption))
const hasIcons = computed(() => props.boxes.some(b => b.icon))
const rem = (n: number) => `${n}rem`
</script>

<template>
  <div class="dgm-wide text-xs tracking-[0.08em] text-faint" :class="labelClass">{{ label }}</div>

  <div v-if="hasIcons" class="flex items-end mb-1.5" :style="{ width: rem(total) }">
    <template v-for="(box, i) in boxes" :key="box.label">
      <div v-if="i > 0" class="w-4" />
      <div class="flex justify-center" :style="{ width: rem(box.w) }">
        <OwnerIcon v-if="box.icon" :kind="box.icon" :class="box.accent ? 'text-brand' : 'text-muted'" />
      </div>
    </template>
  </div>

  <div class="flex items-center" :style="{ width: rem(total) }">
    <template v-for="(box, i) in boxes" :key="box.label">
      <div v-if="i > 0" class="text-center w-4" :class="tone.arrow">&rarr;</div>
      <div
        class="rounded-lg text-sm py-2 text-center"
        :class="box.accent ? 'border-2 border-brand bg-brand text-white font-bold' : `border ${tone.box} font-medium`"
        :style="{ width: rem(box.w) }"
      >
        {{ box.label }}
      </div>
    </template>
  </div>

  <div v-if="hasCaptions" class="flex items-start mt-1.5" :style="{ width: rem(total) }">
    <template v-for="(box, i) in boxes" :key="box.label">
      <div v-if="i > 0" class="w-4" />
      <div
        class="text-xs text-center"
        :class="box.accent ? 'font-bold text-brand' : 'text-faint'"
        :style="{ width: rem(box.w) }"
      >
        {{ box.caption }}
      </div>
    </template>
  </div>

  <template v-if="bracket">
    <div class="flex" :style="{ width: rem(total) }">
      <div :style="{ width: rem(bracket.start) }" />
      <div
        class="h-[1.4rem] rounded-b-lg"
        :style="{
          width: rem(bracket.width),
          borderLeft: `2px solid ${tone.loop}`,
          borderRight: `2px solid ${tone.loop}`,
          borderBottom: `2px solid ${tone.loop}`,
        }"
      />
    </div>
    <div class="flex" :style="{ width: rem(total) }">
      <div :style="{ width: rem(bracket.start) }" />
      <div class="text-xs text-center mt-1" :class="tone.loopText" :style="{ width: rem(bracket.width) }">
        {{ loop!.label }}
      </div>
    </div>
  </template>
</template>

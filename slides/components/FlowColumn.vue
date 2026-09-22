<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{ steps: string[]; active?: number | number[]; width?: string }>(),
  { active: -1, width: '13rem' },
)

const actives = computed(() =>
  (Array.isArray(props.active) ? props.active : [props.active]).filter(i => i >= 0),
)
</script>

<template>
  <div class="flex flex-col" :style="{ width }">
    <template v-for="(step, i) in steps" :key="step">
      <div v-if="i > 0" class="text-steel text-center leading-none my-1">&darr;</div>
      <div
        class="rounded-lg text-sm py-2 text-center"
        :class="
          actives.includes(i)
            ? 'border-2 border-brand bg-white text-brand font-bold'
            : 'border border-rule bg-paper text-muted font-medium'
        "
      >
        {{ step }}
      </div>
    </template>
  </div>
</template>

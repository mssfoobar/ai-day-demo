<script setup lang="ts">
import SpecBody from './SpecBody.vue'

defineProps<{ name: string; caption: string; active: boolean; lines: string[] }>()
</script>

<template>
  <div
    class="rounded-lg overflow-hidden transition-all duration-700 ease-in-out"
    :style="
      active
        ? 'background:#ffffff;border:1px solid #e4e4e7;box-shadow:0 2px 8px rgba(0,0,0,0.08)'
        : 'background:#fafafa;border:1px solid #f4f4f5;box-shadow:none'
    "
  >
    <div class="flex items-center justify-between px-5 py-2.5">
      <div
        class="font-mono text-sm font-bold transition-all duration-700 ease-in-out"
        :class="active ? 'text-brand' : 'text-brand-pale'"
      >
        {{ name }}
      </div>
      <div class="text-[11px] text-faint">{{ caption }}</div>
    </div>
    <div
      class="overflow-hidden transition-all duration-700 ease-in-out"
      :style="active ? 'max-height:9.5rem;opacity:1' : 'max-height:0;opacity:0'"
    >
      <div class="px-5 pb-4 h-[9.5rem]">
        <SpecBody :lines="lines" />
      </div>
    </div>
  </div>
</template>

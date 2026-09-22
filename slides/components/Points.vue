<script setup lang="ts">
withDefaults(defineProps<{ width?: string }>(), { width: '50rem' })
</script>

<template>
  <!-- Full-width wrapper keeps centring independent of a wide sibling heading. -->
  <div class="w-full">
    <div class="mt-8 text-left mx-auto" :style="{ width }">
      <slot />
    </div>
  </div>
</template>

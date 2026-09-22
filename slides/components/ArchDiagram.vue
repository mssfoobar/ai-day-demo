<script setup lang="ts">
withDefaults(defineProps<{ skills?: string[] }>(), {
  skills: () => ['Code', 'Design', 'Code review', 'Test', 'DevOps', 'Documentation'],
})
</script>

<template>
  <div class="flex flex-col w-[50rem]">
    <div class="w-[36rem] flex flex-col items-center">
      <div class="dgm-wide text-[11px] text-faint">Developer</div>
      <div class="text-steel leading-none my-1">&darr;</div>
    </div>

    <div class="flex items-center gap-4">
      <!-- Everything inside this boundary runs on the developer's machine. -->
      <div class="w-[36rem] rounded-xl border border-dashed border-steel px-4 pt-2 pb-3">
        <div class="dgm-wide text-[10px] text-faint mb-2">Developer machine</div>

        <div class="flex flex-col items-center">
          <div class="w-full rounded-lg border border-rule bg-paper px-5 py-2.5 text-center">
            <div class="text-base font-medium text-ink">CLI tool</div>
            <div class="text-xs text-muted mt-0.5">day-zero setup: prerequisites, environment, skills</div>
          </div>

          <div v-click="1" class="text-steel leading-none my-1">&darr;</div>

          <!-- Chrome is its own layer so the harness can frame contents that appeared before it. -->
          <div class="relative w-full px-5 pb-3">
            <div v-click="2" class="absolute inset-0 rounded-lg border border-rule bg-paper" />

            <div class="relative">
              <div v-click="2" class="arch-collapse pt-2.5">
                <div class="flex items-baseline justify-center gap-2">
                  <div class="text-base font-medium text-ink">Coding harness</div>
                  <div class="text-xs text-muted">swappable</div>
                </div>
              </div>

              <div v-click="1" class="rounded-lg border-2 border-brand bg-brand-wash px-4 pt-3 pb-2.5 mt-2.5">
                <div class="flex items-baseline justify-center gap-2">
                  <div class="text-sm font-bold text-brand">Skills</div>
                  <div class="text-xs text-muted">AGIL Ops Hub knowledge</div>
                </div>
                <div class="flex flex-wrap justify-center gap-1.5 mt-2">
                  <div
                    v-for="skill in skills"
                    :key="skill"
                    class="rounded border-2 border-brand bg-brand text-white text-xs font-bold px-2.5 py-1"
                  >
                    {{ skill }}
                  </div>
                </div>
              </div>

              <!-- Grey, not red: the workflow is open source, not ours. -->
              <div v-click="1" class="w-full rounded-lg border-2 border-brand bg-brand-wash px-4 py-2 mt-2">
                <div class="flex items-baseline justify-center gap-2">
                  <div class="text-sm font-bold text-brand">OpenSpec</div>
                  <div class="text-xs text-muted">spec-driven workflow</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-click="2" class="text-steel">&harr;</div>

      <div v-click="2" class="flex-1 rounded-lg border border-rule bg-paper px-4 py-4 text-center">
        <div class="text-base font-medium text-ink">AI model</div>
        <div class="text-xs text-muted mt-1 leading-relaxed">cloud provider,<br />or local and air-gapped</div>
      </div>
    </div>
  </div>
</template>

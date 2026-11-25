<!-- SPDX-FileCopyrightText: 2026 Artefactual Systems Limited -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<script setup lang="ts">
import { ref } from "vue";

import StructureTree from "@/components/StructureTree.vue";

defineProps<{ uuid: string }>();
const expanded = ref(false);
function toggle(event: Event) {
  if ((event.target as HTMLDetailsElement).open) expanded.value = true;
}
</script>

<template>
  <details class="structure-section border rounded mb-3" @toggle="toggle">
    <summary class="p-3 fw-semibold">Package structures</summary>
    <div v-if="expanded" class="structure-grid border-top p-3">
      <StructureTree :uuid="uuid" kind="aip" title="AM AIP" />
      <StructureTree :uuid="uuid" kind="ocfl" title="OCFL object" />
    </div>
  </details>
</template>

<style scoped>
summary {
  cursor: pointer;
}

.structure-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
}

@media (max-width: 991.98px) {
  .structure-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>

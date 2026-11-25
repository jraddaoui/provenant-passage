<script setup lang="ts">
import AIPUploadLocal from "@/components/AIPUploadLocal.vue";
import { useLayoutStore } from "@/stores/layout";

useLayoutStore().updateBreadcrumb([{ text: "Upload" }]);
</script>
<template>
  <div style="max-width: 920px">
    <h1 class="mb-4">Upload an AIP</h1>
    <AIPUploadLocal />
  </div>
</template>

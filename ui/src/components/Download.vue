<script setup lang="ts">
import { getPath } from "@/client";
import type { ProvenantAip } from "@/openapi-generator";

defineProps<{ aip: ProvenantAip }>();
</script>
<template>
  <a
    v-if="aip.status === 'completed' && aip.ocflObjectKey"
    class="btn btn-sm btn-outline-primary text-nowrap"
    :href="`${getPath()}/aips/${encodeURIComponent(aip.uuid)}/download`"
    download
    >Download ZIP</a
  ><span v-else class="text-body-secondary" aria-label="Download unavailable"
    >-</span
  >
</template>

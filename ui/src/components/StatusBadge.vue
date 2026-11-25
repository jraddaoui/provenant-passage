<script setup lang="ts">
import type { ProvenantAipStatusEnum } from "@/openapi-generator";

defineProps<{ status: ProvenantAipStatusEnum }>();
const classes = {
  queued: "secondary",
  processing: "primary",
  completed: "success",
  error: "danger",
  canceled: "warning",
};
</script>
<template>
  <span
    class="badge text-capitalize"
    :class="`bg-${classes[status]}-subtle text-${classes[status]}-emphasis`"
    >{{ status }}</span
  >
</template>

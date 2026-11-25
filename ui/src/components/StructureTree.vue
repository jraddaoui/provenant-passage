<!-- SPDX-FileCopyrightText: 2026 Artefactual Systems Limited -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";

import { client, errorMessage } from "@/client";
import type { AipsStructureKindEnum } from "@/openapi-generator";

const props = defineProps<{
  uuid: string;
  kind: AipsStructureKindEnum;
  title: string;
}>();
const content = ref<string | null>(null);
const loading = ref(false);
const error = ref("");
const controller = new AbortController();

async function load() {
  if (loading.value || content.value !== null) return;
  loading.value = true;
  error.value = "";
  try {
    const response = await client.aipsStructureRaw(
      { uuid: props.uuid, kind: props.kind },
      { signal: controller.signal },
    );
    const text = await response.raw.text();
    if (!controller.signal.aborted) content.value = text;
  } catch (err) {
    const message = await errorMessage(err);
    if (!controller.signal.aborted) error.value = message;
  } finally {
    loading.value = false;
  }
}

onMounted(load);
onBeforeUnmount(() => controller.abort());
</script>

<template>
  <section class="structure-tree" :aria-label="title">
    <h2 class="h6 mb-3">{{ title }}</h2>
    <p v-if="loading" class="mb-0" role="status">Loading structure...</p>
    <div v-else-if="error" class="alert alert-danger mb-0" role="alert">
      {{ error }}
      <button class="btn btn-sm btn-outline-danger ms-2" @click="load">
        Retry
      </button>
    </div>
    <pre
      v-else-if="content !== null"
      class="border rounded bg-body-tertiary p-3 mb-0"
      tabindex="0"
      :aria-label="`${title} structure`"
      >{{ content }}</pre>
  </section>
</template>

<style scoped>
.structure-tree {
  min-width: 0;
}

pre {
  max-height: 36rem;
  overflow: auto;
  font-size: 0.8125rem;
}
</style>

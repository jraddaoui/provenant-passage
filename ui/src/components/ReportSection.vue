<!-- SPDX-FileCopyrightText: 2026 Artefactual Systems Limited -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<script setup lang="ts">
import DOMPurify from "dompurify";
import { marked } from "marked";
import { computed, onBeforeUnmount, ref } from "vue";

import { client, errorMessage } from "@/client";
import type { AipsReportFormatEnum } from "@/openapi-generator";

const props = defineProps<{ uuid: string; format: AipsReportFormatEnum }>();
const content = ref<string | null>(null);
const loading = ref(false);
const error = ref("");
const controller = new AbortController();
const html = computed(() =>
  props.format === "md" && content.value !== null
    ? DOMPurify.sanitize(marked.parse(content.value, { async: false }), {
        USE_PROFILES: { html: true },
      })
    : "",
);

async function load() {
  if (loading.value || content.value !== null) return;
  loading.value = true;
  error.value = "";
  try {
    const response = await client.aipsReportRaw(
      { uuid: props.uuid, format: props.format },
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

function toggle(event: Event) {
  if ((event.target as HTMLDetailsElement).open) void load();
}

onBeforeUnmount(() => controller.abort());
</script>

<template>
  <details class="report-section border rounded mb-3" @toggle="toggle">
    <summary class="p-3 fw-semibold">
      {{ format === "json" ? "JSON report" : "Markdown report" }}
    </summary>
    <div class="border-top p-3">
      <p v-if="loading" class="mb-0" role="status">Loading report...</p>
      <div v-else-if="error" class="alert alert-danger mb-0" role="alert">
        {{ error }}
        <button class="btn btn-sm btn-outline-danger ms-2" @click="load">
          Retry
        </button>
      </div>
      <template v-else-if="content !== null">
        <pre v-if="format === 'json'" class="mb-0" tabindex="0">{{
          content
        }}</pre>
        <!-- HTML is sanitized with DOMPurify after Markdown rendering. -->
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div v-else class="report-markdown" v-html="html" />
      </template>
    </div>
  </details>
</template>

<style scoped>
summary {
  cursor: pointer;
}

pre {
  max-height: 36rem;
  overflow: auto;
}

.report-markdown {
  overflow-x: auto;
}

.report-markdown :deep(table) {
  width: 100%;
  margin-bottom: 1rem;
}

.report-markdown :deep(th),
.report-markdown :deep(td) {
  padding: 0.5rem;
  border: 1px solid var(--bs-border-color);
}

.report-markdown :deep(th) {
  background: var(--bs-tertiary-bg);
}

.report-markdown :deep(img) {
  max-width: 100%;
}

.report-markdown :deep(:last-child) {
  margin-bottom: 0;
}
</style>

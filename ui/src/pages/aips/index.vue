<script setup lang="ts">
import { onBeforeUnmount, watch } from "vue";
import { useRoute, useRouter } from "vue-router";

import Download from "@/components/Download.vue";
import Pager from "@/components/Pager.vue";
import ResultCounter from "@/components/ResultCounter.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import UUID from "@/components/UUID.vue";
import { profileLabel } from "@/profiles";
import { useAipsStore } from "@/stores/aips";
import { useLayoutStore } from "@/stores/layout";

const store = useAipsStore();
const route = useRoute();
const router = useRouter();
useLayoutStore().updateBreadcrumb([{ text: "AIPs" }]);
store.listVisible = true;
function queryOffset() {
  const page = Number(route.query.page);
  return Number.isSafeInteger(page) && page > 0 ? (page - 1) * 20 : 0;
}
void store.load(queryOffset());
watch(
  () => route.query.page,
  () => {
    if (queryOffset() !== store.page.offset) void store.load(queryOffset());
  },
);
watch(
  () => store.page.offset,
  (offset) => {
    void router.replace({
      path: "/aips",
      query: offset ? { page: String(offset / 20 + 1) } : {},
    });
  },
);
onBeforeUnmount(() => {
  store.listVisible = false;
});
</script>
<template>
  <div class="d-flex justify-content-between align-items-center mb-4 gap-3">
    <h1 class="mb-0">AIPs</h1>
    <RouterLink to="/upload" class="btn btn-primary">Upload an AIP</RouterLink>
  </div>
  <div class="d-flex justify-content-between align-items-center mb-3 gap-2">
    <p class="mb-0 text-body-secondary" aria-live="polite">
      <ResultCounter v-bind="store.page" />
    </p>
    <span class="small text-body-secondary">{{
      store.connected
        ? "Live updates connected"
        : "Reconnecting to live updates..."
    }}</span>
  </div>
  <div v-if="store.error" class="alert alert-danger" role="alert">
    {{ store.error }}
    <button class="btn btn-sm btn-outline-danger ms-2" @click="store.load()">
      Retry
    </button>
  </div>
  <p v-if="store.loading" role="status" class="small">Loading AIPs...</p>
  <div
    v-if="!store.loading && !store.error && !store.items.length"
    class="border rounded p-5 text-center"
  >
    <h2 class="h4">No AIPs yet</h2>
    <p class="text-body-secondary">
      Upload an Archivematica AIP to create your first OCFL object.
    </p>
    <RouterLink to="/upload" class="btn btn-outline-primary"
      >Upload an AIP</RouterLink
    >
  </div>
  <div v-if="store.items.length" class="table-responsive">
    <table class="table align-middle">
      <caption class="visually-hidden">
        AIP import requests, newest first
      </caption>
      <thead>
        <tr>
          <th scope="col">Name</th>
          <th scope="col">UUID</th>
          <th scope="col">Status</th>
          <th scope="col">Output contents</th>
          <th scope="col">Created</th>
          <th scope="col">Download</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="aip in store.items" :key="aip.uuid">
          <th scope="row" class="fw-normal">
            <RouterLink :to="`/aips/${aip.uuid}`">{{ aip.name }}</RouterLink>
          </th>
          <td class="small"><UUID :id="aip.uuid" /></td>
          <td><StatusBadge :status="aip.status" /></td>
          <td>{{ profileLabel(aip.profile) }}</td>
          <td class="text-nowrap small">
            <time :datetime="aip.createdAt.toISOString()">{{
              aip.createdAt.toLocaleString()
            }}</time>
          </td>
          <td><Download :aip="aip" /></td>
        </tr>
      </tbody>
    </table>
  </div>
  <Pager
    v-if="store.page.total > store.page.limit"
    v-bind="store.page"
    @page-change="(page) => store.load((page - 1) * store.page.limit)"
  />
</template>

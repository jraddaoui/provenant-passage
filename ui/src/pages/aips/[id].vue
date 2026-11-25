<script setup lang="ts">
import { onBeforeUnmount, watch } from "vue";
import { useRoute, useRouter } from "vue-router";

import Download from "@/components/Download.vue";
import ReportSection from "@/components/ReportSection.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import UUID from "@/components/UUID.vue";
import { profileLabel } from "@/profiles";
import { isActive, useAipsStore } from "@/stores/aips";
import { useLayoutStore } from "@/stores/layout";

const route = useRoute();
const router = useRouter();
const store = useAipsStore();
const layout = useLayoutStore();
watch(
  () => route.params,
  (params) => {
    void store.show(String("id" in params ? params.id : ""));
  },
  { immediate: true },
);
watch(
  () => store.detail,
  (aip) => {
    layout.updateBreadcrumb([
      { route: router.resolve("/aips"), text: "AIPs" },
      { text: aip?.name ?? "Details" },
    ]);
  },
  { immediate: true },
);
onBeforeUnmount(() => {
  store.detailID = "";
  store.detail = null;
});
</script>
<template>
  <RouterLink to="/aips" class="d-inline-block mb-3">Back to AIPs</RouterLink>
  <h1 class="mb-4">{{ store.detail?.name ?? "AIP details" }}</h1>
  <div v-if="store.detailError" class="alert alert-danger" role="alert">
    {{ store.detailError }}
    <button class="btn btn-sm btn-outline-danger" @click="store.show()">
      Retry
    </button>
  </div>
  <p v-if="store.detailLoading" role="status">Loading AIP...</p>
  <template v-if="store.detail">
    <div
      v-if="store.detail.status === 'error'"
      class="alert alert-danger"
      role="alert"
    >
      Import failed. No OCFL download is available. Check the source AIP before
      submitting it again.
    </div>
    <div
      v-else-if="store.detail.status === 'canceled'"
      class="alert alert-warning"
    >
      Import was canceled. No OCFL download is available.
    </div>
    <div
      v-else-if="isActive(store.detail)"
      class="alert alert-info"
      role="status"
    >
      {{
        store.detail.status === "queued"
          ? "Your AIP is queued for import."
          : "Your AIP is being converted."
      }}
      This page updates automatically.
    </div>
    <div v-else class="alert alert-success" role="status">
      Import completed. Your OCFL object is ready to download.
    </div>
    <dl class="row border rounded p-4 mt-4 mx-0">
      <dt class="col-sm-3 mb-2">Name</dt>
      <dd class="col-sm-9">{{ store.detail.name }}</dd>
      <dt class="col-sm-3 mb-2">UUID</dt>
      <dd class="col-sm-9"><UUID :id="store.detail.uuid" /></dd>
      <dt class="col-sm-3 mb-2">Status</dt>
      <dd class="col-sm-9"><StatusBadge :status="store.detail.status" /></dd>
      <dt class="col-sm-3 mb-2">Output contents</dt>
      <dd class="col-sm-9">{{ profileLabel(store.detail.profile) }}</dd>
      <dt class="col-sm-3 mb-2">Created</dt>
      <dd class="col-sm-9">
        <time :datetime="store.detail.createdAt.toISOString()">{{
          store.detail.createdAt.toLocaleString()
        }}</time>
      </dd>
      <dt class="col-sm-3 mb-0">Download</dt>
      <dd class="col-sm-9 mb-0"><Download :aip="store.detail" /></dd>
    </dl>
    <div v-if="store.detail.status === 'completed'">
      <ReportSection
        v-for="format in ['json', 'md'] as const"
        :key="`${store.detail.uuid}:${format}`"
        :uuid="store.detail.uuid"
        :format="format"
      />
    </div>
  </template>
</template>

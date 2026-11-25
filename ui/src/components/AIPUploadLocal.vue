<script setup lang="ts">
import "@uppy/core/css/style.css";
import "@uppy/dashboard/css/style.css";
import Dashboard from "@uppy/vue/dashboard";
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { useRouter } from "vue-router";

import { useTheme } from "@/composables/useTheme";
import type { ProvenantAipProfileEnum } from "@/openapi-generator";
import { profiles } from "@/profiles";
import { createUploader } from "@/upload";

const selected = ref<ProvenantAipProfileEnum>("files_and_metadata");
const locked = ref(false);
const error = ref("");
const router = useRouter();
const { theme } = useTheme();
const { uppy } = createUploader(
  () => selected.value,
  (value) => {
    locked.value = value;
  },
  () => {
    void router.push("/aips");
  },
);
uppy.on("upload-error", (_file, cause) => {
  error.value = cause.message;
});
uppy.on("upload", () => {
  error.value = "";
});
const dashboardOptions = computed(() => ({
  theme: theme.value,
  width: "100%",
  height: 340,
  proudlyDisplayPoweredByUppy: false,
}));
watch(theme, (value) =>
  uppy.getPlugin("Dashboard")?.setOptions({ theme: value }),
);
onBeforeUnmount(() => uppy.destroy());
</script>
<template>
  <fieldset class="mb-3" :disabled="locked">
    <legend class="h4">What should the OCFL object contain?</legend>
    <p class="text-body-secondary">
      Choose what to keep in the downloadable object.
    </p>
    <label
      v-for="(profile, index) in profiles"
      :key="profile.value"
      class="profile-option border rounded p-3 mb-2 d-flex gap-3"
      :class="{ selected: selected === profile.value }"
    >
      <input
        v-model="selected"
        class="form-check-input flex-shrink-0"
        type="radio"
        name="profile"
        :value="profile.value"
        :aria-describedby="profile.value + '-help'"
      />
      <span
        ><span class="fw-semibold">{{ profile.label }}</span
        ><span
          v-if="index === 0"
          class="badge bg-primary-subtle text-primary-emphasis ms-2"
          >Default</span
        ><span
          :id="profile.value + '-help'"
          class="d-block small text-body-secondary mt-1"
          >{{ profile.help }}</span
        ></span
      >
    </label>
  </fieldset>
  <details class="mb-4 rounded border p-3">
    <summary class="fw-semibold">What is kept in each option?</summary>
    <div class="small mt-3">
      <p>
        The first two choices create one OCFL version. The original AIP choice
        keeps the complete archive in v1 and the extracted content in v2. It
        does not add separate original checksum files to the extracted view.
      </p>
      <p>
        Original checksum files are BagIt manifests and tag manifests. Every
        option also receives OCFL's own inventories and checksums. Each
        extracted view retains the source AIP METS metadata verbatim.
      </p>
      <p class="mb-0">
        The extracted view excludes <code>submissionDocumentation/</code>,
        including transfer METS, and does not keep <code>bagit.txt</code> or
        <code>bag-info.txt</code> as separate files. The original AIP option
        retains these inside the uploaded archive.
      </p>
    </div>
  </details>
  <p>
    Choose one Archivematica AIP in <strong>.tar or .7z</strong> format. Import
    starts after upload.
  </p>
  <p v-if="locked" class="small text-body-secondary">
    Output contents are fixed for this upload and its retries. Remove the file
    to change the selection.
  </p>
  <div v-if="error" class="alert alert-danger" role="alert">
    {{ error }} <RouterLink to="/aips">Check the AIPs list</RouterLink> before
    retrying if the response was interrupted.
  </div>
  <Dashboard :uppy="uppy" :props="dashboardOptions" />
</template>
<style scoped>
.profile-option {
  cursor: pointer;
}
.profile-option.selected {
  border-color: var(--bs-primary) !important;
  background: var(--bs-primary-bg-subtle);
}
:deep(.uppy-Root),
:deep(.uppy-u-reset) {
  font-family: var(--bs-body-font-family);
}
:deep(.uppy-Dashboard-inner) {
  border-radius: var(--bs-border-radius);
  border-color: var(--bs-border-color);
  background: var(--bs-tertiary-bg);
}
</style>

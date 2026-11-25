<script setup lang="ts">
import Logo from "@/components/Logo.vue";
import { useLayoutStore } from "@/stores/layout";

useLayoutStore().updateBreadcrumb([]);
</script>
<template>
  <div class="home">
    <div class="text-primary mb-4"><Logo class="hero-logo" /></div>
    <p class="text-uppercase small fw-semibold text-body-secondary mb-2">
      Provenant Passage
    </p>
    <h1 class="display-5 mb-4">Provenant workflows,<br />on the web.</h1>
    <p class="lead mb-4">
      Its first workflow turns an Archivematica AIP into a downloadable OCFL
      object using <code>provenant-import</code>.
    </p>
    <p class="mb-4">
      Upload an AIP, choose what the output should contain, follow its progress,
      and download the resulting object.
    </p>
    <div class="d-flex gap-3 mb-5">
      <RouterLink to="/upload" class="btn btn-primary px-4"
        >Upload an AIP</RouterLink
      ><RouterLink to="/aips" class="btn btn-outline-primary"
        >View AIPs</RouterLink
      >
    </div>
    <div class="border-top pt-4 text-body-secondary">
      <h2 class="h5 text-body">An import proof of concept</h2>
      <p>
        This proof of concept uses Go, Goa, Temporal and Vue, with MySQL,
        SeaweedFS and Redis. The import workflow preserves the converter's
        profile rules and loss accounting.
      </p>
      <p>
        Conversion does not authenticate the archive's history or sign an
        attestation.
      </p>
    </div>
  </div>
</template>
<style scoped>
.home {
  max-width: 760px;
  padding-top: 1rem;
}
.hero-logo {
  width: 72px;
  height: 72px;
}
.display-5 {
  font-weight: 600;
  letter-spacing: -0.035em;
  line-height: 1.15;
}
</style>

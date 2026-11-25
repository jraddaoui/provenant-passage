<script setup lang="ts">
import IconMenu from "~icons/clarity/menu-line";

import Breadcrumb from "@/components/Breadcrumb.vue";
import Logo from "@/components/Logo.vue";
import ThemeToggle from "@/components/ThemeToggle.vue";
import { useLayoutStore } from "@/stores/layout";

const layout = useLayoutStore();
</script>
<template>
  <header class="bg-body border-bottom sticky-top">
    <nav class="navbar navbar-expand-md p-0 flex-nowrap">
      <button
        type="button"
        class="navbar-toggler btn border-0 p-3"
        data-bs-toggle="offcanvas"
        data-bs-target="#menu-offcanvas"
        aria-controls="menu-offcanvas"
        aria-label="Open navigation"
      >
        <IconMenu />
      </button>
      <button
        type="button"
        class="btn p-3 d-none d-md-block"
        :aria-label="
          (layout.sidebarCollapsed ? 'Expand' : 'Collapse') + ' navigation'
        "
        @click="layout.toggleSidebar()"
      >
        <IconMenu class="fs-4" />
      </button>
      <RouterLink
        to="/"
        class="navbar-brand m-0 me-auto d-flex align-items-center gap-2 fw-semibold text-primary"
        ><Logo /><span>Provenant Passage</span></RouterLink
      >
      <div class="ms-4 me-auto d-none d-lg-block overflow-hidden">
        <Breadcrumb />
      </div>
      <ThemeToggle compact class="mx-2" />
    </nav>
  </header>
</template>
<style scoped lang="scss">
header {
  min-height: $header-height;
}
.navbar {
  min-height: $header-height;
}
@media (max-width: 400px) {
  .navbar-brand {
    font-size: 1rem;
  }
}
</style>

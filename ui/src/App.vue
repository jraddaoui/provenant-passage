<script setup lang="ts">
import Header from "@/components/Header.vue";
import Sidebar from "@/components/Sidebar.vue";
</script>
<template>
  <a href="#main" class="visually-hidden-focusable skip-link"
    >Skip to content</a
  >
  <Header />
  <div class="d-flex">
    <Sidebar />
    <main id="main" class="flex-grow-1 min-w-0 p-3 p-lg-5" tabindex="-1">
      <RouterView />
    </main>
  </div>
</template>
<style scoped>
.skip-link:focus {
  position: absolute;
  z-index: 2000;
  background: var(--bs-body-bg);
  padding: 1rem;
}
</style>

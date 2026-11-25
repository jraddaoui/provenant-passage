<script setup lang="ts">
import Offcanvas from "bootstrap/js/dist/offcanvas";
import { onBeforeUnmount, onMounted, ref } from "vue";
import { useRouter } from "vue-router";

import IconAIPs from "~icons/clarity/bundle-line?width=2em&height=2em";
import IconHome from "~icons/clarity/home-line?width=2em&height=2em";
import IconUpload from "~icons/clarity/plus-circle-line?width=2em&height=2em";

import { useLayoutStore } from "@/stores/layout";

const layoutStore = useLayoutStore();
const router = useRouter();

const menuItems = [
  { route: router.resolve("/"), icon: IconHome, text: "Home", show: true },
  {
    route: router.resolve("/upload"),
    icon: IconUpload,
    text: "Upload",
    show: true,
  },
  { route: router.resolve("/aips"), icon: IconAIPs, text: "AIPs", show: true },
];

let offcanvasInstance: Offcanvas | null = null;
const offcanvas = ref<HTMLElement | null>(null);

onMounted(() => {
  if (offcanvas.value) offcanvasInstance = new Offcanvas(offcanvas.value);
});

onBeforeUnmount(() => offcanvasInstance?.dispose());

const closeOffcanvas = () => {
  if (offcanvasInstance) offcanvasInstance.hide();
};
</script>

<template>
  <div
    id="menu-offcanvas"
    ref="offcanvas"
    class="sidebar offcanvas-md offcanvas-start d-flex bg-body-tertiary"
    :class="layoutStore.sidebarCollapsed ? 'collapsed' : ''"
    tabindex="-1"
    aria-labelledby="offcanvasLabel"
  >
    <div class="offcanvas-header">
      <h5 id="offcanvasLabel" class="offcanvas-title">Navigation</h5>
      <button
        type="button"
        class="btn-close"
        data-bs-dismiss="offcanvas"
        data-bs-target="#menu-offcanvas"
        aria-label="Close"
      />
    </div>
    <div class="offcanvas-body d-flex flex-grow-1 p-0">
      <nav
        aria-labelledby="offcanvasLabel"
        class="flex-grow-1 d-flex flex-column"
      >
        <ul class="list-unstyled flex-grow-1 mb-0">
          <li v-for="(item, i) in menuItems.filter((it) => it.show)" :key="i">
            <div
              v-if="!item.route"
              class="py-2 text-muted small"
              :class="layoutStore.sidebarCollapsed ? 'text-center' : 'ps-3'"
            >
              {{ item.text }}
            </div>
            <RouterLink
              v-else
              class="d-block py-3 text-decoration-none sidebar-link"
              active-class="active"
              exact-active-class="exact-active"
              :to="item.route"
              @click="closeOffcanvas"
            >
              <div class="container-fluid">
                <div class="row">
                  <div
                    class="d-flex p-0 col-3 justify-content-end"
                    :class="
                      layoutStore.sidebarCollapsed
                        ? 'col-md-12 justify-content-md-center'
                        : ''
                    "
                  >
                    <span aria-hidden="true">
                      <component :is="item.icon" />
                    </span>
                  </div>
                  <div
                    class="col-9 d-flex align-items-center"
                    :class="
                      layoutStore.sidebarCollapsed
                        ? 'col-md-12 justify-content-md-center pt-md-2'
                        : ''
                    "
                  >
                    {{ item.text }}
                  </div>
                </div>
              </div>
            </RouterLink>
          </li>
        </ul>
      </nav>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.sidebar-link {
  color: var(--bs-body-color);

  &.active:not(.exact-active) {
    color: var(--bs-primary);
    font-weight: $font-weight-bold;
  }

  &.exact-active {
    color: var(--bs-white);
    background-color: var(--bs-primary);
  }

  &:hover,
  &:focus {
    background-color: var(--bs-secondary-bg);

    &.exact-active {
      background-color: color-mix(in srgb, var(--bs-primary) 75%, black);
    }
  }
}

@include color-mode(dark) {
  .sidebar-link.active:not(.exact-active) {
    color: var(--bs-emphasis-color);
  }
}

@media (min-width: 768px) {
  .sidebar {
    position: sticky;
    top: $header-height;
    height: calc(100vh - $header-height);
    overflow-y: auto;
    overflow-x: hidden;
    border-right: var(--bs-border-width) $border-style var(--bs-border-color);
    width: $sidebar-width;
    min-width: $sidebar-width;

    &.collapsed {
      width: $sidebar-collapsed-width;
      min-width: $sidebar-collapsed-width;

      [class^="col-"] {
        max-width: $sidebar-collapsed-width;
      }

      .sidebar-link .col-9 {
        font-size: 0.75 * $font-size-base;
      }
    }
  }
}
</style>

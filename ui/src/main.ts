import { createPinia } from "pinia";
import { createApp } from "vue";

import App from "./App.vue";
import { getPath } from "./client";
import { useTheme } from "./composables/useTheme";
import { MonitorConnection } from "./monitor";
import router from "./router";
import { useAipsStore } from "./stores/aips";
import "./styles/main.scss";

useTheme();
const app = createApp(App);
app.use(createPinia());
app.use(router);
const store = useAipsStore();
const monitor = new MonitorConnection(
  getPath(),
  (hint) => {
    void store.handleEvent(hint);
  },
  () => {
    store.connected = true;
    void store.refresh();
  },
  () => {
    store.connected = false;
  },
);
monitor.dial();
const poll = setInterval(() => {
  void store.refreshActive();
}, 5000);
const cleanup = () => {
  monitor.close();
  clearInterval(poll);
};
window.addEventListener("pagehide", cleanup, { once: true });
app.onUnmount(cleanup);
if (import.meta.hot)
  import.meta.hot.dispose(() => {
    cleanup();
    window.removeEventListener("pagehide", cleanup);
    app.unmount();
  });
app.mount("#app");

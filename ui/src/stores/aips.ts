import { acceptHMRUpdate, defineStore } from "pinia";

import { client, errorMessage } from "@/client";
import type { AipHint } from "@/monitor";
import type { ProvenantAip } from "@/openapi-generator";

export const isActive = (aip: ProvenantAip) =>
  aip.status === "queued" || aip.status === "processing";

export const useAipsStore = defineStore("aips", {
  state: () => ({
    items: [] as ProvenantAip[],
    page: { offset: 0, limit: 20, total: 0 },
    listVisible: false,
    loading: false,
    error: "",
    listRequest: 0,
    detail: null as ProvenantAip | null,
    detailID: "",
    detailLoading: false,
    detailError: "",
    detailRequest: 0,
    connected: false,
  }),
  actions: {
    async load(offset?: number): Promise<void> {
      offset ??= this.page.offset;
      const request = ++this.listRequest;
      this.page.offset = offset;
      this.loading = true;
      try {
        const result = await client.aipsList({
          offset,
          limit: this.page.limit,
        });
        if (request !== this.listRequest) return;
        if (result.page.total > 0 && offset >= result.page.total)
          return this.load(0);
        this.items = result.items;
        this.page = result.page;
        this.error = "";
      } catch (error) {
        const message = await errorMessage(error);
        if (request === this.listRequest) this.error = message;
      } finally {
        if (request === this.listRequest) this.loading = false;
      }
    },
    async show(uuid?: string): Promise<void> {
      uuid ??= this.detailID;
      if (!uuid) return;
      const request = ++this.detailRequest;
      if (this.detailID !== uuid) this.detail = null;
      this.detailID = uuid;
      this.detailLoading = true;
      try {
        const result = await client.aipsShow({ uuid });
        if (request === this.detailRequest && uuid === this.detailID) {
          this.detail = result;
          this.detailError = "";
        }
      } catch (error) {
        const message = await errorMessage(error);
        if (request === this.detailRequest && uuid === this.detailID)
          this.detailError = message;
      } finally {
        if (request === this.detailRequest) this.detailLoading = false;
      }
    },
    async handleEvent(hint: AipHint) {
      await Promise.all([
        this.listVisible
          ? this.load(hint.type === "aip_created_event" ? 0 : this.page.offset)
          : undefined,
        this.detailID === hint.uuid ? this.show() : undefined,
      ]);
    },
    async refresh() {
      await Promise.all([
        this.listVisible ? this.load() : undefined,
        this.detailID ? this.show() : undefined,
      ]);
    },
    async refreshActive() {
      await Promise.all([
        this.listVisible &&
        !this.loading &&
        (this.error || this.items.some(isActive))
          ? this.load()
          : undefined,
        this.detailID &&
        !this.detailLoading &&
        (this.detailError || (this.detail && isActive(this.detail)))
          ? this.show()
          : undefined,
      ]);
    },
  },
});

if (import.meta.hot)
  import.meta.hot.accept(acceptHMRUpdate(useAipsStore, import.meta.hot));

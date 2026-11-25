import { createPinia, setActivePinia } from "pinia";
import { beforeEach, expect, it, vi } from "vitest";

import { client } from "@/client";
import type { ListResponseBody } from "@/openapi-generator";
import { useAipsStore } from "@/stores/aips";

vi.mock("@/client", () => ({
  client: { aipsList: vi.fn(), aipsShow: vi.fn() },
  errorMessage: async () => "Request failed",
}));
beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

it("discards an old page response when a creation resets pagination", async () => {
  let resolveOld!: (v: ListResponseBody) => void;
  vi.mocked(client.aipsList).mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        resolveOld = resolve;
      }),
  );
  vi.mocked(client.aipsList).mockResolvedValueOnce({
    items: [],
    page: { offset: 0, limit: 20, total: 21 },
  });
  const store = useAipsStore();
  store.listVisible = true;
  const old = store.load(20);
  await store.handleEvent({ type: "aip_created_event", uuid: "new" });
  resolveOld({ items: [], page: { offset: 20, limit: 20, total: 20 } });
  await old;
  expect(store.page.offset).toBe(0);
  expect(store.page.total).toBe(21);
});

it("polls active detail state to recover a missed terminal event", async () => {
  const store = useAipsStore();
  store.detailID = "one";
  const aip = {
    uuid: "one",
    name: "test.tar",
    status: "processing" as const,
    profile: "files_and_metadata" as const,
    createdAt: new Date(),
  };
  store.detail = aip;
  vi.mocked(client.aipsShow).mockResolvedValue({
    ...aip,
    status: "completed",
    ocflObjectKey: "aips/one/object.zip",
  });
  await store.refreshActive();
  expect(store.detail?.status).toBe("completed");
  await store.refreshActive();
  expect(client.aipsShow).toHaveBeenCalledTimes(1);
});

// @vitest-environment jsdom
// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0
import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, expect, it, vi } from "vitest";

import StructureSection from "@/components/StructureSection.vue";

afterEach(() => vi.unstubAllGlobals());
const uuid = "11111111-2222-4333-8444-555555555555";

async function expand(wrapper: ReturnType<typeof mount>) {
  (wrapper.get("details").element as HTMLDetailsElement).open = true;
  await wrapper.get("details").trigger("toggle");
  await flushPromises();
}

it("loads both trees only on expansion and displays literal text in separate pre elements", async () => {
  const aip = "bag/\n`-- <script>alert(1)</script>\n";
  const ocfl = "ocfl/\n`-- inventory.json\n";
  const fetch = vi
    .fn()
    .mockResolvedValueOnce(new Response(aip))
    .mockResolvedValueOnce(new Response(ocfl));
  vi.stubGlobal("fetch", fetch);
  const wrapper = mount(StructureSection, { props: { uuid } });
  expect(wrapper.get("summary").text()).toBe("Package structures");
  expect((wrapper.get("details").element as HTMLDetailsElement).open).toBe(
    false,
  );
  expect(fetch).not.toHaveBeenCalled();
  await expand(wrapper);
  expect(fetch).toHaveBeenCalledWith(
    `/api/aips/${uuid}/structures/aip`,
    expect.objectContaining({ signal: expect.any(AbortSignal) }),
  );
  expect(fetch).toHaveBeenCalledWith(
    `/api/aips/${uuid}/structures/ocfl`,
    expect.objectContaining({ signal: expect.any(AbortSignal) }),
  );
  expect(wrapper.findAll("h2").map((h) => h.text())).toEqual([
    "AM AIP",
    "OCFL object",
  ]);
  expect(wrapper.findAll("pre").map((p) => p.element.textContent)).toEqual([
    aip,
    ocfl,
  ]);
  expect(wrapper.find("script").exists()).toBe(false);
  (wrapper.get("details").element as HTMLDetailsElement).open = false;
  await wrapper.get("details").trigger("toggle");
  await expand(wrapper);
  expect(fetch).toHaveBeenCalledTimes(2);
});

it("retries a missing tree independently while retaining the other tree", async () => {
  const fetch = vi
    .fn()
    .mockResolvedValueOnce(
      new Response('{"message":"package structure not found"}', {
        status: 404,
      }),
    )
    .mockResolvedValueOnce(new Response("ocfl/\n"));
  vi.stubGlobal("fetch", fetch);
  const wrapper = mount(StructureSection, { props: { uuid } });
  await expand(wrapper);
  expect(wrapper.get('[role="alert"]').text()).toContain(
    "package structure not found",
  );
  expect(wrapper.findAll("pre")).toHaveLength(1);
  fetch.mockResolvedValueOnce(new Response("bag/\n"));
  await wrapper.get("button").trigger("click");
  await flushPromises();
  expect(wrapper.find('[role="alert"]').exists()).toBe(false);
  expect(wrapper.findAll("pre").map((p) => p.element.textContent)).toEqual([
    "bag/\n",
    "ocfl/\n",
  ]);
  expect(fetch).toHaveBeenCalledTimes(3);
});

it("shows loading for both trees and cancels both requests when leaving the AIP", async () => {
  const fetch = vi.fn().mockImplementation(() => new Promise(() => {}));
  vi.stubGlobal("fetch", fetch);
  const wrapper = mount(StructureSection, { props: { uuid } });
  await expand(wrapper);
  expect(wrapper.findAll('[role="status"]')).toHaveLength(2);
  const signals = fetch.mock.calls.map((call) => call[1].signal as AbortSignal);
  expect(signals.every((signal) => !signal.aborted)).toBe(true);
  wrapper.unmount();
  expect(signals.every((signal) => signal.aborted)).toBe(true);
});

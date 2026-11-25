// @vitest-environment jsdom
// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0
import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, expect, it, vi } from "vitest";

import ReportSection from "@/components/ReportSection.vue";

afterEach(() => vi.unstubAllGlobals());

const uuid = "11111111-2222-4333-8444-555555555555";

async function expand(wrapper: ReturnType<typeof mount>) {
  const details = wrapper.get("details");
  (details.element as HTMLDetailsElement).open = true;
  await details.trigger("toggle");
  await flushPromises();
}

it("starts collapsed, fetches JSON on expansion and displays literal text in a pre", async () => {
  const text = '{\n  "filename": "<script>alert(1)</script>"\n}\n';
  const fetch = vi.fn().mockResolvedValue(new Response(text));
  vi.stubGlobal("fetch", fetch);
  const wrapper = mount(ReportSection, { props: { uuid, format: "json" } });

  expect(wrapper.get("summary").text()).toBe("Import event record");
  expect((wrapper.get("details").element as HTMLDetailsElement).open).toBe(
    false,
  );
  expect(fetch).not.toHaveBeenCalled();
  expect(wrapper.find("pre").exists()).toBe(false);
  await expand(wrapper);

  expect(fetch).toHaveBeenCalledWith(
    `/api/aips/${uuid}/reports/json`,
    expect.objectContaining({ signal: expect.any(AbortSignal) }),
  );
  expect(wrapper.get("pre").element.textContent).toBe(text);
  expect(wrapper.find("script").exists()).toBe(false);
  (wrapper.get("details").element as HTMLDetailsElement).open = false;
  await wrapper.get("details").trigger("toggle");
  await expand(wrapper);
  expect(fetch).toHaveBeenCalledTimes(1);
});

it("renders Markdown headings, tables and code while sanitizing active HTML", async () => {
  const text =
    '# Import report\n\n| Result | Count |\n| --- | --- |\n| Preserved | 18 |\n\n`file.txt`\n\n<script>alert(1)</script>\n\n<img src=x onerror="alert(1)">\n\n[unsafe](javascript:alert%281%29)';
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(text)));
  const wrapper = mount(ReportSection, { props: { uuid, format: "md" } });
  expect(wrapper.get("summary").text()).toBe("Content preservation report");
  await expand(wrapper);
  expect(wrapper.get("h1").text()).toBe("Import report");
  expect(wrapper.get("table td").text()).toBe("Preserved");
  expect(wrapper.get("code").text()).toBe("file.txt");
  expect(
    wrapper.find("script, [onerror], a[href^='javascript:']").exists(),
  ).toBe(false);
});

it("shows loading and a retryable error without treating a missing report as content", async () => {
  let resolve!: (response: Response) => void;
  const fetch = vi.fn().mockImplementationOnce(
    () =>
      new Promise<Response>((r) => {
        resolve = r;
      }),
  );
  vi.stubGlobal("fetch", fetch);
  const wrapper = mount(ReportSection, { props: { uuid, format: "json" } });
  await expand(wrapper);
  expect(wrapper.get('[role="status"]').text()).toContain("Loading");
  resolve(
    new Response('{"message":"import report not found"}', { status: 404 }),
  );
  await flushPromises();
  expect(wrapper.get('[role="alert"]').text()).toContain(
    "import report not found",
  );
  expect(wrapper.find("pre").exists()).toBe(false);
  fetch.mockResolvedValueOnce(new Response('{"status":"success"}'));
  await wrapper.get("button").trigger("click");
  await flushPromises();
  expect(wrapper.find('[role="alert"]').exists()).toBe(false);
  expect(wrapper.get("pre").text()).toBe('{"status":"success"}');
});

it("cancels a pending report when leaving the AIP", async () => {
  const fetch = vi.fn().mockImplementation(() => new Promise(() => {}));
  vi.stubGlobal("fetch", fetch);
  const wrapper = mount(ReportSection, { props: { uuid, format: "md" } });
  await expand(wrapper);
  const signal = fetch.mock.calls[0][1].signal as AbortSignal;
  expect(signal.aborted).toBe(false);
  wrapper.unmount();
  expect(signal.aborted).toBe(true);
});

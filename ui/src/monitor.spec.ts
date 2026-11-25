import { afterEach, expect, it, vi } from "vitest";

import { MonitorConnection, parseMonitorEvent } from "@/monitor";

class FakeSource {
  static instances: FakeSource[] = [];
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  close = vi.fn();
  constructor() {
    FakeSource.instances.push(this);
  }
}

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  FakeSource.instances = [];
});

it("ignores malformed data and ping, validates AIP event identity", () => {
  for (const value of [
    "bad",
    "null",
    "{}",
    '{"value":{"type":"ping_event","value":{}}}',
    '{"value":{"type":"aip_created_event","value":{}}}',
  ])
    expect(parseMonitorEvent(value)).toBeNull();
  expect(
    parseMonitorEvent(
      JSON.stringify({
        value: {
          type: "aip_updated_event",
          value: { uuid: "a", status: "completed" },
        },
      }),
    ),
  ).toEqual({ type: "aip_updated_event", uuid: "a" });
});

it("refreshes on reconnect, accepts repeated/out-of-order hints, and stops on close", () => {
  vi.useFakeTimers();
  vi.stubGlobal("EventSource", FakeSource);
  const refresh = vi.fn();
  const changed = vi.fn();
  const monitor = new MonitorConnection("/api", changed, refresh);
  monitor.dial();
  const first = FakeSource.instances[0];
  first.onopen?.();
  const send = (status: string) =>
    first.onmessage?.({
      data: JSON.stringify({
        value: { type: "aip_updated_event", value: { uuid: "one", status } },
      }),
    });
  send("completed");
  send("completed");
  send("processing");
  expect(changed).toHaveBeenCalledTimes(3); // Hints refetch; they never apply stale status.
  first.onerror?.();
  expect(first.close).toHaveBeenCalled();
  vi.advanceTimersByTime(1500);
  expect(FakeSource.instances).toHaveLength(2);
  FakeSource.instances[1].onopen?.();
  expect(refresh).toHaveBeenCalledTimes(2);
  FakeSource.instances[1].onerror?.();
  monitor.close();
  vi.runAllTimers();
  expect(FakeSource.instances).toHaveLength(2);
});

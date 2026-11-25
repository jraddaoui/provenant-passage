// Adapted from Enduro's monitor: no authorization or additional domain streams.
// Events are invalidation hints. Database reads establish the displayed state.
export type AipHint = {
  type: "aip_created_event" | "aip_updated_event";
  uuid: string;
};

function isObject(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object";
}

export function parseMonitorEvent(raw: string): AipHint | null {
  try {
    const body: unknown = JSON.parse(raw);
    if (!isObject(body) || !isObject(body.value)) return null;
    const event = body.value;
    if (
      event.type !== "aip_created_event" &&
      event.type !== "aip_updated_event"
    )
      return null;
    if (
      !isObject(event.value) ||
      typeof event.value.uuid !== "string" ||
      !event.value.uuid
    )
      return null;
    return { type: event.type, uuid: event.value.uuid };
  } catch {
    return null;
  }
}

export class MonitorConnection {
  private source: EventSource | null = null;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private closed = false;
  private attempts = 0;

  constructor(
    private base: string,
    private changed: (hint: AipHint) => void,
    private opened: () => void,
    private disconnected: () => void = () => {},
  ) {}

  dial() {
    if (this.closed || this.source) return;
    try {
      const source = new EventSource(this.base + "/aips/monitor");
      this.source = source;
      source.onopen = () => {
        this.attempts = 0;
        this.opened();
      };
      source.onmessage = (event) => {
        const hint = parseMonitorEvent(event.data);
        if (hint) this.changed(hint);
      };
      source.onerror = () => {
        this.disconnectSource();
        this.reconnect();
      };
    } catch {
      this.reconnect();
    }
  }

  private reconnect() {
    this.disconnected();
    if (this.closed || this.timer) return;
    const delay =
      Math.min(1000 * 2 ** Math.min(this.attempts++, 5), 30000) +
      Math.random() * 500;
    this.timer = setTimeout(() => {
      this.timer = null;
      this.dial();
    }, delay);
  }

  private disconnectSource() {
    if (!this.source) return;
    this.source.onopen = null;
    this.source.onerror = null;
    this.source.onmessage = null;
    this.source.close();
    this.source = null;
  }

  close() {
    this.closed = true;
    if (this.timer) clearTimeout(this.timer);
    this.timer = null;
    this.disconnectSource();
  }
}

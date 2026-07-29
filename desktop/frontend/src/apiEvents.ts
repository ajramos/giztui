import type { Backend } from "./apiTypes";

// The Wails runtime surface we use for event streaming, plus the browser-dev
// shim. Split out of api.ts so both api.ts (real streaming) and apiMock.ts (mock
// progress events) can share `mockEmit` without a circular import.
interface WailsRuntime {
  EventsOn(name: string, cb: (...data: unknown[]) => void): () => void;
  EventsOff(name: string): void;
}

declare global {
  interface Window {
    go?: { main?: { App?: Backend } };
    runtime?: WailsRuntime;
  }
}

// In browser dev (no real Wails runtime), install a tiny event bus so mock
// backend methods can emit the same progress events the UI listens for. This is
// inert in the packaged app (a real window.runtime already exists) and never
// activates the streaming path (that also requires realBackend()).
export let mockEmit: ((name: string, data: unknown) => void) | null = null;
if (typeof window !== "undefined" && !window.runtime) {
  const listeners: Record<string, Array<(...d: unknown[]) => void>> = {};
  window.runtime = {
    EventsOn(name, cb) {
      (listeners[name] ||= []).push(cb);
      return () => {
        listeners[name] = (listeners[name] || []).filter((f) => f !== cb);
      };
    },
    EventsOff(name) {
      delete listeners[name];
    },
  };
  mockEmit = (name, data) => (listeners[name] || []).forEach((f) => f(data));
}

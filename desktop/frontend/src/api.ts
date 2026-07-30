// api.ts — the typed bridge to the Wails backend. Types live in apiTypes.ts and
// the browser dev mock in apiMock.ts; this file keeps the runtime surface:
// realBackend detection, the event-stream helpers, and the backend Proxy that
// routes to the real bindings (packaged app) or the mock (npm run dev).

export * from "./apiTypes";
import type { Backend } from "./apiTypes";
import { mockBackend } from "./apiMock";
import "./apiEvents"; // installs the browser-dev window.runtime shim (side-effect)

function realBackend(): Backend | null {
  return window.go?.main?.App ?? null;
}

export const isWails = (): boolean => realBackend() !== null;

// streamViaEvent runs a backend call that emits tokens as a Wails runtime event,
// forwarding each token to onToken and returning the final text. When the Wails
// runtime is absent (browser dev), it falls back to chunking the resolved
// result so the UI streams identically against the mock backend.
async function streamViaEvent(
  eventName: string,
  run: () => Promise<string>,
  onToken: (token: string) => void,
): Promise<string> {
  const rt = window.runtime;
  if (realBackend() && rt) {
    let acc = "";
    const off = rt.EventsOn(eventName, (...data: unknown[]) => {
      const tok = String(data[0] ?? "");
      acc += tok;
      onToken(tok);
    });
    try {
      const final = await run();
      return final || acc;
    } finally {
      if (typeof off === "function") off();
      else rt.EventsOff(eventName);
    }
  }
  // Mock streaming: run() resolves the full text, then we chunk it.
  const full = await run();
  for (const chunk of full.match(/[\s\S]{1,6}/g) ?? [full]) {
    await new Promise((r) => setTimeout(r, 35));
    onToken(chunk);
  }
  return full;
}

// summarizeStream streams an AI summary of a message. When force is true it
// bypasses the cache and regenerates the summary.
export function summarizeStream(
  id: string,
  onToken: (token: string) => void,
  force = false,
): Promise<string> {
  return streamViaEvent(
    "summary:token",
    () => backend.SummarizeStream(id, force),
    onToken,
  );
}

// threadSummaryStream streams an AI summary of a conversation.
export function threadSummaryStream(
  threadId: string,
  onToken: (token: string) => void,
): Promise<string> {
  return streamViaEvent(
    "summary:token",
    () => backend.ThreadSummaryStream(threadId),
    onToken,
  );
}

// applyPromptStream streams the result of applying a saved prompt to a message.
export function applyPromptStream(
  id: string,
  promptId: number,
  onToken: (token: string) => void,
  force = false,
): Promise<string> {
  return streamViaEvent(
    "prompt:token",
    () => backend.ApplyPromptStream(id, promptId, force),
    onToken,
  );
}

// applyBulkPromptStream streams the result of a prompt applied across messages.
export function applyBulkPromptStream(
  ids: string[],
  promptId: number,
  onToken: (token: string) => void,
): Promise<string> {
  return streamViaEvent(
    "prompt:token",
    () => backend.ApplyBulkPromptStream(ids, promptId),
    onToken,
  );
}

// chatStream streams the assistant's reply to a user message, grounded on the
// email `id` and the prior conversation for that email (kept by the backend).
export function chatStream(
  id: string,
  message: string,
  onToken: (token: string) => void,
): Promise<string> {
  return streamViaEvent("chat:token", () => backend.ChatStream(id, message), onToken);
}

// backend proxies to the real Wails bindings when present, otherwise to a mock
// so the UI is fully explorable in a normal browser during development.
export const backend: Backend = new Proxy({} as Backend, {
  get(_target, prop: keyof Backend) {
    const real = realBackend();
    if (real) {
      return (real[prop] as unknown as (...args: unknown[]) => unknown).bind(real);
    }
    return (mockBackend[prop] as unknown as (...args: unknown[]) => unknown).bind(mockBackend);
  },
});

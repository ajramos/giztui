// api.ts — a thin, typed wrapper over the Wails-bound Go backend.
//
// At runtime inside the packaged app, Wails injects window.go.main.App with all
// of App's exported methods. We call those directly rather than importing the
// generated wailsjs bindings, so this frontend also builds and runs in a plain
// browser (npm run dev) using the mock backend below.

export interface MessageSummary {
  id: string;
  threadId: string;
  subject: string;
  from: string;
  snippet: string;
  date: string;
  unread: boolean;
  labels: string[];
}

export interface MessageDetail {
  id: string;
  threadId: string;
  subject: string;
  from: string;
  to: string;
  cc: string;
  date: string;
  unread: boolean;
  labels: string[];
  plainText: string;
  html: string;
}

export interface MessageList {
  messages: MessageSummary[];
  nextPageToken: string;
}

export interface Label {
  id: string;
  name: string;
}

interface Backend {
  InitError(): Promise<string>;
  AccountEmail(): Promise<string>;
  ListInbox(pageToken: string, pageSize: number): Promise<MessageList>;
  Search(query: string, pageToken: string, pageSize: number): Promise<MessageList>;
  GetMessage(id: string): Promise<MessageDetail>;
  Archive(id: string): Promise<void>;
  Trash(id: string): Promise<void>;
  MarkRead(id: string): Promise<void>;
  MarkUnread(id: string): Promise<void>;
  ListLabels(): Promise<Label[]>;
}

declare global {
  interface Window {
    go?: { main?: { App?: Backend } };
  }
}

function realBackend(): Backend | null {
  return window.go?.main?.App ?? null;
}

export const isWails = (): boolean => realBackend() !== null;

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

// --- mock backend (browser dev only) ----------------------------------------

const mockMessages: MessageSummary[] = Array.from({ length: 24 }, (_, i) => ({
  id: `m${i}`,
  threadId: `t${i}`,
  subject: [
    "Welcome to GizTUI Desktop",
    "Your weekly digest is ready",
    "Re: Project roadmap Q3",
    "Invoice #2043 from Acme Corp",
    "Security alert: new sign-in",
    "Lunch on Friday?",
  ][i % 6],
  from: [
    "GizTUI <team@giztui.dev>",
    "Digest <digest@news.io>",
    "Ada Lovelace <ada@compute.org>",
    "Acme Billing <billing@acme.com>",
    "Google <no-reply@google.com>",
    "Grace Hopper <grace@navy.mil>",
  ][i % 6],
  snippet:
    "This is a preview of the message body shown in the inbox list so you can scan quickly…",
  date: new Date(Date.now() - i * 3600_000).toISOString(),
  unread: i % 3 === 0,
  labels: i % 4 === 0 ? ["Work"] : i % 5 === 0 ? ["Personal", "Travel"] : [],
}));

const mockBackend: Backend = {
  async InitError() {
    return "";
  },
  async AccountEmail() {
    return "you@example.com (mock)";
  },
  async ListInbox(_pageToken: string, pageSize: number) {
    return { messages: mockMessages.slice(0, pageSize || 50), nextPageToken: "" };
  },
  async Search(query: string, _pageToken: string, pageSize: number) {
    const q = query.toLowerCase();
    const filtered = mockMessages.filter(
      (m) => m.subject.toLowerCase().includes(q) || m.from.toLowerCase().includes(q),
    );
    return { messages: filtered.slice(0, pageSize || 50), nextPageToken: "" };
  },
  async GetMessage(id: string) {
    const m = mockMessages.find((x) => x.id === id) ?? mockMessages[0];
    return {
      ...m,
      to: "you@example.com",
      cc: "",
      plainText: `${m.snippet}\n\nHi there,\n\nThis is the full body of "${m.subject}" rendered in the reading pane. In the packaged Wails app this content comes straight from Gmail via the GizTUI service layer.\n\nBest,\n${m.from}`,
      html: "",
    };
  },
  async Archive() {},
  async Trash() {},
  async MarkRead() {},
  async MarkUnread() {},
  async ListLabels() {
    return [
      { id: "1", name: "Work" },
      { id: "2", name: "Personal" },
      { id: "3", name: "Travel" },
    ];
  },
};

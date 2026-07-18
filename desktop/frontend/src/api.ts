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

export interface Attachment {
  attachmentId: string;
  filename: string;
  mimeType: string;
  size: number;
  type: string;
  inline: boolean;
}

export interface Prompt {
  id: number;
  name: string;
  description: string;
  category: string;
}

export interface AccountInfo {
  id: string;
  email: string;
  displayName: string;
  active: boolean;
}

export interface DraftSummary {
  id: string;
  to: string;
  subject: string;
  snippet: string;
}

export interface DraftDetail {
  id: string;
  to: string;
  cc: string;
  subject: string;
  body: string;
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
  AIEnabled(): Promise<boolean>;
  Summarize(id: string): Promise<string>;
  SendMail(to: string, subject: string, body: string, cc: string[], bcc: string[]): Promise<void>;
  Reply(originalID: string, body: string, cc: string[]): Promise<void>;
  MessageLabelIDs(id: string): Promise<string[]>;
  ApplyLabel(messageID: string, labelID: string): Promise<void>;
  RemoveLabel(messageID: string, labelID: string): Promise<void>;
  ListAttachments(id: string): Promise<Attachment[]>;
  DownloadAttachment(messageID: string, attachmentID: string, filename: string): Promise<string>;
  OpenAttachment(path: string): Promise<void>;
  SummarizeStream(id: string): Promise<string>;
  BulkArchive(ids: string[]): Promise<void>;
  BulkTrash(ids: string[]): Promise<void>;
  BulkMarkRead(ids: string[]): Promise<void>;
  BulkMarkUnread(ids: string[]): Promise<void>;
  BulkApplyLabel(ids: string[], labelID: string): Promise<void>;
  BulkRemoveLabel(ids: string[], labelID: string): Promise<void>;
  PromptsEnabled(): Promise<boolean>;
  ListPrompts(): Promise<Prompt[]>;
  ApplyPromptStream(messageID: string, promptID: number): Promise<string>;
  ListAccounts(): Promise<AccountInfo[]>;
  SwitchAccount(id: string): Promise<void>;
  OpenGmailWeb(messageID: string): Promise<void>;
  ListDrafts(): Promise<DraftSummary[]>;
  GetDraft(draftID: string): Promise<DraftDetail>;
  SaveDraft(to: string, subject: string, body: string, cc: string[]): Promise<string>;
  UpdateDraft(draftID: string, to: string, subject: string, body: string, cc: string[]): Promise<void>;
  DeleteDraft(draftID: string): Promise<void>;
}

// Wails runtime surface we use (event streaming).
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

// summarizeStream streams an AI summary of a message.
export function summarizeStream(
  id: string,
  onToken: (token: string) => void,
): Promise<string> {
  return streamViaEvent("summary:token", () => backend.SummarizeStream(id), onToken);
}

// applyPromptStream streams the result of applying a saved prompt to a message.
export function applyPromptStream(
  id: string,
  promptId: number,
  onToken: (token: string) => void,
): Promise<string> {
  return streamViaEvent(
    "prompt:token",
    () => backend.ApplyPromptStream(id, promptId),
    onToken,
  );
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
    return mockActiveAccount === "work"
      ? "you@company.com (mock)"
      : "you@example.com (mock)";
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
    const plain = `${m.snippet}\n\nHi there,\n\nThis is the full body of "${m.subject}" rendered in the reading pane. In the packaged Wails app this content comes straight from Gmail via the GizTUI service layer.\n\nBest,\n${m.from}`;
    // Give some messages an HTML body so the HTML renderer is demonstrable.
    const html =
      Number(id.replace(/\D/g, "") || "0") % 2 === 0
        ? `<div style="font-family:Arial,sans-serif">
             <h2 style="color:#1a56db">${m.subject}</h2>
             <p>Hi there,</p>
             <p>This is a <strong>rich HTML</strong> version of the email, with
             <a href="https://example.com">a link</a>, a list:</p>
             <ul><li>First point</li><li>Second point</li><li>Third point</li></ul>
             <p style="background:#f0f4ff;padding:12px;border-radius:8px">
               A highlighted callout box rendered from the email's own inline styles.</p>
             <p>Best,<br>${m.from}</p>
           </div>`
        : "";
    return { ...m, to: "you@example.com", cc: "", plainText: plain, html };
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
  async AIEnabled() {
    return true;
  },
  async Summarize(id: string) {
    const m = mockMessages.find((x) => x.id === id) ?? mockMessages[0];
    await new Promise((r) => setTimeout(r, 500));
    return `• ${m.subject}\n• Sent by ${m.from}\n• (mock summary) In the packaged app this is generated by your configured LLM via GizTUI's AIService.`;
  },
  async SendMail() {
    await new Promise((r) => setTimeout(r, 300));
  },
  async Reply() {
    await new Promise((r) => setTimeout(r, 300));
  },
  async MessageLabelIDs(id: string) {
    return mockAppliedLabels[id] ?? ["1"];
  },
  async ApplyLabel(messageID: string, labelID: string) {
    const cur = new Set(mockAppliedLabels[messageID] ?? ["1"]);
    cur.add(labelID);
    mockAppliedLabels[messageID] = [...cur];
  },
  async RemoveLabel(messageID: string, labelID: string) {
    const cur = new Set(mockAppliedLabels[messageID] ?? ["1"]);
    cur.delete(labelID);
    mockAppliedLabels[messageID] = [...cur];
  },
  async ListAttachments(id: string) {
    if (id.endsWith("3") || id.endsWith("0")) {
      return [
        {
          attachmentId: "att1",
          filename: "invoice-2043.pdf",
          mimeType: "application/pdf",
          size: 84213,
          type: "document",
          inline: false,
        },
      ];
    }
    return [];
  },
  async DownloadAttachment(_m: string, _a: string, filename: string) {
    await new Promise((r) => setTimeout(r, 300));
    return `~/Downloads/gmail-attachments/${filename}`;
  },
  async OpenAttachment() {},
  async SummarizeStream(id: string) {
    // In the browser mock the streaming helper drives token delivery; this is
    // only the fallback that returns the full text.
    return this.Summarize(id);
  },
  async BulkArchive() {
    await new Promise((r) => setTimeout(r, 250));
  },
  async BulkTrash() {
    await new Promise((r) => setTimeout(r, 250));
  },
  async BulkMarkRead() {
    await new Promise((r) => setTimeout(r, 200));
  },
  async BulkMarkUnread() {
    await new Promise((r) => setTimeout(r, 200));
  },
  async BulkApplyLabel() {
    await new Promise((r) => setTimeout(r, 200));
  },
  async BulkRemoveLabel() {
    await new Promise((r) => setTimeout(r, 200));
  },
  async PromptsEnabled() {
    return true;
  },
  async ListPrompts() {
    return [
      { id: 1, name: "Summarize concisely", description: "3-bullet summary", category: "general" },
      { id: 2, name: "Extract action items", description: "List to-dos & owners", category: "productivity" },
      { id: 3, name: "Draft a polite reply", description: "Suggest a response", category: "compose" },
      { id: 4, name: "Translate to Spanish", description: "Translate the email", category: "language" },
    ];
  },
  async ApplyPromptStream(_id: string, promptID: number) {
    const names: Record<number, string> = {
      1: "• Key point one\n• Key point two\n• Key point three",
      2: "1. Reply to Ada by Friday (owner: you)\n2. Review the Q3 roadmap draft",
      3: "Hi,\n\nThanks for the update — Friday works for me. See you then!\n\nBest,",
      4: "Hola,\n\nEste es el cuerpo del correo traducido al español por tu LLM.",
    };
    return names[promptID] ?? "(mock prompt result)";
  },
  async ListAccounts() {
    return [
      { id: "personal", email: "you@example.com", displayName: "Personal", active: mockActiveAccount === "personal" },
      { id: "work", email: "you@company.com", displayName: "Work", active: mockActiveAccount === "work" },
    ];
  },
  async SwitchAccount(id: string) {
    await new Promise((r) => setTimeout(r, 250));
    mockActiveAccount = id;
  },
  async OpenGmailWeb() {
    /* mock: no-op (would open the system browser) */
  },
  async ListDrafts() {
    return mockDrafts;
  },
  async GetDraft(draftID: string) {
    const d = mockDrafts.find((x) => x.id === draftID) ?? mockDrafts[0];
    return {
      id: d.id,
      to: d.to,
      cc: "",
      subject: d.subject,
      body: `${d.snippet}\n\n(draft body loaded from the mock backend)`,
    };
  },
  async SaveDraft(to: string, subject: string) {
    await new Promise((r) => setTimeout(r, 200));
    const id = "draft-" + (mockDrafts.length + 1);
    mockDrafts = [{ id, to, subject, snippet: "New draft" }, ...mockDrafts];
    return id;
  },
  async UpdateDraft() {
    await new Promise((r) => setTimeout(r, 200));
  },
  async DeleteDraft(draftID: string) {
    await new Promise((r) => setTimeout(r, 200));
    mockDrafts = mockDrafts.filter((x) => x.id !== draftID);
  },
};

let mockActiveAccount = "personal";
let mockDrafts: DraftSummary[] = [
  { id: "d1", to: "ada@compute.org", subject: "Re: Project roadmap Q3", snippet: "Thanks Ada, I think we should…" },
  { id: "d2", to: "team@giztui.dev", subject: "Release notes draft", snippet: "Here's a first pass at the notes…" },
];

const mockAppliedLabels: Record<string, string[]> = {};

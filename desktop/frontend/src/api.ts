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
  // Content-ID (without <>) for inline attachments, used to resolve cid: image
  // references in the HTML body.
  contentId: string;
}

export interface Prompt {
  id: number;
  name: string;
  description: string;
  category: string;
}

export interface Link {
  index: number;
  url: string;
  text: string;
  type: string;
}

export interface PromptDetail {
  id: number;
  name: string;
  description: string;
  category: string;
  text: string;
}

export interface SavedQuery {
  id: number;
  name: string;
  query: string;
  description: string;
  category: string;
}

export interface AnalyzerInput {
  id: string;
  subject: string;
  from: string;
  snippet: string;
}

export interface PlanCategory {
  name: string;
  priority: string;
  description: string;
  action: string;
  label: string;
  messageIds: string[];
  byRule?: boolean;
  // The synthetic "read manually" bucket (AI left these to review).
  readManually?: boolean;
  // Saved prompt to run when action === "prompt" (from a deterministic rule).
  promptId?: number;
}

export interface ActionPlanResult {
  categories: PlanCategory[];
  totalAnalyzed: number;
  readManually: number;
}

export interface AnalyzerRule {
  id: number;
  text: string;
}

// Deterministic rule (:rules): action applied to messages matching `query`.
export interface DeterministicRule {
  id: number;
  query: string;
  action: string; // archive | mark_read | trash | label | prompt
  label: string;
  promptId: number;
  synced: boolean; // mirrored as a Gmail filter (☁)
  createdAt: number;
}

export interface ImportResult {
  imported: number;
  adopted: number;
  removed: number;
  unsupported: { description: string; reason: string }[];
}

export interface AutoRefreshSettings {
  enabled: boolean;
  intervalSeconds: number;
}

export interface Invite {
  isInvite: boolean;
  uid: string;
  summary: string;
  organizer: string;
  dtStart: string;
  dtEnd: string;
}

export interface UsageStat {
  name: string;
  category: string;
  usageCount: number;
}

export interface UsageStats {
  totalUsage: number;
  uniquePrompts: number;
  topPrompts: UsageStat[];
}

export interface ConfigInfo {
  configPath: string;
  logPath: string;
  account: string;
  llmProvider: string;
  llmModel: string;
  theme: string;
  obsidianOn: boolean;
  slackOn: boolean;
  autoRefresh: boolean;
  downloadPath: string;
}

export interface AccountInfo {
  id: string;
  email: string;
  displayName: string;
  active: boolean;
}

export interface ThemeColors {
  name: string;
  bg: string;
  fg: string;
  border: string;
  accent: string;
  primary: string;
  danger: string;
  warning: string;
  success: string;
  selectionBg: string;
  inputBg: string;
  unread: string;
  muted: string;
}

export interface KeyMap {
  summarize: string;
  prompt: string;
  archive: string;
  trash: string;
  toggleRead: string;
  manageLabels: string;
  compose: string;
  reply: string;
  forward: string;
  search: string;
  refresh: string;
  loadMore: string;
  drafts: string;
  openGmail: string;
  bulkMode: string;
  bulkSelect: string;
  markdown: string;
  attachments: string;
  help: string;
  gotoTop: string;
  gotoBottom: string;
  linkPicker: string;
  replyAll: string;
  saveMessage: string;
  suggestLabel: string;
  obsidian: string;
  slack: string;
  commandMode: string;
  threading: string;
  savedQueries: string;
  saveQuery: string;
  actionPlan: string;
  themePicker: string;
  generateReply: string;
  move: string;
  toggleHeaders: string;
  searchFrom: string;
  searchTo: string;
  searchSubject: string;
  searchAdvanced: string;
  contentSearch: string;
  undo: string;
  unread: string;
  archived: string;
  saveRaw: string;
  rsvp: string;
  quit: string;
  vimTimeoutMs: number;
  vimRangeTimeoutMs: number;
}

export const DEFAULT_KEYMAP: KeyMap = {
  summarize: "y", prompt: "p", archive: "a", trash: "d", toggleRead: "t",
  manageLabels: "l", compose: "c", reply: "r", forward: "f", search: "s",
  refresh: "R", loadMore: "N", drafts: "D", openGmail: "O", bulkMode: "v",
  bulkSelect: "space", markdown: "M", attachments: "A", help: "?",
  gotoTop: "gg", gotoBottom: "G", linkPicker: "L", replyAll: "E",
  saveMessage: "w", suggestLabel: "o", obsidian: "O", slack: "K",
  commandMode: ":", threading: "T", savedQueries: "Q", saveQuery: "Z",
  actionPlan: "P", themePicker: "H", generateReply: "g", move: "m",
  toggleHeaders: "h", searchFrom: "F", searchTo: "T", searchSubject: "S",
  searchAdvanced: "ctrl+f",
  contentSearch: "/", undo: "U", unread: "u", archived: "B", saveRaw: "W",
  rsvp: "V", quit: "q", vimTimeoutMs: 1000, vimRangeTimeoutMs: 2000,
};

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
  Ready(): Promise<boolean>;
  InitError(): Promise<string>;
  NeedsCredentials(): Promise<boolean>;
  CredentialsPath(): Promise<string>;
  ImportCredentials(): Promise<string>;
  RetryInit(): Promise<void>;
  Quit(): Promise<void>;
  AccountEmail(): Promise<string>;
  ListInbox(pageToken: string, pageSize: number): Promise<MessageList>;
  Search(query: string, pageToken: string, pageSize: number): Promise<MessageList>;
  GetMessage(id: string): Promise<MessageDetail>;
  Archive(id: string): Promise<void>;
  Trash(id: string): Promise<void>;
  MarkRead(id: string): Promise<void>;
  MarkUnread(id: string): Promise<void>;
  Unarchive(id: string): Promise<void>;
  Untrash(id: string): Promise<void>;
  BulkUnarchive(ids: string[]): Promise<void>;
  BulkUntrash(ids: string[]): Promise<void>;
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
  SummarizeStream(id: string, force: boolean): Promise<string>;
  GenerateReply(id: string): Promise<string>;
  TouchUp(id: string): Promise<string>;
  MoveToLabel(messageID: string, name: string): Promise<void>;
  BulkMoveToLabel(ids: string[], name: string): Promise<void>;
  BulkArchive(ids: string[]): Promise<void>;
  BulkTrash(ids: string[]): Promise<void>;
  BulkMarkRead(ids: string[]): Promise<void>;
  BulkMarkUnread(ids: string[]): Promise<void>;
  BulkApplyLabel(ids: string[], labelID: string): Promise<void>;
  BulkRemoveLabel(ids: string[], labelID: string): Promise<void>;
  PromptsEnabled(): Promise<boolean>;
  ListPrompts(): Promise<Prompt[]>;
  GetPrompt(id: number): Promise<PromptDetail>;
  CreatePrompt(name: string, description: string, text: string, category: string): Promise<number>;
  UpdatePrompt(id: number, name: string, description: string, text: string, category: string): Promise<void>;
  DeletePrompt(id: number): Promise<void>;
  RefinePromptText(text: string): Promise<string>;
  ApplyPromptStream(messageID: string, promptID: number): Promise<string>;
  ApplyBulkPromptStream(ids: string[], promptID: number): Promise<string>;
  ListAccounts(): Promise<AccountInfo[]>;
  SwitchAccount(id: string): Promise<void>;
  KeyMap(): Promise<KeyMap>;
  ThreadingEnabled(): Promise<boolean>;
  GetThread(threadID: string): Promise<MessageDetail[]>;
  ThreadSummaryStream(threadID: string): Promise<string>;
  SavedQueriesEnabled(): Promise<boolean>;
  ListSavedQueries(): Promise<SavedQuery[]>;
  SaveQuery(name: string, query: string): Promise<void>;
  DeleteSavedQuery(id: number): Promise<void>;
  RecordQueryUse(id: number): Promise<void>;
  ActionPlanEnabled(): Promise<boolean>;
  AnalyzeInbox(inputs: AnalyzerInput[]): Promise<ActionPlanResult>;
  RunDeterministicRules(inputs: AnalyzerInput[]): Promise<ActionPlanResult>;
  DeterministicRulesRunnable(): Promise<boolean>;
  BulkApplyLabelByName(ids: string[], name: string): Promise<void>;
  AnalyzerRulesEnabled(): Promise<boolean>;
  ListAnalyzerRules(): Promise<AnalyzerRule[]>;
  SaveAnalyzerRule(text: string): Promise<void>;
  DeleteAnalyzerRule(id: number): Promise<void>;
  DeterministicRulesEnabled(): Promise<boolean>;
  ListDeterministicRules(): Promise<DeterministicRule[]>;
  SaveDeterministicRule(
    query: string,
    action: string,
    label: string,
    promptId: number,
  ): Promise<void>;
  UpdateDeterministicRule(
    id: number,
    query: string,
    action: string,
    label: string,
    promptId: number,
  ): Promise<void>;
  DeleteDeterministicRule(id: number): Promise<void>;
  SyncDeterministicRule(id: number): Promise<void>;
  UnsyncDeterministicRule(id: number): Promise<void>;
  ImportGmailFilters(): Promise<ImportResult>;
  ViewAnalyzerPrompt(): Promise<string>;
  ListLinks(messageID: string): Promise<Link[]>;
  OpenURL(url: string): Promise<void>;
  FetchImage(url: string): Promise<string>;
  FetchInlineImage(messageID: string, attachmentID: string): Promise<string>;
  LogUI(msg: string): Promise<void>;
  PendingAuthURL(): Promise<string>;
  OpenAuthURL(): Promise<void>;
  Version(): Promise<string>;
  SaveMessage(messageID: string): Promise<string>;
  SaveRawMessage(messageID: string): Promise<string>;
  AutoRefreshSettings(): Promise<AutoRefreshSettings>;
  RSVPEnabled(): Promise<boolean>;
  InviteInfo(messageID: string): Promise<Invite>;
  RespondInvite(messageID: string, status: string): Promise<void>;
  UsageStats(): Promise<UsageStats>;
  ClearCaches(): Promise<void>;
  ConfigInfo(): Promise<ConfigInfo>;
  ObsidianEnabled(): Promise<boolean>;
  SendToObsidian(messageID: string): Promise<string>;
  SlackEnabled(): Promise<boolean>;
  ForwardToSlack(messageID: string): Promise<void>;
  SuggestLabels(messageID: string): Promise<string[]>;
  ApplyLabelByName(messageID: string, name: string): Promise<void>;
  OpenGmailWeb(messageID: string): Promise<void>;
  ListDrafts(): Promise<DraftSummary[]>;
  GetDraft(draftID: string): Promise<DraftDetail>;
  SaveDraft(to: string, subject: string, body: string, cc: string[]): Promise<string>;
  UpdateDraft(draftID: string, to: string, subject: string, body: string, cc: string[]): Promise<void>;
  DeleteDraft(draftID: string): Promise<void>;
  ThemesEnabled(): Promise<boolean>;
  ListThemes(): Promise<string[]>;
  GetThemeColors(name: string): Promise<ThemeColors | null>;
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

// In browser dev (no real Wails runtime), install a tiny event bus so mock
// backend methods can emit the same progress events the UI listens for. This is
// inert in the packaged app (a real window.runtime already exists) and never
// activates the streaming path (that also requires realBackend()).
let mockEmit: ((name: string, data: unknown) => void) | null = null;
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
  mockEmit = (name, data) =>
    (listeners[name] || []).forEach((f) => f(data));
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
): Promise<string> {
  return streamViaEvent(
    "prompt:token",
    () => backend.ApplyPromptStream(id, promptId),
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
  async Ready() {
    return true;
  },
  async InitError() {
    return "";
  },
  async NeedsCredentials() {
    return false;
  },
  async CredentialsPath() {
    return "~/.config/giztui/credentials.json";
  },
  async ImportCredentials() {
    return ""; // mock: no native file dialog in the browser
  },
  async RetryInit() {
    /* mock: nothing to retry */
  },
  async Quit() {
    /* mock: no-op in the browser */
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
    const plain = `${m.snippet}\n\nHi there,\n\nThis is the full body of "${m.subject}" rendered in the reading pane. In the packaged Wails app this content comes straight from Gmail via the GizTUI service layer.\n\nSee https://example.com/expenses for details (or www.example.org).\n\nBest,\n${m.from}`;
    // Give some messages an HTML body so the HTML renderer is demonstrable.
    const html =
      Number(id.replace(/\D/g, "") || "0") % 2 === 0
        ? `<div style="font-family:Arial,sans-serif">
             <img src="https://example.com/logo.png" alt="logo" width="120">
             <img src="cid:inlineimg1" alt="inline attachment">
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
  async Unarchive() {},
  async Untrash() {},
  async BulkUnarchive() {},
  async BulkUntrash() {},
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
          contentId: "",
        },
        {
          attachmentId: "att-inline",
          filename: "image.png",
          mimeType: "image/png",
          size: 15210,
          type: "image",
          inline: true,
          contentId: "inlineimg1",
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
  async SummarizeStream(id: string, force: boolean) {
    // In the browser mock the streaming helper drives token delivery; this is
    // only the fallback that returns the full text.
    const s = await this.Summarize(id);
    return force ? s + "\n\n(regenerated)" : s;
  },
  async GenerateReply(id: string) {
    const m = mockMessages.find((x) => x.id === id) ?? mockMessages[0];
    await new Promise((r) => setTimeout(r, 400));
    return `Hi,\n\nThanks for your message about "${m.subject}". (mock AI draft) In the packaged app this reply is drafted by your configured LLM — edit it before sending.\n\nBest,`;
  },
  async TouchUp(id: string) {
    const m = mockMessages.find((x) => x.id === id) ?? mockMessages[0];
    await new Promise((r) => setTimeout(r, 400));
    return `## ${m.subject}\n\nHi there,\n\nThis is the message body, **reformatted** by the AI for readability — tidy paragraphs, fixed wrapping and clean markdown.\n\n- Point one\n- Point two\n\nBest,\n${m.from}`;
  },
  async MoveToLabel() {
    await new Promise((r) => setTimeout(r, 200));
  },
  async BulkMoveToLabel() {
    await new Promise((r) => setTimeout(r, 250));
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
    return mockPrompts.map((p) => ({
      id: p.id,
      name: p.name,
      description: p.description,
      category: p.category,
    }));
  },
  async GetPrompt(id: number) {
    return (
      mockPrompts.find((p) => p.id === id) ?? mockPrompts[0]
    );
  },
  async CreatePrompt(name: string, description: string, text: string, category: string) {
    const id = Math.max(0, ...mockPrompts.map((p) => p.id)) + 1;
    mockPrompts = [...mockPrompts, { id, name, description, text, category }];
    return id;
  },
  async UpdatePrompt(id: number, name: string, description: string, text: string, category: string) {
    mockPrompts = mockPrompts.map((p) =>
      p.id === id ? { id, name, description, text, category } : p,
    );
  },
  async DeletePrompt(id: number) {
    mockPrompts = mockPrompts.filter((p) => p.id !== id);
  },
  async RefinePromptText(text: string) {
    await new Promise((r) => setTimeout(r, 400));
    return `${text.trim()}\n\nBe concise and use {{body}} for the email content. (refined by mock AI)`;
  },
  async ApplyBulkPromptStream(ids: string[], _promptID: number) {
    await new Promise((r) => setTimeout(r, 400));
    return `Applied to ${ids.length} messages:\n\n• Combined key points across the selection\n• (mock bulk prompt result)`;
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
  async KeyMap() {
    return DEFAULT_KEYMAP;
  },
  async ThreadingEnabled() {
    return true;
  },
  async GetThread(_id: string) {
    const mk = (i: number, from: string, unread: boolean): MessageDetail => ({
      id: `t${i}`,
      threadId: "thread-1",
      subject: "Re: Project roadmap Q3",
      from,
      to: "you@example.com",
      cc: "",
      date: new Date(Date.now() - (3 - i) * 3600_000).toISOString(),
      unread,
      labels: ["Work"],
      plainText:
        i === 0
          ? "Hi team,\n\nHere is the first pass at the Q3 roadmap. Thoughts?"
          : i === 1
            ? "Looks great — I'd move the analytics milestone earlier though."
            : "Agreed. Let's lock it in for the Friday review.",
      html: "",
    });
    return [
      mk(0, "Ada Lovelace <ada@compute.org>", false),
      mk(1, "you <you@example.com>", false),
      mk(2, "Grace Hopper <grace@navy.mil>", true),
    ];
  },
  async ThreadSummaryStream() {
    return "• Ada shared the Q3 roadmap draft\n• Team agreed to move the analytics milestone earlier\n• Locked in for the Friday review";
  },
  async SavedQueriesEnabled() {
    return true;
  },
  async ListSavedQueries() {
    return mockQueries;
  },
  async SaveQuery(name: string, query: string) {
    mockQueries = [
      { id: mockQueries.length + 1, name, query, description: "", category: "" },
      ...mockQueries,
    ];
  },
  async DeleteSavedQuery(id: number) {
    mockQueries = mockQueries.filter((q) => q.id !== id);
  },
  async RecordQueryUse() {},
  async ActionPlanEnabled() {
    return true;
  },
  async AnalyzeInbox(inputs: AnalyzerInput[]) {
    // Emit fake per-batch progress so the determinate bar is exercised in dev:
    // an initial 0/total (blocks known), then each block completing.
    const total = 4;
    mockEmit?.("plan:progress", { done: 0, total });
    for (let i = 1; i <= total; i++) {
      await new Promise((r) => setTimeout(r, 350));
      mockEmit?.("plan:progress", { done: i, total });
    }
    const ids = inputs.map((i) => i.id);
    return {
      totalAnalyzed: inputs.length,
      readManually: 2,
      categories: [
        { name: "Archive: from:github.com", priority: "medium", description: "Matched by rule: from:github.com", action: "archive", label: "", messageIds: ids.slice(0, 1), byRule: true },
        { name: "Newsletters", priority: "low", description: "Digests and weekly roundups", action: "archive", label: "", messageIds: ids.slice(1, 3) },
        { name: "Calendar invites", priority: "medium", description: "Accepted meeting notifications", action: "mark_read", label: "", messageIds: ids.slice(3, 5) },
        { name: "Finance", priority: "high", description: "Invoices and expenses to review", action: "label", label: "Finance", messageIds: ids.slice(5, 6) },
        { name: "Prompt: summarize", priority: "medium", description: "Matched by rule: label:receipts", action: "prompt", label: "", messageIds: ids.slice(6, 8), byRule: true, promptId: 1 },
        { name: "Read manually", priority: "low", description: "The AI left these for you to review", action: "none", label: "", messageIds: ids.slice(8, 10), readManually: true },
      ],
    };
  },
  async RunDeterministicRules(inputs: AnalyzerInput[]) {
    await new Promise((r) => setTimeout(r, 250));
    const ids = inputs.map((i) => i.id);
    return {
      totalAnalyzed: Math.min(3, ids.length),
      readManually: 0,
      categories: [
        { name: "Archive: from:github.com", priority: "medium", description: "Matched by rule: from:github.com", action: "archive", label: "", messageIds: ids.slice(0, 1), byRule: true },
        { name: "Label Finance: from:billing", priority: "medium", description: "Matched by rule: from:billing", action: "label", label: "Finance", messageIds: ids.slice(1, 3), byRule: true },
      ],
    };
  },
  async DeterministicRulesRunnable() {
    return true;
  },
  async BulkApplyLabelByName() {
    await new Promise((r) => setTimeout(r, 200));
  },
  async AnalyzerRulesEnabled() {
    return true;
  },
  async ListAnalyzerRules() {
    return mockRules;
  },
  async SaveAnalyzerRule(text: string) {
    const id = Math.max(0, ...mockRules.map((r) => r.id)) + 1;
    mockRules = [...mockRules, { id, text }];
  },
  async DeleteAnalyzerRule(id: number) {
    mockRules = mockRules.filter((r) => r.id !== id);
  },
  async DeterministicRulesEnabled() {
    return true;
  },
  async ListDeterministicRules() {
    return mockDetRules;
  },
  async SaveDeterministicRule(
    query: string,
    action: string,
    label: string,
    promptId: number,
  ) {
    const id = Math.max(0, ...mockDetRules.map((r) => r.id)) + 1;
    mockDetRules = [
      ...mockDetRules,
      { id, query, action, label, promptId, synced: false, createdAt: 0 },
    ];
  },
  async UpdateDeterministicRule(
    id: number,
    query: string,
    action: string,
    label: string,
    promptId: number,
  ) {
    mockDetRules = mockDetRules.map((r) =>
      r.id === id ? { ...r, query, action, label, promptId } : r,
    );
  },
  async DeleteDeterministicRule(id: number) {
    mockDetRules = mockDetRules.filter((r) => r.id !== id);
  },
  async SyncDeterministicRule(id: number) {
    mockDetRules = mockDetRules.map((r) =>
      r.id === id ? { ...r, synced: true } : r,
    );
  },
  async UnsyncDeterministicRule(id: number) {
    mockDetRules = mockDetRules.map((r) =>
      r.id === id ? { ...r, synced: false } : r,
    );
  },
  async ImportGmailFilters() {
    return { imported: 2, adopted: 1, removed: 0, unsupported: [] };
  },
  async ViewAnalyzerPrompt() {
    const rulesBlock = mockRules.length
      ? "User preferences:\n" + mockRules.map((r) => `- ${r.text}`).join("\n") + "\n\n"
      : "";
    return `${rulesBlock}You are an inbox assistant. Group the following emails into actionable categories (archive, mark_read, trash, label) and return JSON.\n\n{{messages}}`;
  },
  async ListLinks(_id: string) {
    return [
      { index: 1, url: "https://example.com/expenses", text: "página de gastos", type: "html" },
      { index: 2, url: "https://example.com/open", text: "Open in Expensify", type: "html" },
    ];
  },
  async OpenURL() {
    /* mock: no-op */
  },
  async FetchImage(url: string) {
    // Mock: echo the URL back so the browser loads it directly (real backend
    // returns a data: URI).
    return url;
  },
  async FetchInlineImage(_m: string, attachmentID: string) {
    // Mock: return a tiny inline SVG data URI so cid: images render in the
    // browser (real backend returns the attachment bytes as a data URI).
    const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="120" height="80"><rect width="120" height="80" fill="#1a56db"/><text x="60" y="45" fill="#fff" font-size="12" text-anchor="middle">${attachmentID}</text></svg>`;
    return `data:image/svg+xml;base64,${btoa(svg)}`;
  },
  async LogUI(msg: string) {
    // Mock: mirror to the browser console for dev visibility.
    console.log("[ui]", msg);
  },
  async PendingAuthURL() {
    // Mock: never prompts for sign-in in the browser.
    return "";
  },
  async OpenAuthURL() {},
  async Version() {
    return "dev (mock)";
  },
  async SaveMessage() {
    await new Promise((r) => setTimeout(r, 200));
    return "~/Downloads/gmail-attachments/message.txt";
  },
  async SaveRawMessage() {
    await new Promise((r) => setTimeout(r, 200));
    return "~/Downloads/gmail-attachments/message.eml";
  },
  async AutoRefreshSettings() {
    return { enabled: false, intervalSeconds: 60 };
  },
  async RSVPEnabled() {
    return true;
  },
  async InviteInfo(messageID: string) {
    // Mock: treat every 5th message as a calendar invite so the RSVP bar shows.
    const isInvite = Number(messageID.replace(/\D/g, "") || "0") % 5 === 0;
    return {
      isInvite,
      uid: isInvite ? "mock-uid-123" : "",
      summary: "Weekly sync: AllFunds · AWS · DoiT",
      organizer: "mailto:ada@compute.org",
      dtStart: "20260720T150000",
      dtEnd: "20260720T153000",
    };
  },
  async RespondInvite() {
    await new Promise((r) => setTimeout(r, 300));
  },
  async UsageStats() {
    return {
      totalUsage: 42,
      uniquePrompts: 4,
      topPrompts: [
        { name: "Summarize concisely", category: "general", usageCount: 21 },
        { name: "Extract action items", category: "productivity", usageCount: 12 },
        { name: "Translate to Spanish", category: "language", usageCount: 6 },
        { name: "Draft a polite reply", category: "compose", usageCount: 3 },
      ],
    };
  },
  async ClearCaches() {
    await new Promise((r) => setTimeout(r, 250));
  },
  async ConfigInfo() {
    return {
      configPath: "~/.config/giztui/config.json",
      logPath: "~/.config/giztui/desktop.log",
      account: "you@example.com (mock)",
      llmProvider: "ollama",
      llmModel: "llama3.1",
      theme: "slate-blue",
      obsidianOn: true,
      slackOn: true,
      autoRefresh: false,
      downloadPath: "~/Downloads/gmail-attachments",
    };
  },
  async ObsidianEnabled() {
    return true;
  },
  async SendToObsidian() {
    await new Promise((r) => setTimeout(r, 300));
    return "00-Inbox/2026-07-18_welcome.md";
  },
  async SlackEnabled() {
    return true;
  },
  async ForwardToSlack() {
    await new Promise((r) => setTimeout(r, 300));
  },
  async SuggestLabels() {
    await new Promise((r) => setTimeout(r, 400));
    return ["Work", "Finance", "Follow-up"];
  },
  async ApplyLabelByName() {
    await new Promise((r) => setTimeout(r, 150));
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
  async ThemesEnabled() {
    return true;
  },
  async ListThemes() {
    return ["gmail-dark", "slate-blue", "dracula", "solarized-dark", "gruvbox"];
  },
  async GetThemeColors(name: string) {
    const themes: Record<string, ThemeColors> = {
      "slate-blue": {
        name: "slate-blue", bg: "#0f172a", fg: "#e2e8f0", border: "#334155",
        accent: "#38bdf8", primary: "#7dd3fc", danger: "#f87171", warning: "#fbbf24",
        success: "#4ade80", selectionBg: "#1e293b", inputBg: "#1e293b",
        unread: "#7dd3fc", muted: "#64748b",
      },
      dracula: {
        name: "dracula", bg: "#282a36", fg: "#f8f8f2", border: "#44475a",
        accent: "#bd93f9", primary: "#ff79c6", danger: "#ff5555", warning: "#f1fa8c",
        success: "#50fa7b", selectionBg: "#44475a", inputBg: "#21222c",
        unread: "#8be9fd", muted: "#6272a4",
      },
    };
    return themes[name] ?? themes["slate-blue"];
  },
};

let mockActiveAccount = "personal";
let mockRules: AnalyzerRule[] = [
  { id: 1, text: "Always archive newsletters and weekly digests" },
  { id: 2, text: "Never trash anything from my bank" },
];
let mockDetRules: DeterministicRule[] = [
  { id: 1, query: "from:github.com", action: "archive", label: "", promptId: 0, synced: true, createdAt: 0 },
  { id: 2, query: "from:(support@zendesk.com)", action: "trash", label: "", promptId: 0, synced: false, createdAt: 0 },
  { id: 3, query: "from:(confluence@atlassian.net)", action: "label", label: "Docs", promptId: 0, synced: true, createdAt: 0 },
  { id: 4, query: "from:(substack.com OR medium.com)", action: "label", label: "Newsletter", promptId: 0, synced: false, createdAt: 0 },
];
let mockPrompts: PromptDetail[] = [
  { id: 1, name: "Summarize concisely", description: "3-bullet summary", category: "general", text: "Summarize the following email in 3 bullets:\n\n{{body}}" },
  { id: 2, name: "Extract action items", description: "List to-dos & owners", category: "productivity", text: "List the action items and owners in this email:\n\n{{body}}" },
  { id: 3, name: "Draft a polite reply", description: "Suggest a response", category: "compose", text: "Draft a polite reply to:\n\n{{body}}" },
  { id: 4, name: "Translate to Spanish", description: "Translate the email", category: "language", text: "Translate this email to Spanish:\n\n{{body}}" },
];
let mockQueries: SavedQuery[] = [
  { id: 1, name: "Unread from team", query: "is:unread from:team", description: "", category: "" },
  { id: 2, name: "Has attachments", query: "has:attachment newer_than:7d", description: "", category: "" },
];
let mockDrafts: DraftSummary[] = [
  { id: "d1", to: "ada@compute.org", subject: "Re: Project roadmap Q3", snippet: "Thanks Ada, I think we should…" },
  { id: "d2", to: "team@giztui.dev", subject: "Release notes draft", snippet: "Here's a first pass at the notes…" },
];

const mockAppliedLabels: Record<string, string[]> = {};

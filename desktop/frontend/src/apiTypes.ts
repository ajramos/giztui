// api types — DTOs + the Backend method contract shared by the real Wails
// binding (api.ts) and the browser dev mock (apiMock.ts). Split out of api.ts
// to keep every file under 500 lines.

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
  aiJobs: string;
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
  rsvp: "V", aiJobs: "J", quit: "q", vimTimeoutMs: 1000, vimRangeTimeoutMs: 2000,
};

export interface CachedPromptResult {
  promptId: number;
  name: string;
  text: string;
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

export interface Backend {
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
  JobsNotifyOnComplete(): Promise<boolean>;
  ChatEnabled(): Promise<boolean>;
  ChatStream(id: string, message: string): Promise<string>;
  ChatReset(id: string): Promise<void>;
  ListPrompts(): Promise<Prompt[]>;
  GetPrompt(id: number): Promise<PromptDetail>;
  CreatePrompt(name: string, description: string, text: string, category: string): Promise<number>;
  UpdatePrompt(id: number, name: string, description: string, text: string, category: string): Promise<void>;
  DeletePrompt(id: number): Promise<void>;
  RefinePromptText(text: string): Promise<string>;
  ApplyPromptStream(
    messageID: string,
    promptID: number,
    force: boolean,
  ): Promise<string>;
  CachedPrompts(messageID: string): Promise<CachedPromptResult[]>;
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
  BulkApplyLabelByName(ids: string[], name: string): Promise<void>;
  AnalyzerRulesEnabled(): Promise<boolean>;
  ListAnalyzerRules(): Promise<AnalyzerRule[]>;
  SaveAnalyzerRule(text: string): Promise<void>;
  DeleteAnalyzerRule(id: number): Promise<void>;
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

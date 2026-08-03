import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import type {
  MessageSummary,
  MessageDetail,
  Attachment,
  Invite,
  AccountInfo,
} from "./apiTypes";
import type { ComposeInit } from "./Compose";

// Everything the command runner (executeCommand) needs from App. Passing a
// single ctx bag lets the ~490-line command switch live in its own module
// (commandRunner.ts) instead of inside App.tsx. App builds this each render.
export interface CommandCtx {
  detail: MessageDetail | null;
  load: (q: string) => Promise<void>;
  doAction: (action: "archive" | "trash" | "read" | "unread" | "star" | "unstar", id: string) => Promise<void>;
  bulkAction: (action: "archive" | "trash" | "read" | "unread" | "star" | "unstar") => Promise<void>;
  activeQuery: string;
  openDrafts: () => void;
  saveMessage: (id: string) => void;
  openObsidian: () => void;
  openSlackForward: () => void;
  obsidianOn: boolean;
  slackOn: boolean;
  aiEnabled: boolean;
  aiPromptsEnabled: boolean;
  summarize: (id: string, force?: boolean) => Promise<void>;
  openSuggest: (id: string) => Promise<void>;
  openInGmail: (id: string) => void;
  openQueries: () => Promise<void>;
  savedQueriesOn: boolean;
  runActionPlan: () => void;
  runDeterministicRules: () => Promise<void>;
  actionPlanOn: boolean;
  bulkMode: boolean;
  selected: Set<string>;
  showToast: (m: string) => void;
  doMove: (id: string, name: string) => Promise<void>;
  doBulkMove: (name: string) => Promise<void>;
  generateReply: (d: MessageDetail) => Promise<void>;
  quickSearch: (kind: "from" | "to" | "subject", d: MessageDetail) => void;
  themesOn: boolean;
  applyTheme: (name: string) => Promise<void>;
  rulesEnabled: boolean;
  openRules: () => Promise<void>;
  viewAnalyzerPrompt: () => Promise<void>;
  toggleToolbar: () => void;
  touchUp: (id: string) => Promise<void>;
  touchUpText: string | null;
  localFilter: boolean;
  applyLocalFilter: (q: string) => void;
  query: string;
  runUndo: () => Promise<void>;
  toggleAutoRefresh: () => void;
  toggleNumbers: () => void;
  saveRawMessage: (id: string) => void;
  invite: Invite | null;
  respondInvite: (id: string, status: "accepted" | "tentative" | "declined") => Promise<void>;
  openStats: () => Promise<void>;
  openTelemetry: (days?: number) => Promise<void>;
  resetTelemetry: () => Promise<void>;
  openConfig: () => Promise<void>;
  clearCaches: () => Promise<void>;
  loadMore: () => Promise<void>;
  attachments: Attachment[];
  threadingOn: boolean;
  toggleThread: () => Promise<void>;
  threadMsgs: MessageDetail[] | null;
  summarizeThread: () => Promise<void>;
  messages: MessageSummary[];
  previewMessage: (m: MessageSummary) => void;
  accounts: AccountInfo[];
  applyLabelChange: (ids: Set<string>, change: { added?: string; removed?: string }) => void;
  bumpZoom: (delta: number) => void;
  resetZoom: () => void;
  setZoom: (z: number) => void;
  dismissAI: () => boolean;
  regenerateActive: () => void;
  setError: (e: string) => void;
  setCompose: Dispatch<SetStateAction<ComposeInit | null>>;
  setSelected: Dispatch<SetStateAction<Set<string>>>;
  setSelectedId: Dispatch<SetStateAction<string | null>>;
  setMessages: Dispatch<SetStateAction<MessageSummary[]>>;
  setLabelsFor: Dispatch<SetStateAction<string | null>>;
  setLinksFor: Dispatch<SetStateAction<string | null>>;
  setMoveFor: Dispatch<SetStateAction<string | null>>;
  setTouchUpText: Dispatch<SetStateAction<string | null>>;
  setCsQuery: Dispatch<SetStateAction<string>>;
  setCsIndex: Dispatch<SetStateAction<number>>;
  setCsOpen: Dispatch<SetStateAction<boolean>>;
  setCollapsedMsgs: Dispatch<SetStateAction<Set<string>>>;
  setLocalFilter: Dispatch<SetStateAction<boolean>>;
  setBulkMode: Dispatch<SetStateAction<boolean>>;
  setViewHtml: Dispatch<SetStateAction<boolean>>;
  setHeadersHidden: Dispatch<SetStateAction<boolean>>;
  setLoadRemote: Dispatch<SetStateAction<boolean>>;
  setAccountsOpen: Dispatch<SetStateAction<boolean>>;
  setAdvOpen: Dispatch<SetStateAction<boolean>>;
  setAttachmentsOpen: Dispatch<SetStateAction<boolean>>;
  setBulkMove: Dispatch<SetStateAction<boolean>>;
  setDetRulesOpen: Dispatch<SetStateAction<boolean>>;
  setPromptsOpen: Dispatch<SetStateAction<boolean>>;
  setRsvpPickerOpen: Dispatch<SetStateAction<boolean>>;
  setJobsPickerOpen: Dispatch<SetStateAction<boolean>>;
  openChat: () => void;
  setSaveQueryOpen: Dispatch<SetStateAction<boolean>>;
  setShowHelp: Dispatch<SetStateAction<boolean>>;
  setThemePickerOpen: (v: boolean) => void;
  setAlwaysImagesOn: (v: boolean) => void;
  alwaysImagesRef: MutableRefObject<boolean>;
  imageOptIn: MutableRefObject<Set<string>>;
  fullMessagesRef: MutableRefObject<MessageSummary[]>;
}

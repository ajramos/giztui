// Mutable mock state shared by the mock method halves (apiMockA/apiMockB). A
// single container object so both files can read AND reassign it across module
// boundaries (ES modules forbid reassigning an imported binding, but object
// property mutation is fine). Browser-dev only.
import type {
  MessageSummary,
  AnalyzerRule,
  DeterministicRule,
  PromptDetail,
  SavedQuery,
  DraftSummary,
} from "./apiTypes";

export const mockMessages: MessageSummary[] = Array.from({ length: 24 }, (_, i) => ({
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
  starred: i % 4 === 1,
  labels: i % 4 === 0 ? ["Work"] : i % 5 === 0 ? ["Personal", "Travel"] : [],
}));

export const md = {
  activeAccount: "personal",
  appliedLabels: {} as Record<string, string[]>,
  rules: [
  { id: 1, text: "Always archive newsletters and weekly digests" },
  { id: 2, text: "Never trash anything from my bank" },
] as AnalyzerRule[],
  detRules: [
  { id: 1, query: "from:github.com", action: "archive", label: "", promptId: 0, synced: true, createdAt: 0 },
  { id: 2, query: "from:(support@zendesk.com)", action: "trash", label: "", promptId: 0, synced: false, createdAt: 0 },
  { id: 3, query: "from:(confluence@atlassian.net)", action: "label", label: "Docs", promptId: 0, synced: true, createdAt: 0 },
  { id: 4, query: "from:(substack.com OR medium.com)", action: "label", label: "Newsletter", promptId: 0, synced: false, createdAt: 0 },
] as DeterministicRule[],
  prompts: [
  { id: 1, name: "Summarize concisely", description: "3-bullet summary", category: "general", text: "Summarize the following email in 3 bullets:\n\n{{body}}" },
  { id: 2, name: "Extract action items", description: "List to-dos & owners", category: "productivity", text: "List the action items and owners in this email:\n\n{{body}}" },
  { id: 3, name: "Draft a polite reply", description: "Suggest a response", category: "compose", text: "Draft a polite reply to:\n\n{{body}}" },
  { id: 4, name: "Translate to Spanish", description: "Translate the email", category: "language", text: "Translate this email to Spanish:\n\n{{body}}" },
] as PromptDetail[],
  queries: [
  { id: 1, name: "Unread from team", query: "is:unread from:team", description: "", category: "" },
  { id: 2, name: "Has attachments", query: "has:attachment newer_than:7d", description: "", category: "" },
] as SavedQuery[],
  drafts: [
  { id: "d1", to: "ada@compute.org", subject: "Re: Project roadmap Q3", snippet: "Thanks Ada, I think we should…" },
  { id: "d2", to: "team@giztui.dev", subject: "Release notes draft", snippet: "Here's a first pass at the notes…" },
] as DraftSummary[],
};

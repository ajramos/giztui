// Mock backend — part B (accounts, threading, saved queries, analyzer/
// deterministic rules, links, drafts, config, themes). Merged in apiMock.ts.
import { mockEmit } from "./apiEvents";
import type { AnalyzerInput, ThemeColors, Backend } from "./apiTypes";
import { md } from "./apiMockData";

export const mockB: Partial<Backend> = {
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
  async BulkApplyLabelByName() {
    await new Promise((r) => setTimeout(r, 200));
  },
  async AnalyzerRulesEnabled() {
    return true;
  },
  async ListAnalyzerRules() {
    return md.rules;
  },
  async SaveAnalyzerRule(text: string) {
    const id = Math.max(0, ...md.rules.map((r) => r.id)) + 1;
    md.rules = [...md.rules, { id, text }];
  },
  async DeleteAnalyzerRule(id: number) {
    md.rules = md.rules.filter((r) => r.id !== id);
  },
  async ListDeterministicRules() {
    return md.detRules;
  },
  async SaveDeterministicRule(
    query: string,
    action: string,
    label: string,
    promptId: number,
  ) {
    const id = Math.max(0, ...md.detRules.map((r) => r.id)) + 1;
    md.detRules = [
      ...md.detRules,
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
    md.detRules = md.detRules.map((r) =>
      r.id === id ? { ...r, query, action, label, promptId } : r,
    );
  },
  async DeleteDeterministicRule(id: number) {
    md.detRules = md.detRules.filter((r) => r.id !== id);
  },
  async SyncDeterministicRule(id: number) {
    md.detRules = md.detRules.map((r) =>
      r.id === id ? { ...r, synced: true } : r,
    );
  },
  async UnsyncDeterministicRule(id: number) {
    md.detRules = md.detRules.map((r) =>
      r.id === id ? { ...r, synced: false } : r,
    );
  },
  async ImportGmailFilters() {
    return { imported: 2, adopted: 1, removed: 0, unsupported: [] };
  },
  async ViewAnalyzerPrompt() {
    const rulesBlock = md.rules.length
      ? "User preferences:\n" + md.rules.map((r) => `- ${r.text}`).join("\n") + "\n\n"
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
    return md.drafts;
  },
  async GetDraft(draftID: string) {
    const d = md.drafts.find((x) => x.id === draftID) ?? md.drafts[0];
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
    const id = "draft-" + (md.drafts.length + 1);
    md.drafts = [{ id, to, subject, snippet: "New draft" }, ...md.drafts];
    return id;
  },
  async UpdateDraft() {
    await new Promise((r) => setTimeout(r, 200));
  },
  async DeleteDraft(draftID: string) {
    await new Promise((r) => setTimeout(r, 200));
    md.drafts = md.drafts.filter((x) => x.id !== draftID);
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

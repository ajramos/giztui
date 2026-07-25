import type { KeyMap } from "./api";

// Feature availability, so sections/rows reflect what's actually configured.
export interface HelpFlags {
  ai: boolean;
  prompts: boolean;
  obsidian: boolean;
  slack: boolean;
  threading: boolean;
  savedQueries: boolean;
  actionPlan: boolean;
  rsvp: boolean;
  themes: boolean;
  rules: boolean;
}

// fmtKey turns a config key token into a readable label ("space" → "Space",
// "gg" → "g g", "ctrl+k" → "Ctrl+K", "enter" → "Enter").
function fmtKey(k: string): string {
  if (!k) return "—";
  if (k === "space" || k === " ") return "Space";
  if (k === "gg") return "g g";
  return k
    .split("+")
    .map((p) => (p.length === 1 ? p : p.charAt(0).toUpperCase() + p.slice(1)))
    .join("+");
}

type Row = { icon: string; keys: string; desc: string };
type Section = { title: string; icon: string; rows: Row[] };

function buildSections(k: KeyMap, f: HelpFlags): Section[] {
  const secs: Section[] = [];

  secs.push({
    icon: "📧",
    title: "Triage",
    rows: [
      { icon: "✏️", keys: fmtKey(k.compose), desc: "Compose new message" },
      { icon: "📁", keys: fmtKey(k.archive), desc: "Archive" },
      { icon: "🗑️", keys: fmtKey(k.trash), desc: "Move to trash" },
      { icon: "👁️", keys: fmtKey(k.toggleRead), desc: "Toggle read / unread" },
      { icon: "↩️", keys: fmtKey(k.undo), desc: "Undo last action" },
      { icon: "📦", keys: fmtKey(k.move), desc: "Move to folder" },
      { icon: "🔖", keys: fmtKey(k.manageLabels), desc: "Manage labels" },
      { icon: "📝", keys: fmtKey(k.drafts), desc: "Drafts" },
    ],
  });

  secs.push({
    icon: "🧭",
    title: "Navigate & search",
    rows: [
      { icon: "↕️", keys: "j / k", desc: "Move cursor (preview, no read)" },
      { icon: "↵", keys: "Enter", desc: "Open message (marks read)" },
      { icon: "⬆️", keys: fmtKey(k.gotoTop), desc: "Go to top" },
      { icon: "⬇️", keys: fmtKey(k.gotoBottom), desc: "Go to bottom" },
      { icon: "🔄", keys: fmtKey(k.refresh), desc: "Refresh inbox" },
      { icon: "⏬", keys: fmtKey(k.loadMore), desc: "Load more" },
      { icon: "🔍", keys: fmtKey(k.search), desc: "Search mail (Gmail)" },
      { icon: "📬", keys: fmtKey(k.unread), desc: "Unread messages" },
      { icon: "🗄️", keys: fmtKey(k.archived), desc: "Archived messages" },
      { icon: "📥", keys: ":inbox", desc: "Back to inbox" },
      { icon: "📫", keys: fmtKey(k.searchFrom), desc: "Search from this sender" },
      { icon: "📤", keys: fmtKey(k.searchTo), desc: "Search to this recipient" },
      { icon: "🧵", keys: fmtKey(k.searchSubject), desc: "Search this subject" },
      { icon: "🔎", keys: `${fmtKey(k.searchAdvanced)} / :advanced`, desc: "Advanced search builder" },
      { icon: "🎯", keys: ":local", desc: "Local filter ↔ Gmail search" },
      ...(f.savedQueries
        ? [
            { icon: "★", keys: fmtKey(k.savedQueries), desc: "Saved searches" },
            { icon: "💾", keys: fmtKey(k.saveQuery), desc: "Save current search" },
          ]
        : []),
    ],
  });

  secs.push({
    icon: "📖",
    title: "Reading a message",
    rows: [
      { icon: "🔍", keys: fmtKey(k.contentSearch), desc: "Find in message" },
      { icon: "⤵️", keys: "n / N", desc: "Next / previous match" },
      { icon: "🔀", keys: fmtKey(k.markdown), desc: "Toggle HTML / text" },
      { icon: "📄", keys: fmtKey(k.toggleHeaders), desc: "Hide / show header block" },
      { icon: "🖼️", keys: ":images", desc: "Load remote images (inline load auto)" },
      { icon: "📎", keys: fmtKey(k.attachments), desc: "Attachments picker (↑↓ · 1-9 · Enter save)" },
      { icon: "🔗", keys: fmtKey(k.linkPicker), desc: "Links in message" },
      { icon: "↪️", keys: fmtKey(k.reply), desc: "Reply" },
      { icon: "↩️", keys: fmtKey(k.replyAll), desc: "Reply all" },
      { icon: "➡️", keys: fmtKey(k.forward), desc: "Forward" },
      { icon: "🌐", keys: fmtKey(k.openGmail), desc: "Open in Gmail" },
    ],
  });

  secs.push({
    icon: "📦",
    title: "Bulk / selection",
    rows: [
      { icon: "✅", keys: fmtKey(k.bulkMode), desc: "Toggle bulk mode" },
      { icon: "➕", keys: `${fmtKey(k.bulkSelect)} / Space`, desc: "Select (enters bulk)" },
      { icon: "🌟", keys: "*", desc: "Select all loaded" },
      { icon: "📁", keys: fmtKey(k.archive), desc: "Archive selected" },
      { icon: "🗑️", keys: fmtKey(k.trash), desc: "Trash selected" },
      { icon: "🔖", keys: fmtKey(k.manageLabels), desc: "Label selected" },
      { icon: "📦", keys: fmtKey(k.move), desc: "Move selected to folder" },
      { icon: "❌", keys: "Esc", desc: "Exit bulk mode" },
    ],
  });

  // VIM-style range operations: key, count, same key → act on the next N rows.
  secs.push({
    icon: "⚡",
    title: "Power ops (VIM ranges)",
    rows: [
      { icon: "📁", keys: `${k.archive}3${k.archive}`, desc: "Archive next 3" },
      { icon: "🗑️", keys: `${k.trash}2${k.trash}`, desc: "Trash next 2" },
      { icon: "👁️", keys: `${k.toggleRead}5${k.toggleRead}`, desc: "Toggle read on next 5" },
      { icon: "🔖", keys: `${k.manageLabels}2${k.manageLabels}`, desc: "Label next 2" },
      { icon: "💡", keys: "key · N · key", desc: "One press = single message" },
    ],
  });

  if (f.ai) {
    secs.push({
      icon: "🤖",
      title: "AI",
      rows: [
        { icon: "📝", keys: fmtKey(k.summarize), desc: "Summarize" },
        { icon: "🔄", keys: fmtKey(k.summarize.toUpperCase()), desc: "Regenerate (ignore cache)" },
        ...(f.prompts
          ? [{ icon: "🎯", keys: fmtKey(k.prompt), desc: "Apply a prompt" }]
          : []),
        { icon: "✒️", keys: "⋯ / :draft", desc: "Draft reply (AI)" },
        { icon: "✨", keys: ":touch-up", desc: "Reformat message (AI)" },
        { icon: "🔖", keys: fmtKey(k.suggestLabel), desc: "Suggest labels (AI)" },
        ...(f.prompts
          ? [{ icon: "⚙️", keys: ":prompts · ⌘E", desc: "Manage prompts (⌘E in picker)" }]
          : []),
        ...(f.actionPlan
          ? [
              { icon: "🧠", keys: fmtKey(k.actionPlan), desc: "Inbox action plan" },
              { icon: "☑️", keys: "Space", desc: "Select / deselect an email (in plan)" },
              { icon: "👁️", keys: "Enter", desc: "Peek email · apply bucket (label→move)" },
              { icon: "🏷️", keys: "l", desc: "Apply a label bucket as label-only" },
              { icon: "🔀", keys: "m", desc: "Recategorize email / bucket (in plan)" },
              { icon: "📄", keys: "p / :action-plan prompt", desc: "Preview the analyzer prompt" },
              ...(f.rules
                ? [{ icon: "🎛️", keys: ":action-plan rules", desc: "AI analyzer preference rules" }]
                : []),
              { icon: "📐", keys: ":rules", desc: "Deterministic rules manager" },
              { icon: "⚡", keys: ":rules plan", desc: "Run deterministic rules (no AI)" },
            ]
          : []),
      ],
    });
  }

  if (f.threading) {
    secs.push({
      icon: "🧵",
      title: "Conversation",
      rows: [
        { icon: "🔄", keys: `${fmtKey(k.threading)} / :threads`, desc: "Toggle conversation" },
        { icon: "📂", keys: "click", desc: "Expand / collapse a message" },
        { icon: "📤", keys: ":expand-all", desc: "Expand all messages" },
        { icon: "📥", keys: ":collapse-all", desc: "Collapse all messages" },
        ...(f.ai
          ? [{ icon: "🧵", keys: ":thread-summary", desc: "Summarize thread" }]
          : []),
      ],
    });
  }

  const tools: Row[] = [
    { icon: "💾", keys: fmtKey(k.saveMessage), desc: "Save to file" },
    { icon: "📄", keys: fmtKey(k.saveRaw), desc: "Save raw .eml" },
  ];
  if (f.obsidian) tools.push({ icon: "📝", keys: fmtKey(k.obsidian), desc: "Send to Obsidian" });
  if (f.slack) tools.push({ icon: "💬", keys: fmtKey(k.slack), desc: "Forward to Slack" });
  if (f.rsvp) {
    tools.push({ icon: "📅", keys: `${fmtKey(k.rsvp)} · :rsvp · :accept/:decline`, desc: "RSVP to invites" });
  }
  tools.push({ icon: "👤", keys: "Ctrl+A", desc: "Switch account" });
  if (f.themes) tools.push({ icon: "🎨", keys: fmtKey(k.themePicker), desc: "Theme picker" });
  tools.push({ icon: "🔎", keys: "Cmd/Ctrl +/-/0", desc: "Zoom UI in / out / reset" });
  tools.push({ icon: "🕐", keys: ":autorefresh", desc: "Toggle auto-refresh" });
  tools.push({ icon: "🖼️", keys: ":images-always", desc: "Always load remote images (on/off)" });
  {
    // The regenerate key is the uppercase of the summarize key, but only if that
    // slot isn't already taken by another binding (e.g. load-more). Show the key
    // only when it's actually free, so the help never claims a shadowed shortcut.
    const regenKey = (k.summarize || "y").toUpperCase();
    const taken = [
      k.loadMore,
      k.refresh,
      k.replyAll,
      k.forward,
      k.savedQueries,
      k.saveQuery,
      k.actionPlan,
      k.themePicker,
      k.slack,
      k.obsidian,
      k.openGmail,
      k.drafts,
      k.markdown,
      k.attachments,
      k.linkPicker,
      k.threading,
      k.searchFrom,
      k.searchTo,
      k.searchSubject,
      k.unread,
      k.archived,
      k.saveRaw,
      k.rsvp,
      k.move,
    ].includes(regenKey);
    tools.push({
      icon: "🔁",
      keys: taken ? ":regenerate" : `${regenKey} · :regenerate`,
      desc: "Regenerate the open AI panel (summary/prompt)",
    });
  }
  tools.push({ icon: "✕", keys: "Esc · :dismiss", desc: "Close the open AI panel" });
  tools.push({ icon: "▤", keys: ":toolbar", desc: "Show / hide reader toolbar" });
  tools.push({ icon: "📊", keys: ":stats", desc: "AI usage" });
  tools.push({ icon: "🛠️", keys: ":config", desc: "Configuration" });
  tools.push({ icon: "⌨️", keys: fmtKey(k.commandMode), desc: "Command palette" });
  tools.push({ icon: "❓", keys: fmtKey(k.help), desc: "This help" });
  tools.push({ icon: "🚪", keys: fmtKey(k.quit), desc: "Quit (:quit)" });
  secs.push({ icon: "🔧", title: "Tools & view", rows: tools });

  return secs;
}

export default function Help({
  keymap,
  flags,
  version,
  onClose,
}: {
  keymap: KeyMap;
  flags: HelpFlags;
  version?: string;
  onClose: () => void;
}) {
  const sections = buildSections(keymap, flags);
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal help-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>GizTUI Desktop{version ? ` · ${version}` : ""}</h3>
          <span className="help-hint muted">
            your configured keys · press <kbd>Esc</kbd> to close
          </span>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body help-body">
          <div className="help-masonry">
            {sections.map((s) => (
              <section key={s.title} className="help-card">
                <h4 className="help-card-head">
                  <span className="help-card-ico">{s.icon}</span>
                  {s.title}
                </h4>
                {s.rows.map((r, i) => (
                  <div key={i} className="help-line">
                    <span className="help-ico">{r.icon}</span>
                    <kbd className="help-key">{r.keys}</kbd>
                    <span className="help-desc">{r.desc}</span>
                  </div>
                ))}
              </section>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

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
      { icon: "📫", keys: fmtKey(k.searchFrom), desc: "Search from this sender" },
      { icon: "📤", keys: fmtKey(k.searchTo), desc: "Search to this recipient" },
      { icon: "🧵", keys: fmtKey(k.searchSubject), desc: "Search this subject" },
      { icon: "🔎", keys: ":advanced", desc: "Advanced search builder" },
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
      { icon: "🔀", keys: fmtKey(k.markdown), desc: "Toggle HTML / text" },
      { icon: "📄", keys: fmtKey(k.toggleHeaders), desc: "Toggle headers" },
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
      { icon: "❌", keys: "Esc", desc: "Exit bulk mode" },
    ],
  });

  if (f.ai) {
    secs.push({
      icon: "🤖",
      title: "AI",
      rows: [
        { icon: "📝", keys: fmtKey(k.summarize), desc: "Summarize" },
        { icon: "🔄", keys: "regenerate", desc: "Regenerate (ignore cache)" },
        ...(f.prompts
          ? [{ icon: "🎯", keys: fmtKey(k.prompt), desc: "Apply a prompt" }]
          : []),
        { icon: "✒️", keys: "⋯ / :draft", desc: "Draft reply (AI)" },
        { icon: "✨", keys: ":touch-up", desc: "Reformat message (AI)" },
        { icon: "🔖", keys: fmtKey(k.suggestLabel), desc: "Suggest labels (AI)" },
        ...(f.prompts
          ? [{ icon: "⚙️", keys: ":prompts", desc: "Manage prompts" }]
          : []),
        ...(f.actionPlan
          ? [
              { icon: "🧠", keys: fmtKey(k.actionPlan), desc: "Inbox action plan" },
              { icon: "📐", keys: ":rules", desc: "Analyzer rules" },
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
        { icon: "🔄", keys: "⋯ / :threads", desc: "Toggle conversation" },
        { icon: "📂", keys: "click", desc: "Expand / collapse a message" },
        { icon: "📤", keys: "Expand all", desc: "Expand / collapse all" },
        ...(f.ai
          ? [{ icon: "🧵", keys: "✦ Summarize", desc: "Summarize thread" }]
          : []),
      ],
    });
  }

  const tools: Row[] = [
    { icon: "💾", keys: fmtKey(k.saveMessage), desc: "Save to file" },
    { icon: "📄", keys: ":save-raw", desc: "Save raw .eml" },
  ];
  if (f.obsidian) tools.push({ icon: "📝", keys: fmtKey(k.obsidian), desc: "Send to Obsidian" });
  if (f.slack) tools.push({ icon: "💬", keys: fmtKey(k.slack), desc: "Forward to Slack" });
  if (f.rsvp) tools.push({ icon: "📅", keys: ":accept / :decline", desc: "RSVP to invites" });
  if (f.themes) tools.push({ icon: "🎨", keys: fmtKey(k.themePicker), desc: "Theme picker" });
  tools.push({ icon: "🕐", keys: ":autorefresh", desc: "Toggle auto-refresh" });
  tools.push({ icon: "▤", keys: ":toolbar", desc: "Show / hide reader toolbar" });
  tools.push({ icon: "📊", keys: ":stats", desc: "AI usage" });
  tools.push({ icon: "🛠️", keys: ":config", desc: "Configuration" });
  tools.push({ icon: "⌨️", keys: fmtKey(k.commandMode), desc: "Command palette" });
  tools.push({ icon: "❓", keys: fmtKey(k.help), desc: "This help" });
  secs.push({ icon: "🔧", title: "Tools & view", rows: tools });

  return secs;
}

export default function Help({
  keymap,
  flags,
  onClose,
}: {
  keymap: KeyMap;
  flags: HelpFlags;
  onClose: () => void;
}) {
  const sections = buildSections(keymap, flags);
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal help-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>Keyboard shortcuts</h3>
          <span className="help-hint muted">
            shows your configured keys · press <kbd>Esc</kbd> to close
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

// Keyboard shortcut reference. These mirror GizTUI's TUI default key bindings
// so muscle memory carries over between the terminal and desktop clients.
export const SHORTCUTS: { group: string; keys: [string, string][] }[] = [
  {
    group: "Navigation",
    keys: [
      ["j / ↓", "Next message"],
      ["k / ↑", "Previous message"],
      ["Enter", "Open message"],
      ["g g", "Go to top"],
      ["G", "Go to bottom"],
      ["N", "Load more"],
      ["R", "Refresh inbox"],
      ["M", "Toggle HTML / text"],
      ["Esc", "Close panel / back"],
    ],
  },
  {
    group: "Triage",
    keys: [
      ["a", "Archive"],
      ["d", "Trash"],
      ["t", "Toggle read / unread"],
      ["l", "Labels"],
      ["U", "Undo last action"],
    ],
  },
  {
    group: "Compose",
    keys: [
      ["c", "Compose new"],
      ["r", "Reply"],
      ["E", "Reply all"],
      ["f", "Forward"],
      ["g", "Draft reply (AI) — ⋯ menu / :draft"],
      ["D", "Drafts"],
      ["O", "Open in Gmail"],
    ],
  },
  {
    group: "Organize & find",
    keys: [
      ["m", "Move to folder"],
      ["h", "Toggle headers"],
      ["/", "Find in message"],
      ["F", "Search from sender"],
      ["T", "Search to recipient"],
      ["S", "Search this subject"],
    ],
  },
  {
    group: "Select / bulk",
    keys: [
      ["v", "Toggle select mode"],
      ["Space", "Toggle selection"],
      ["*", "Select all"],
      ["a / d", "Bulk archive / trash"],
      ["t", "Bulk mark unread"],
      ["l", "Bulk label"],
    ],
  },
  {
    group: "AI & search",
    keys: [
      ["y", "Summarize (AI)"],
      ["p", "Apply a prompt"],
      [":prompts", "Manage prompts (new/edit/refine)"],
      [":touch-up", "Reformat message with AI"],
      [":advanced", "Advanced search builder"],
      [":local", "Toggle local filter / Gmail"],
      ["o", "Suggest labels (AI)"],
      ["P", "AI inbox action plan"],
      [":rules", "Analyzer preference rules"],
      ["s", "Search inbox"],
      [":", "Command mode"],
      ["?", "This help"],
    ],
  },
  {
    group: "Tools & view",
    keys: [
      ["L", "Links in message"],
      ["w", "Save to file"],
      ["O", "Send to Obsidian"],
      ["K", "Forward to Slack"],
      ["⋯ / :threads", "Toggle conversation"],
      ["Q", "Saved searches"],
      ["Z", "Save current search"],
      ["H", "Theme picker"],
      [":toolbar", "Show/hide reader toolbar"],
    ],
  },
];

export default function Help({ onClose }: { onClose: () => void }) {
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>Keyboard shortcuts</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body help-grid">
          {SHORTCUTS.map((s) => (
            <div key={s.group} className="help-col">
              <h4>{s.group}</h4>
              {s.keys.map(([k, desc]) => (
                <div key={k} className="help-row">
                  <kbd>{k}</kbd>
                  <span>{desc}</span>
                </div>
              ))}
            </div>
          ))}
        </div>
        <div className="modal-foot">
          <button onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  );
}

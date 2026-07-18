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
    ],
  },
  {
    group: "Compose",
    keys: [
      ["c", "Compose new"],
      ["r", "Reply"],
      ["f", "Forward"],
      ["D", "Drafts"],
      ["O", "Open in Gmail"],
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
      ["s /  /", "Search"],
      ["?", "This help"],
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

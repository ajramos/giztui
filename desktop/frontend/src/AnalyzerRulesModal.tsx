import { useListNav } from "./useListNav";
import { usePickerCrud } from "./usePickerCrud";
import { useConfirm } from "./useConfirm";
import { Icon } from "./Icons";
import type { AnalyzerRule } from "./api";

// The analyzer preference rules (opened by :action-plan rules). Keyboard-first
// per the picker premises: useListNav drives ↑↓ from the window (WKWebView won't
// focus the modal), and usePickerCrud removes the highlighted rule with the
// shared convention (d/Delete/Backspace, or Shift+Delete while typing in the
// add-rule input). The `rules` / `newRule` state stays in App (this is a
// behavior-preserving move); only the nav + delete-key wiring live here.
export default function AnalyzerRulesModal({
  rules,
  newRule,
  onNewRuleChange,
  onAddRule,
  onDeleteRule,
  onClose,
}: {
  rules: AnalyzerRule[];
  newRule: string;
  onNewRuleChange: (value: string) => void;
  onAddRule: () => void;
  onDeleteRule: (id: number) => void;
  onClose: () => void;
}) {
  const nav = useListNav(rules, { onEscape: onClose, windowKeys: true });
  const confirm = useConfirm();

  const askDelete = (r: AnalyzerRule) =>
    confirm.ask(`Delete this rule?\n\n“${r.text}”`, () => onDeleteRule(r.id));

  usePickerCrud(confirm.open ? [] : rules, nav.active, { onDelete: askDelete });

  return (
    <>
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal narrow"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          if (e.key === "Escape") onClose();
        }}
      >
        <div className="modal-head">
          <h3>Analyzer rules</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          <div className="muted plan-summary">
            Natural-language preferences the analyzer follows when planning.
          </div>
          <div className="label-list" ref={nav.listRef}>
            {rules.length === 0 ? (
              <div className="placeholder">No rules yet</div>
            ) : (
              rules.map((r, i) => (
                <div
                  key={r.id}
                  className={
                    "prompt-manage-row" + (i === nav.active ? " nav-active" : "")
                  }
                  onMouseEnter={() => nav.setActiveHover(i)}
                >
                  <span className="rule-text">{r.text}</span>
                  <button
                    className="ghost tiny danger"
                    title="Delete"
                    onClick={() => askDelete(r)}
                  >
                    {Icon.trash}
                  </button>
                </div>
              ))
            )}
          </div>
          <div className="field">
            <input
              value={newRule}
              onChange={(e) => onNewRuleChange(e.target.value)}
              placeholder="e.g. Always archive newsletters"
              onKeyDown={(e) => {
                if (e.key === "Enter") onAddRule();
              }}
            />
          </div>
        </div>
        <div className="modal-foot">
          <span className="foot-hint">↑↓ move · d / ⇧⌫ delete · Esc close</span>
          <button onClick={onAddRule} disabled={!newRule.trim()}>
            Add rule
          </button>
        </div>
      </div>
    </div>
    {confirm.node}
    </>
  );
}

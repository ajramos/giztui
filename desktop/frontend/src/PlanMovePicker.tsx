import { useEffect, useRef } from "react";
import { useListNav } from "./useListNav";
import type { MoveTarget } from "./planMove";

// PlanMovePicker chooses a destination when reassigning an email (or a whole
// category) to another action-plan bucket. Keyboard-first per the desktop picker
// premises: 1-9 quick-select, arrows/Enter/Escape via useListNav windowKeys.
export default function PlanMovePicker({
  title,
  targets,
  onChoose,
  onClose,
}: {
  title: string;
  targets: MoveTarget[];
  onChoose: (t: MoveTarget) => void;
  onClose: () => void;
}) {
  const nav = useListNav(targets, {
    onEnter: (t) => onChoose(t),
    onEscape: onClose,
    windowKeys: true,
  });
  const targetsRef = useRef(targets);
  targetsRef.current = targets;

  // 1-9 pick a destination directly (the TUI numbers the first nine).
  useEffect(() => {
    const h = (e: globalThis.KeyboardEvent) => {
      if (e.key >= "1" && e.key <= "9") {
        const i = e.key.charCodeAt(0) - 49;
        if (i < targetsRef.current.length) {
          e.preventDefault();
          onChoose(targetsRef.current[i]);
        }
      }
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [onChoose]);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>{title}</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          <div className="label-list" ref={nav.listRef}>
            {targets.map((t, i) => (
              <button
                key={t.label}
                className={"label-row" + (i === nav.active ? " nav-active" : "")}
                onMouseEnter={() => nav.setActiveHover(i)}
                onClick={() => onChoose(t)}
              >
                <span className="name">
                  {i < 9 && <span className="link-idx">[{i + 1}]</span>}{" "}
                  {t.label}
                </span>
              </button>
            ))}
          </div>
        </div>
        <div className="modal-foot">
          <span className="foot-hint">
            1-9 or ↑↓ move · Enter choose · Esc back
          </span>
        </div>
      </div>
    </div>
  );
}

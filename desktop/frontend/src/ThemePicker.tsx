import { useEffect, useRef } from "react";
import { useListNav } from "./useListNav";

export default function ThemePicker({
  themes,
  current,
  onPick,
  onClose,
}: {
  themes: string[];
  current: string;
  onPick: (name: string) => void;
  onClose: () => void;
}) {
  const modalRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    modalRef.current?.focus();
  }, []);

  const nav = useListNav(themes, { onEnter: onPick, onEscape: onClose });

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        ref={modalRef}
        className="modal narrow"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={nav.onKeyDown}
        tabIndex={-1}
      >
        <div className="modal-head">
          <h3>Theme</h3>
          <span className="help-hint muted">↑↓ · Enter · Esc</span>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          <div className="theme-list" ref={nav.listRef}>
            {themes.map((name, i) => (
              <button
                key={name}
                className={
                  "theme-item" +
                  (name === current ? " active" : "") +
                  (i === nav.active ? " nav-active" : "")
                }
                onMouseEnter={() => nav.setActive(i)}
                onClick={() => onPick(name)}
              >
                <span className="theme-dot" />
                {name}
                {name === current && <span className="theme-check">✓</span>}
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

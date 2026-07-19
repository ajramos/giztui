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
  const nav = useListNav(themes, {
    onEnter: onPick,
    onEscape: onClose,
    windowKeys: true,
  });

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
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

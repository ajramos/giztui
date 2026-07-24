import { useEffect, useRef, useState, type ReactNode } from "react";
import { Icon } from "./Icons";

export interface MoreItem {
  icon: ReactNode;
  label: string;
  onClick: () => void;
  disabled?: boolean;
  hidden?: boolean;
}

// A "⋯" overflow menu that keeps the reader toolbar compact: primary actions
// stay as icons, secondary ones live here as labeled rows.
export default function MoreMenu({ items }: { items: MoreItem[] }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, []);

  const visible = items.filter((i) => !i.hidden);
  if (visible.length === 0) return null;

  return (
    <div className="more-menu" ref={ref}>
      <button
        type="button"
        className={"icon-btn" + (open ? " primary" : "")}
        data-tip="More"
        aria-label="More actions"
        onClick={() => setOpen((o) => !o)}
      >
        {Icon.more}
      </button>
      {open && (
        <div className="more-list">
          {visible.map((it, i) => (
            <button
              key={i}
              className="more-item"
              disabled={it.disabled}
              onClick={() => {
                setOpen(false);
                it.onClick();
              }}
            >
              <span className="more-ico">{it.icon}</span>
              {it.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

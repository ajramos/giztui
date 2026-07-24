import { useEffect, useRef } from "react";
import { useListNav } from "./useListNav";
import type { AccountInfo } from "./api";

export default function AccountSwitcher({
  accounts,
  email,
  switching,
  onSwitch,
  open: controlledOpen,
  onOpenChange,
}: {
  accounts: AccountInfo[];
  email: string;
  switching: boolean;
  onSwitch: (a: AccountInfo) => void;
  // Optional controlled open state so a keyboard shortcut (Ctrl+A) can drive it.
  open?: boolean;
  onOpenChange?: (v: boolean) => void;
}) {
  const open = controlledOpen ?? false;
  const setOpen = (v: boolean) => onOpenChange?.(v);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Keyboard-first like every other picker: arrows move, Enter switches, Escape
  // closes. windowKeys is gated on `open` so the listener is only live while the
  // dropdown is showing (App adds accountsOpen to its modal guard so the inbox
  // list nav doesn't also fire). 1-9 quick-select mirrors the TUI.
  const nav = useListNav(accounts, {
    onEnter: (a) => {
      setOpen(false);
      if (!a.active) onSwitch(a);
    },
    onEscape: () => setOpen(false),
    windowKeys: open,
  });
  useEffect(() => {
    if (!open) return;
    const h = (e: globalThis.KeyboardEvent) => {
      if (e.key >= "1" && e.key <= "9") {
        const i = Number(e.key) - 1;
        if (i < accounts.length) {
          e.preventDefault();
          setOpen(false);
          if (!accounts[i].active) onSwitch(accounts[i]);
        }
      }
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, accounts, onSwitch]);

  // With a single account there is nothing to switch between.
  if (accounts.length <= 1) {
    return <span className="email">{email}</span>;
  }

  return (
    <div className="acct-switcher" ref={ref}>
      <button
        className="ghost acct-btn"
        disabled={switching}
        onClick={() => setOpen(!open)}
        title="Switch account"
      >
        {switching ? "Switching…" : email} ▾
      </button>
      {open && (
        <div className="acct-menu" ref={nav.listRef}>
          {accounts.map((a, i) => (
            <button
              key={a.id}
              className={
                "acct-item" +
                (a.active ? " active" : "") +
                (i === nav.active ? " nav-active" : "")
              }
              onMouseEnter={() => nav.setActiveHover(i)}
              onClick={() => {
                setOpen(false);
                if (!a.active) onSwitch(a);
              }}
            >
              <span className="acct-check">{a.active ? "✓" : ""}</span>
              {i < 9 && <span className="link-idx">[{i + 1}]</span>}
              <span className="acct-text">
                <span className="acct-name">
                  {a.displayName || a.email || a.id}
                </span>
                {a.email && a.email !== a.displayName && (
                  <span className="acct-email">{a.email}</span>
                )}
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

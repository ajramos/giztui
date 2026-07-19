import { useEffect, useRef, useState } from "react";
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
  const [uncontrolled, setUncontrolled] = useState(false);
  const open = controlledOpen ?? uncontrolled;
  const setOpen = (v: boolean) => {
    setUncontrolled(v);
    onOpenChange?.(v);
  };
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, []);

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
        <div className="acct-menu">
          {accounts.map((a) => (
            <button
              key={a.id}
              className={"acct-item" + (a.active ? " active" : "")}
              onClick={() => {
                setOpen(false);
                if (!a.active) onSwitch(a);
              }}
            >
              <span className="acct-check">{a.active ? "✓" : ""}</span>
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

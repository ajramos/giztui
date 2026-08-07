import { useEffect, useRef, useState } from "react";

// useFilterMode implements the "list mode by default, '/' to filter" behaviour
// shared by pickers that have BOTH a filter and letter shortcuts (saved searches,
// prompts). The picker opens in list mode with the filter blurred, so bare e / d /
// n act as CRUD without colliding with typing. Pressing "/" focuses the filter for
// free-text entry (uppercase and all); Escape returns to list mode (a second
// Escape then closes the picker via useListNav). `active` gates the shortcuts so
// they're inert while a stacked dialog (edit / confirm) is open. Optional onNew
// binds bare "n" to create in list mode.
export function useFilterMode(active: boolean, onNew?: () => void) {
  const [filtering, setFiltering] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const start = () => {
    setFiltering(true);
    inputRef.current?.focus();
  };
  const stop = () => {
    setFiltering(false);
    inputRef.current?.blur();
  };

  const activeRef = useRef(active);
  activeRef.current = active;
  const filteringRef = useRef(filtering);
  filteringRef.current = filtering;
  const onNewRef = useRef(onNew);
  onNewRef.current = onNew;

  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (!activeRef.current || filteringRef.current) return;
      if (e.key === "/") {
        e.preventDefault();
        e.stopImmediatePropagation();
        start();
      } else if (onNewRef.current && (e.key === "n" || e.key === "N")) {
        e.preventDefault();
        e.stopImmediatePropagation();
        onNewRef.current();
      }
    };
    window.addEventListener("keydown", h, true);
    return () => window.removeEventListener("keydown", h, true);
  }, []);

  // Spread onto the filter <input>: a ref to focus/blur it, Escape to leave filter
  // mode (without closing the picker), and onBlur to keep state in sync when the
  // pointer moves focus away.
  const inputProps = {
    ref: inputRef,
    onFocus: () => setFiltering(true),
    onKeyDown: (e: React.KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopPropagation();
        stop();
      }
    },
    onBlur: () => setFiltering(false),
  };
  return { filtering, start, stop, inputProps };
}

import { useEffect, useRef, useState, type KeyboardEvent } from "react";

// useListNav gives a picker keyboard control: an `active` index moved with
// ArrowUp/Down (and Ctrl+P/N), Enter to act on the active item, Escape to close.
// Attach onKeyDown to the focused input (or the modal), spread the active row
// with the "nav-active" class, and set listRef on the scroll container so the
// active row scrolls into view. Everything stays keyboard-first.
export function useListNav<T>(
  items: T[],
  opts: {
    onEnter?: (item: T, index: number) => void;
    onEscape?: () => void;
    // Bind the keyboard handler at the window instead of (only) an element.
    // WKWebView won't reliably focus a bare div, so pickers without a text input
    // must not depend on element focus. Gate it on the picker being open so the
    // listener isn't live when the picker is closed (matters for pickers whose
    // hook lives in a always-mounted parent).
    windowKeys?: boolean;
  } = {},
) {
  const [active, setActive] = useState(0);
  const listRef = useRef<HTMLDivElement>(null);
  const windowKeys = opts.windowKeys ?? false;

  // Keep the active index within bounds as the (filtered) list changes.
  useEffect(() => {
    setActive((a) => {
      if (items.length === 0) return 0;
      return Math.max(0, Math.min(a, items.length - 1));
    });
  }, [items.length]);

  // Scroll the active row into view when it moves.
  useEffect(() => {
    listRef.current
      ?.querySelector<HTMLElement>(".nav-active")
      ?.scrollIntoView({ block: "nearest" });
  }, [active]);

  const onKeyDown = (e: KeyboardEvent) => {
    if (e.key === "ArrowDown" || (e.ctrlKey && e.key === "n")) {
      e.preventDefault();
      hoverArmed.current = false;
      setActive((a) => (items.length ? Math.min(items.length - 1, a + 1) : 0));
    } else if (e.key === "ArrowUp" || (e.ctrlKey && e.key === "p")) {
      e.preventDefault();
      hoverArmed.current = false;
      setActive((a) => Math.max(0, a - 1));
    } else if (e.key === "Enter") {
      e.preventDefault();
      const item = items[active];
      if (item !== undefined) opts.onEnter?.(item, active);
    } else if (e.key === "Escape") {
      e.preventDefault();
      opts.onEscape?.();
    }
  };

  // Window-level binding for focus-less pickers. A ref keeps the listener bound
  // once while always calling the freshest handler (which closes over `active`
  // and `items`), so arrow/enter stay correct without re-subscribing.
  const handlerRef = useRef(onKeyDown);
  handlerRef.current = onKeyDown;
  useEffect(() => {
    if (!windowKeys) return;
    const h = (e: globalThis.KeyboardEvent) =>
      handlerRef.current(e as unknown as KeyboardEvent);
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [windowKeys]);

  // Keyboard nav disarms hover; a real pointer move re-arms it. Without this, a
  // keyboard move that scrolls the list slides a row under the stationary cursor,
  // firing mouseenter → setActive, which fights the keyboard (the cursor gets
  // "trapped" in a long, scrolling list). setActiveHover honors hover only when
  // the pointer has genuinely moved since the last keyboard nav.
  const hoverArmed = useRef(true);
  useEffect(() => {
    const onMove = () => {
      hoverArmed.current = true;
    };
    window.addEventListener("pointermove", onMove);
    return () => window.removeEventListener("pointermove", onMove);
  }, []);
  const setActiveHover = (i: number) => {
    if (hoverArmed.current && i >= 0) setActive(i);
  };

  return { active, setActive, setActiveHover, onKeyDown, listRef };
}

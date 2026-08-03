import { useEffect, useRef } from "react";

// pickerCrudAction is the pure decision behind usePickerCrud: given a key, the
// Shift state, and whether a text field is focused, it returns which CRUD action
// (if any) the keypress maps to. Kept separate so it can be unit-tested without a
// DOM. The bare keys act only when not typing; the Shift variants act always.
export function pickerCrudAction(
  key: string,
  shiftKey: boolean,
  typing: boolean,
): "edit" | "delete" | null {
  // While typing, only the Shift variants are allowed to act.
  if (typing && !shiftKey) return null;
  if (key === "e" || key === "E") return "edit";
  if (key === "d" || key === "D" || key === "Delete" || key === "Backspace")
    return "delete";
  return null;
}

// typingInTextField reports whether keyboard focus is currently in a text field
// (a filter box or an edit form). While the user is typing there, a bare "e"/"d"
// must reach the field, so edit/delete require the Shift escape hatch instead.
function typingInTextField(): boolean {
  const el = document.activeElement as HTMLElement | null;
  if (!el) return false;
  return (
    el.tagName === "INPUT" || el.tagName === "TEXTAREA" || el.isContentEditable
  );
}

// usePickerCrud standardizes keyboard edit/delete across every picker whose rows
// are editable/deletable entities (saved searches, prompts, analyzer rules,
// deterministic rules), so they all behave the same and mirror the TUI's e/d:
//
//   • e                              → edit the highlighted row
//   • d / Delete / Backspace         → delete the highlighted row
//   • Shift+E                        → edit    (always — the escape hatch)
//   • Shift+Delete / Shift+Backspace → delete  (always)
//
// The bare keys act only when no text field is focused; when a filter or edit
// form holds focus (so bare keys are needed for typing) the Shift variants take
// over. Pickers with an always-focused filter (saved searches) therefore use the
// Shift variants exclusively, while pickers where the list itself holds focus can
// use the bare keys — one rule, a superset of both old behaviours.
//
// Either handler may be omitted (e.g. delete-only modals). The listener runs in
// the capture phase and stops propagation only for the keys it consumes, so it
// beats the App's global window handler without swallowing arrows/Enter/Escape
// that the picker's own useListNav still needs. Refs keep it registered once
// while always reading the freshest list + active index.
export function usePickerCrud<T>(
  items: T[],
  active: number,
  opts: { onEdit?: (item: T) => void; onDelete?: (item: T) => void },
): void {
  const ref = useRef({ items, active, opts });
  ref.current = { items, active, opts };

  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      const { items, active, opts } = ref.current;
      const item = items[active];
      if (item === undefined) return;

      const action = pickerCrudAction(e.key, e.shiftKey, typingInTextField());
      const handler =
        action === "edit" ? opts.onEdit : action === "delete" ? opts.onDelete : undefined;
      if (!handler) return;
      e.preventDefault();
      e.stopImmediatePropagation();
      handler(item);
    };
    window.addEventListener("keydown", h, true);
    return () => window.removeEventListener("keydown", h, true);
  }, []);
}

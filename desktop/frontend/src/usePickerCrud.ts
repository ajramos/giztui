import { useEffect, useRef } from "react";

// pickerCrudAction is the pure decision behind usePickerCrud: given a key and
// whether a text field is focused, it returns which CRUD action (if any) the
// keypress maps to. Kept separate so it can be unit-tested without a DOM.
//
// While a text field is focused NOTHING fires — every key is the user typing
// (a capital "E"/"D" is just an uppercase letter, not a command). CRUD keys work
// only in "list mode", where no input is focused: bare e edits, d/Delete/
// Backspace deletes. Pickers with a filter open in list mode and enter a focused
// filter with "/", so the two never collide.
export function pickerCrudAction(
  key: string,
  typing: boolean,
): "edit" | "delete" | null {
  if (typing) return null;
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
// deterministic rules, jobs), so they all behave the same and mirror the TUI:
//
//   • e                      → edit the highlighted row
//   • d / Delete / Backspace → delete the highlighted row
//
// These fire only in "list mode" — when no text input is focused. A picker with a
// filter opens in list mode and focuses its filter only when the user presses "/";
// while the filter is focused every key is typing (so a capital "E" is a letter,
// not a command) and CRUD is unavailable until "/"-mode is exited with Escape.
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

      const action = pickerCrudAction(e.key, typingInTextField());
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

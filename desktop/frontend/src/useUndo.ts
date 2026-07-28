import { useCallback, useRef, useState } from "react";

// useUndo owns the undo stack: a bounded LIFO of {label, run} entries. Handlers
// (archive/trash/move/…) push a reversal closure; U (or the topbar button) runs
// the most recent. Extracted from App.tsx unchanged — the stack lives in a ref
// so pushing never re-renders, and undoLabel drives the button label.
export interface Undo {
  undoLabel: string;
  pushUndo: (label: string, run: () => Promise<void>) => void;
  runUndo: () => Promise<void>;
}

export function useUndo(deps: {
  showToast: (m: string) => void;
  setError: (e: string) => void;
}): Undo {
  const { showToast, setError } = deps;
  const undoRef = useRef<{ label: string; run: () => Promise<void> }[]>([]);
  const [undoLabel, setUndoLabel] = useState("");

  const pushUndo = useCallback(
    (label: string, run: () => Promise<void>) => {
      undoRef.current.push({ label, run });
      if (undoRef.current.length > 25) undoRef.current.shift();
      setUndoLabel(label);
    },
    [],
  );

  const runUndo = useCallback(async () => {
    const entry = undoRef.current.pop();
    setUndoLabel(
      undoRef.current.length
        ? undoRef.current[undoRef.current.length - 1].label
        : "",
    );
    if (!entry) {
      showToast("Nothing to undo");
      return;
    }
    try {
      await entry.run();
      showToast(`Undone: ${entry.label}`);
    } catch (e) {
      setError(String(e));
    }
  }, [showToast, setError]);

  return { undoLabel, pushUndo, runUndo };
}

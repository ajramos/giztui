import { useCallback, useEffect, useRef } from "react";
import { handleKeyDown } from "./keydownHandler";
import { runCommand } from "./commandRunner";
import type { KeydownCtx } from "./keydownCtx";
import type { CommandCtx } from "./commandCtx";

// useAppWiring owns the two global input surfaces that both need "everything":
// the window keydown listener and the command runner. It takes the merged
// context (a superset of both KeydownCtx and CommandCtx), stashes it in a ref
// refreshed every render, and dispatches through that ref — so the listener is
// registered once and executeCommand stays stable, while both always read fresh
// state. handleKeyDown/runCommand only read their own fields from the superset.
export function useAppWiring(ctx: KeydownCtx & CommandCtx) {
  const ref = useRef(ctx);
  ref.current = ctx;

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => handleKeyDown(e, ref.current);
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const executeCommand = useCallback(
    (input: string) => runCommand(input, ref.current),
    [],
  );

  return { executeCommand };
}

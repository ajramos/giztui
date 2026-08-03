import { forwardRef } from "react";
import Markdown from "./Markdown";

// AiPanel renders one AI output block in the reader — the summary panel and the
// per-prompt result panel are structurally identical (header with title +
// regenerate/dismiss actions, then a streaming Markdown body with a caret while
// generating). Extracted so the two stay in lockstep. `generating` is already
// scoped to the open message by the caller (summaryForId/promptForId), so this
// component stays presentational.
const AiPanel = forwardRef<
  HTMLDivElement,
  {
    title: string;
    text: string | null;
    generating: boolean;
    onRegenerate: () => void;
    onDismiss: () => void;
    regenerateTitle?: string;
    dismissTitle?: string;
    className?: string;
  }
>(function AiPanel(
  {
    title,
    text,
    generating,
    onRegenerate,
    onDismiss,
    regenerateTitle,
    dismissTitle,
    className,
  },
  ref,
) {
  if (text === null && !generating) return null;
  const showActions = text !== null && !generating;
  return (
    <div className={"summary-panel" + (className ? " " + className : "")} ref={ref}>
      <div className="summary-head">
        {/* Fall back to a generic label so a restored result with a missing
            title (e.g. a DB-cached prompt whose name wasn't persisted) never
            renders as a bare "✦" with no heading. */}
        <span>✦ {title || "Prompt result"}</span>
        <span className="summary-head-actions">
          {showActions && (
            <button
              className="ghost tiny"
              title={regenerateTitle}
              onClick={onRegenerate}
            >
              regenerate
            </button>
          )}
          {showActions && (
            <button className="ghost tiny" title={dismissTitle} onClick={onDismiss}>
              dismiss
            </button>
          )}
        </span>
      </div>
      {generating && !text ? (
        <div className="muted">Generating…</div>
      ) : (
        <div className="summary-text">
          <Markdown text={text || ""} />
          {generating && <span className="caret">▍</span>}
        </div>
      )}
    </div>
  );
});

export default AiPanel;

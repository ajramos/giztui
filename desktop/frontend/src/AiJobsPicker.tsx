import { useEffect } from "react";
import { useListNav } from "./useListNav";
import { usePickerCrud } from "./usePickerCrud";
import type { AiJob } from "./useAiJobs";

// Human-readable, one-glyph-free status label. Kept as text (not an emoji) per
// the desktop picker conventions; colored via the data-status attribute in CSS.
const STATUS_LABEL: Record<AiJob["status"], string> = {
  queued: "queued",
  running: "running",
  done: "done",
  error: "error",
};

function age(ms: number): string {
  const s = Math.max(0, Math.round((Date.now() - ms) / 1000));
  if (s < 60) return `${s}s ago`;
  const m = Math.round(s / 60);
  if (m < 60) return `${m}m ago`;
  return `${Math.round(m / 60)}h ago`;
}

// The AI background-jobs picker (":jobs"): browse running/finished jobs, re-open
// a result, remove one, or clear the finished ones. Keyboard-first and
// focus-independent (window-level nav) like the other pickers.
export default function AiJobsPicker({
  jobs,
  onClose,
  onOpen,
  onRemove,
  onClear,
}: {
  jobs: AiJob[];
  onClose: () => void;
  onOpen: (id: string) => void;
  onRemove: (id: string) => void;
  onClear: () => void;
}) {
  // Newest first so the most recent job is at the top (and highlighted).
  const ordered = [...jobs].sort((a, b) => b.createdAt - a.createdAt);

  const nav = useListNav(ordered, {
    onEnter: (j) => {
      onOpen(j.id);
      onClose();
    },
    onEscape: onClose,
    windowKeys: true,
  });

  // Remove the highlighted job via the shared CRUD convention (d / Delete /
  // Backspace / Shift+Del), same as every other list that deletes an entity.
  usePickerCrud(ordered, nav.active, { onDelete: (j) => onRemove(j.id) });

  // 'c' clears all finished jobs — a bulk action distinct from the per-row delete.
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (e.key === "c" || e.key === "C") {
        e.preventDefault();
        e.stopImmediatePropagation();
        onClear();
      }
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [onClear]);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>AI jobs</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          <div className="label-list" ref={nav.listRef}>
            {ordered.length === 0 ? (
              <div className="placeholder">No AI jobs yet</div>
            ) : (
              ordered.map((j, i) => (
                <button
                  key={j.id}
                  className={"prompt-row" + (i === nav.active ? " nav-active" : "")}
                  onMouseEnter={() => nav.setActiveHover(i)}
                  onClick={() => {
                    onOpen(j.id);
                    onClose();
                  }}
                >
                  <span className="prompt-name">
                    <span className="job-status" data-status={j.status}>
                      {STATUS_LABEL[j.status]}
                    </span>{" "}
                    {j.label}
                  </span>
                  <span className="prompt-desc">
                    {j.status === "error" && j.error
                      ? j.error
                      : `${age(j.createdAt)}`}
                  </span>
                </button>
              ))
            )}
          </div>
        </div>
        <div className="modal-foot">
          <span className="foot-hint">
            ↑↓ move · Enter open · d remove · c clear finished · Esc close
          </span>
        </div>
      </div>
    </div>
  );
}

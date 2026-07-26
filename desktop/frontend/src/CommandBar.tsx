import { useMemo, useState } from "react";
import { type CommandDef, filterCommands, resolveEnter } from "./commands";

export type { CommandDef };

export default function CommandBar({
  commands,
  onRun,
  onClose,
}: {
  commands: CommandDef[];
  onRun: (input: string) => void;
  onClose: () => void;
}) {
  const [input, setInput] = useState("");
  const [active, setActive] = useState(0);

  const matches = useMemo(
    () => filterCommands(commands, input),
    [commands, input],
  );

  const submit = (value: string) => {
    const v = value.trim();
    if (v) onRun(v);
    onClose();
  };

  return (
    <div className="modal-overlay cmd-overlay" onClick={onClose}>
      <div
        className="cmd-bar"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          // Keep keys inside the command bar. Otherwise the Enter that runs a
          // command bubbles to window and is caught by a picker this command
          // just opened (its window-level nav listener), instantly selecting the
          // first item — e.g. :theme applied a theme instead of showing the list.
          e.stopPropagation();
          if (e.key === "Escape") {
            onClose();
          } else if (e.key === "ArrowDown") {
            e.preventDefault();
            setActive((i) => Math.min(matches.length - 1, i + 1));
          } else if (e.key === "ArrowUp") {
            e.preventDefault();
            setActive((i) => Math.max(0, i - 1));
          } else if (e.key === "Tab") {
            e.preventDefault();
            if (matches[active]) setInput(matches[active].names[0] + " ");
          } else if (e.key === "Enter") {
            e.preventDefault();
            // Exact command → run input as-is (keeps args); else the highlighted
            // suggestion; else the raw input (numeric jumps like :5 / :$).
            submit(resolveEnter(commands, input, active));
          }
        }}
      >
        <div className="cmd-input-row">
          <span className="cmd-prompt">:</span>
          <input
            className="cmd-input"
            value={input}
            onChange={(e) => {
              setInput(e.target.value);
              setActive(0);
            }}
            placeholder="type a command… (e.g. search from:me, archive, labels)"
            autoFocus
          />
        </div>
        <div className="cmd-list">
          {matches.length === 0 ? (
            <div className="placeholder">No matching command</div>
          ) : (
            matches.slice(0, 8).map((c, i) => (
              <button
                key={c.names[0]}
                className={"cmd-item" + (i === active ? " active" : "")}
                onMouseEnter={() => setActive(i)}
                onClick={() => submit(c.names[0])}
              >
                <span className="cmd-name">
                  {c.names[0]}
                  {c.arg && <span className="cmd-arg"> {c.arg}</span>}
                </span>
                <span className="cmd-alias">
                  {c.names.length > 1 ? c.names.slice(1).join(", ") : ""}
                </span>
                <span className="cmd-desc">{c.desc}</span>
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

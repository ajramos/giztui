import { useMemo, useState } from "react";

export interface CommandDef {
  names: string[]; // first is canonical, rest are aliases
  desc: string;
  arg?: string; // placeholder hint when the command takes an argument
}

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

  const matches = useMemo(() => {
    const word = input.trim().split(/\s+/)[0]?.toLowerCase() ?? "";
    if (!word) return commands;
    return commands.filter((c) =>
      c.names.some((n) => n.toLowerCase().startsWith(word)),
    );
  }, [commands, input]);

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
            // If the typed word matches a command exactly, run the input as-is
            // (keeps arguments); otherwise run the highlighted suggestion.
            const word = input.trim().split(/\s+/)[0]?.toLowerCase() ?? "";
            const exact = commands.find((c) =>
              c.names.some((n) => n.toLowerCase() === word),
            );
            if (exact) submit(input);
            else if (matches[active]) submit(matches[active].names[0]);
            // No suggestion matched (e.g. a numeric jump like ":5" or ":$") —
            // run the raw input so the dispatcher can still handle it.
            else submit(input);
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

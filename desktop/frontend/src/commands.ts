// Command-palette data + pure resolution logic, extracted from App.tsx and
// CommandBar.tsx so aliasing / arg-parsing / Enter-resolution can be unit-tested
// without React. The big executeCommand switch in App.tsx stays as the handler
// adapter; only the pure pieces live here.

export interface CommandDef {
  names: string[]; // first is canonical, rest are aliases
  desc: string;
  arg?: string; // placeholder hint when the command takes an argument
}

function firstWord(input: string): string {
  return input.trim().split(/\s+/)[0]?.toLowerCase() ?? "";
}

// parseCommand splits ":cmd arg rest" into a lowercased command and the raw
// remainder (used by executeCommand's dispatcher).
export function parseCommand(input: string): { cmd: string; arg: string } {
  const parts = input.trim().split(/\s+/);
  return { cmd: (parts[0] || "").toLowerCase(), arg: parts.slice(1).join(" ") };
}

// filterCommands is the palette autocomplete: every command with a name that
// starts with the typed first word (all of them when nothing is typed).
export function filterCommands(commands: CommandDef[], input: string): CommandDef[] {
  const word = firstWord(input);
  if (!word) return commands;
  return commands.filter((c) => c.names.some((n) => n.toLowerCase().startsWith(word)));
}

// resolveEnter returns the string the CommandBar should run on Enter: the typed
// input verbatim when it exactly names a command (so arguments are preserved),
// otherwise the highlighted suggestion's canonical name, otherwise the raw input
// (so numeric jumps like ":5" / ":$" still reach the dispatcher).
export function resolveEnter(
  commands: CommandDef[],
  input: string,
  activeIndex: number,
): string {
  const word = firstWord(input);
  const exact = commands.find((c) => c.names.some((n) => n.toLowerCase() === word));
  if (exact) return input;
  const matches = filterCommands(commands, input);
  if (matches[activeIndex]) return matches[activeIndex].names[0];
  return input;
}

// Command palette entries (`:` command mode), mirroring the TUI's command set.
export const COMMANDS: CommandDef[] = [
  { names: ["search", "s"], desc: "Gmail search", arg: "<query>" },
  { names: ["unread", "u"], desc: "Show unread only" },
  { names: ["advanced", "adv"], desc: "Advanced search builder" },
  { names: ["local"], desc: "Toggle local filter / Gmail search" },
  { names: ["stats", "usage"], desc: "AI prompt usage stats" },
  { names: ["config", "cfg"], desc: "Show configuration" },
  { names: ["cache"], desc: "Clear AI caches" },
  { names: ["archive", "a"], desc: "Archive message" },
  { names: ["trash", "d"], desc: "Trash message" },
  { names: ["undo"], desc: "Undo last action" },
  { names: ["read"], desc: "Mark read" },
  { names: ["markunread"], desc: "Mark unread" },
  { names: ["toggle-read", "t"], desc: "Toggle read / unread" },
  { names: ["labels", "l"], desc: "Manage labels" },
  { names: ["compose", "c", "new"], desc: "New message" },
  { names: ["reply", "r"], desc: "Reply" },
  { names: ["replyall", "reply-all", "ra"], desc: "Reply all" },
  { names: ["forward", "f"], desc: "Forward" },
  { names: ["refresh"], desc: "Refresh inbox" },
  { names: ["drafts", "dr"], desc: "Drafts" },
  { names: ["links", "link"], desc: "Links in message" },
  { names: ["save"], desc: "Save to file" },
  { names: ["save-raw", "saveraw"], desc: "Save raw .eml" },
  { names: ["rsvp"], desc: "Open RSVP picker for invite" },
  { names: ["accept"], desc: "RSVP: accept invite" },
  { names: ["tentative", "maybe"], desc: "RSVP: tentative" },
  { names: ["decline"], desc: "RSVP: decline invite" },
  { names: ["autorefresh", "arr"], desc: "Toggle inbox auto-refresh" },
  { names: ["summarize", "sum", "summary"], desc: "AI summary" },
  { names: ["prompt", "pr", "p"], desc: "Apply a prompt" },
  { names: ["prompts", "prompt-new"], desc: "Manage prompts" },
  { names: ["suggest"], desc: "Suggest labels (AI)" },
  { names: ["obsidian", "obs"], desc: "Send to Obsidian" },
  { names: ["slack", "sl"], desc: "Forward to Slack" },
  { names: ["gmail", "web", "open-web", "o"], desc: "Open in Gmail" },
  { names: ["threads", "thr"], desc: "Toggle conversation view" },
  { names: ["expand-all", "expand", "flatten", "flat"], desc: "Expand all in thread" },
  { names: ["collapse-all", "collapse"], desc: "Collapse all in thread" },
  { names: ["thread-summary", "th-sum"], desc: "Summarize thread (AI)" },
  { names: ["inbox", "i"], desc: "Back to inbox" },
  { names: ["archived", "b", "arch-search"], desc: "Archived messages" },
  { names: ["markdown", "md"], desc: "Toggle HTML / text" },
  { names: ["images", "remote", "img"], desc: "Load / block remote images" },
  { names: ["images-always", "always-images", "imgall"], desc: "Always load remote images (on/off)" },
  { names: ["load", "more", "next"], desc: "Load more messages" },
  { names: ["attachments", "attach"], desc: "Focus attachments" },
  { names: ["accounts", "acc"], desc: "Switch account" },
  { names: ["label", "lbl"], desc: "Add label by name", arg: "<name>" },
  { names: ["select", "sel"], desc: "Bulk-select rows", arg: "all|none|<n|a-b>" },
  { names: ["goto", "g"], desc: "Go to row", arg: "[n]" },
  { names: ["bottom", "end", "$"], desc: "Go to last row" },
  { names: ["queries"], desc: "Saved searches" },
  { names: ["savequery", "save-query", "sq"], desc: "Save current search" },
  { names: ["plan", "actionplan", "action-plan", "ap"], desc: "AI inbox action plan", arg: "[rules|prompt]" },
  { names: ["rules", "ru"], desc: "Deterministic rules manager", arg: "[run|plan]" },
  { names: ["rp"], desc: "Run deterministic rules (:rules plan)" },
  { names: ["move", "mv"], desc: "Move to folder", arg: "[label]" },
  { names: ["draft", "replyai"], desc: "Draft reply (AI)" },
  { names: ["find"], desc: "Find in message", arg: "<text>" },
  { names: ["from"], desc: "Search from this sender" },
  { names: ["to"], desc: "Search to this recipient" },
  { names: ["subject"], desc: "Search this subject" },
  { names: ["headers", "toggle-headers"], desc: "Toggle headers" },
  { names: ["toolbar"], desc: "Show/hide reader toolbar" },
  { names: ["zoom-in", "zi"], desc: "Bigger UI text (Cmd/Ctrl +)" },
  { names: ["zoom-out", "zo"], desc: "Smaller UI text (Cmd/Ctrl -)" },
  { names: ["zoom-reset"], desc: "Reset UI zoom (Cmd/Ctrl 0)" },
  { names: ["zoom"], desc: "Set UI zoom", arg: "<0.6-2.4>" },
  { names: ["touch-up", "touchup"], desc: "Reformat message with AI" },
  { names: ["theme", "th"], desc: "Change theme", arg: "[name]" },
  { names: ["regenerate", "regen"], desc: "Regenerate the open AI panel (summary/prompt)" },
  { names: ["dismiss", "close-ai"], desc: "Close the open AI panel" },
  { names: ["quit", "q", "exit"], desc: "Quit GizTUI" },
  { names: ["help", "h"], desc: "Keyboard shortcuts" },
];

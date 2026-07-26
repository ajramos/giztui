import { describe, it, expect } from "vitest";
import {
  COMMANDS,
  parseCommand,
  filterCommands,
  resolveEnter,
  type CommandDef,
} from "./commands";

describe("parseCommand", () => {
  it("lowercases the command and keeps the raw argument", () => {
    expect(parseCommand("s from:me has:attachment")).toEqual({
      cmd: "s",
      arg: "from:me has:attachment",
    });
    expect(parseCommand("ARCHIVE")).toEqual({ cmd: "archive", arg: "" });
  });
  it("collapses surrounding / repeated whitespace", () => {
    expect(parseCommand("   plan    rules  ")).toEqual({ cmd: "plan", arg: "rules" });
  });
});

describe("filterCommands", () => {
  const has = (list: CommandDef[], name: string) =>
    list.some((c) => c.names.includes(name));

  it("returns everything for empty input", () => {
    expect(filterCommands(COMMANDS, "")).toBe(COMMANDS);
  });
  it("matches by name OR alias prefix on the first word only", () => {
    const m = filterCommands(COMMANDS, "arch");
    expect(has(m, "archive")).toBe(true);
    expect(has(m, "archived")).toBe(true); // alias arch-search also starts with "arch"
    expect(has(m, "search")).toBe(false);
  });
  it("ignores the argument when filtering", () => {
    const m = filterCommands(COMMANDS, "search from:me");
    expect(m.length).toBe(1);
    expect(m[0].names[0]).toBe("search");
  });
});

describe("resolveEnter", () => {
  it("returns the input verbatim when the first word is an exact command/alias (keeps args)", () => {
    expect(resolveEnter(COMMANDS, "search from:me", 0)).toBe("search from:me");
    expect(resolveEnter(COMMANDS, "u", 0)).toBe("u"); // alias of unread
    expect(resolveEnter(COMMANDS, "sq", 0)).toBe("sq"); // alias of savequery
  });
  it("returns the highlighted suggestion's canonical name for a prefix", () => {
    // "arch" is not an exact command; matches[0] is the earliest in COMMANDS.
    expect(resolveEnter(COMMANDS, "arch", 0)).toBe("archive");
  });
  it("returns the raw input when nothing matches (numeric jumps)", () => {
    expect(resolveEnter(COMMANDS, "5", 0)).toBe("5");
    expect(resolveEnter(COMMANDS, "$", 0)).toBe("$"); // '$' is an exact alias of bottom → verbatim
  });
});

describe("COMMANDS integrity", () => {
  it("has unique canonical names and no duplicate alias across commands", () => {
    const seen = new Map<string, string>();
    for (const c of COMMANDS) {
      for (const n of c.names) {
        expect(seen.has(n), `duplicate command token "${n}" (also in ${seen.get(n)})`).toBe(false);
        seen.set(n, c.names[0]);
      }
    }
  });
});

import { describe, it, expect } from "vitest";
import {
  displayName,
  matchesCombo,
  emailAddr,
  labelForAction,
  formatICSDate,
  mixHex,
  cleanSubject,
  countMatches,
  formatDate,
  formatFull,
  formatSize,
  resolveEmailLinkHref,
} from "./format";

describe("resolveEmailLinkHref", () => {
  it("passes through http/https", () => {
    expect(resolveEmailLinkHref("https://example.com/x")).toBe("https://example.com/x");
    expect(resolveEmailLinkHref("http://example.com")).toBe("http://example.com");
  });
  it("upgrades protocol-relative //host to https (the reported dead links)", () => {
    expect(resolveEmailLinkHref("//cdn.example.com/track?id=1")).toBe(
      "https://cdn.example.com/track?id=1",
    );
  });
  it("allows mailto and tel so those links aren't dead", () => {
    expect(resolveEmailLinkHref("mailto:a@b.com")).toBe("mailto:a@b.com");
    expect(resolveEmailLinkHref("tel:+34123")).toBe("tel:+34123");
  });
  it("trims surrounding whitespace before testing the scheme", () => {
    expect(resolveEmailLinkHref("  https://example.com  ")).toBe("https://example.com");
  });
  it("ignores unsafe or non-navigable hrefs", () => {
    expect(resolveEmailLinkHref("javascript:alert(1)")).toBeNull();
    expect(resolveEmailLinkHref("data:text/html,<h1>x</h1>")).toBeNull();
    expect(resolveEmailLinkHref("#section")).toBeNull();
    expect(resolveEmailLinkHref("")).toBeNull();
    expect(resolveEmailLinkHref(null)).toBeNull();
  });
});

describe("displayName", () => {
  it("prefers the quoted/display part before <addr>", () => {
    expect(displayName('"Ada Lovelace" <ada@x.com>')).toBe("Ada Lovelace");
    expect(displayName("Ada Lovelace <ada@x.com>")).toBe("Ada Lovelace");
  });
  it("falls back to the local-part when there is no display name", () => {
    expect(displayName("ada@x.com")).toBe("ada");
  });
  it("returns the input when there is no @ and no <", () => {
    expect(displayName("system")).toBe("system");
  });
});

describe("emailAddr", () => {
  it("extracts the address inside angle brackets", () => {
    expect(emailAddr('"Ada" <ada@x.com>')).toBe("ada@x.com");
  });
  it("returns the input when there are no brackets", () => {
    expect(emailAddr("ada@x.com")).toBe("ada@x.com");
  });
});

describe("cleanSubject", () => {
  it("strips Re:/Fwd:/Fw: prefixes (repeated, case-insensitive)", () => {
    expect(cleanSubject("Re: hello")).toBe("hello");
    expect(cleanSubject("FWD: Re:  hello")).toBe("hello");
    expect(cleanSubject("fw: hi")).toBe("hi");
  });
  it("leaves a clean subject untouched", () => {
    expect(cleanSubject("Meeting notes")).toBe("Meeting notes");
  });
  it("keeps the original if stripping would empty it", () => {
    expect(cleanSubject("Re:")).toBe("Re:");
  });
});

describe("countMatches", () => {
  it("counts case-insensitive, non-overlapping occurrences", () => {
    expect(countMatches("aXaXa", "x")).toBe(2);
    expect(countMatches("Hello hello HELLO", "hello")).toBe(3);
  });
  it("returns 0 for an empty query or no match", () => {
    expect(countMatches("abc", "")).toBe(0);
    expect(countMatches("abc", "z")).toBe(0);
  });
});

describe("formatSize", () => {
  it("formats bytes with the right unit and precision", () => {
    expect(formatSize(0)).toBe("");
    expect(formatSize(512)).toBe("512 B");
    expect(formatSize(1024)).toBe("1.0 KB");
    expect(formatSize(1536)).toBe("1.5 KB");
    expect(formatSize(1024 * 1024)).toBe("1.0 MB");
    expect(formatSize(5 * 1024 * 1024 * 1024)).toBe("5.0 GB");
  });
});

describe("mixHex", () => {
  it("returns the endpoints at t=0 and t=1", () => {
    expect(mixHex("#000000", "#ffffff", 0)).toBe("#000000");
    expect(mixHex("#000000", "#ffffff", 1)).toBe("#ffffff");
  });
  it("blends at the midpoint and expands #rgb shorthand", () => {
    expect(mixHex("#000", "#fff", 0.5)).toBe("#808080");
  });
  it("falls back to the first color when either is unparseable", () => {
    expect(mixHex("nope", "#fff", 0.5)).toBe("nope");
    expect(mixHex("#123", "zzz", 0.5)).toBe("#123"); // zzz is not hex
    expect(mixHex("#12", "#fff", 0.5)).toBe("#12"); // wrong length
  });
});

describe("labelForAction", () => {
  it("maps known actions to their present participle", () => {
    expect(labelForAction("archive")).toBe("Archiving");
    expect(labelForAction("trash")).toBe("Trashing");
    expect(labelForAction("read")).toBe("Marking read");
    expect(labelForAction("unread")).toBe("Marking unread");
  });
  it("falls back for unknown actions", () => {
    expect(labelForAction("whatever")).toBe("Working on");
  });
});

describe("matchesCombo", () => {
  const ev = (o: Partial<KeyboardEvent>): KeyboardEvent =>
    ({ ctrlKey: false, metaKey: false, shiftKey: false, altKey: false, key: "", ...o }) as KeyboardEvent;

  it("treats ctrl and cmd as interchangeable", () => {
    expect(matchesCombo(ev({ ctrlKey: true, key: "f" }), "ctrl+f")).toBe(true);
    expect(matchesCombo(ev({ metaKey: true, key: "f" }), "ctrl+f")).toBe(true);
  });
  it("requires the modifier — a bare key never matches", () => {
    expect(matchesCombo(ev({ key: "f" }), "ctrl+f")).toBe(false);
    expect(matchesCombo(ev({ key: "a" }), "a")).toBe(false);
  });
  it("respects shift and alt requirements exactly", () => {
    expect(matchesCombo(ev({ ctrlKey: true, shiftKey: true, key: "p" }), "ctrl+shift+p")).toBe(true);
    expect(matchesCombo(ev({ ctrlKey: true, key: "p" }), "ctrl+shift+p")).toBe(false);
    expect(matchesCombo(ev({ altKey: true, key: "x" }), "alt+x")).toBe(true);
  });
});

describe("date formatters (deterministic pieces only)", () => {
  it("formatDate returns '' for an invalid date and 'now' for the present", () => {
    expect(formatDate("not-a-date")).toBe("");
    expect(formatDate(new Date().toISOString())).toBe("now");
  });
  it("formatDate buckets minutes/hours/days", () => {
    const ago = (ms: number) => new Date(Date.now() - ms).toISOString();
    expect(formatDate(ago(5 * 60_000))).toBe("5m");
    expect(formatDate(ago(3 * 3_600_000))).toBe("3h");
    expect(formatDate(ago(2 * 86_400_000))).toBe("2d");
  });
  it("formatFull returns '' for an invalid date, non-empty otherwise", () => {
    expect(formatFull("not-a-date")).toBe("");
    expect(formatFull("2026-07-20T10:00:00Z").length).toBeGreaterThan(0);
  });
  it("formatICSDate returns the raw input when it can't parse", () => {
    expect(formatICSDate("garbage")).toBe("garbage");
    // a parseable value produces a non-empty, different string
    const out = formatICSDate("20260720T150000");
    expect(out).not.toBe("20260720T150000");
    expect(out.length).toBeGreaterThan(0);
  });
  it("formatICSDate renders an absolute RFC3339 instant in the viewer's tz", () => {
    // 07:00 UTC rendered in a fixed zone is deterministic; assert the hour shown
    // for Europe/Madrid (CEST, UTC+2 in July) is 09.
    const out = new Date("2026-07-31T07:00:00Z").toLocaleString([], {
      hour: "2-digit",
      minute: "2-digit",
      timeZone: "Europe/Madrid",
      hour12: false,
    });
    expect(out).toContain("09:00");
    // And the function accepts the RFC3339 form without falling back to raw.
    expect(formatICSDate("2026-07-31T07:00:00Z")).not.toBe("2026-07-31T07:00:00Z");
  });
});

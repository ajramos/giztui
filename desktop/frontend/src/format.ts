// Pure formatting / parsing helpers shared across the app. Extracted from
// App.tsx so they can be unit-tested in isolation (no React, no state).

export function displayName(from: string): string {
  const m = from.match(/^\s*"?([^"<]+?)"?\s*</);
  if (m) return m[1].trim();
  return from.split("@")[0] || from;
}

// matchesCombo reports whether a keyboard event matches a TUI-style modifier
// combo like "ctrl+f" or "ctrl+shift+p". Ctrl and Cmd are treated as
// interchangeable so a config's "ctrl+f" also fires on macOS's Cmd+F. Only
// combos that include a modifier are matched — a bare key returns false so it
// never hijacks normal typing/actions.
export function matchesCombo(e: KeyboardEvent, combo: string): boolean {
  const parts = combo.toLowerCase().split("+");
  const key = parts[parts.length - 1];
  const wantCtrlOrMeta = parts.includes("ctrl") || parts.includes("cmd") || parts.includes("meta");
  const wantShift = parts.includes("shift");
  const wantAlt = parts.includes("alt") || parts.includes("option");
  if (!wantCtrlOrMeta && !wantAlt) return false; // require a modifier
  if (wantCtrlOrMeta !== (e.ctrlKey || e.metaKey)) return false;
  if (wantAlt !== e.altKey) return false;
  if (wantShift !== e.shiftKey) return false;
  return e.key.toLowerCase() === key;
}

export function emailAddr(from: string): string {
  const m = from.match(/<([^>]+)>/);
  return m ? m[1] : from;
}

// labelForAction is the present-participle verb shown while a bulk action runs.
export function labelForAction(action: string): string {
  switch (action) {
    case "archive":
      return "Archiving";
    case "trash":
      return "Trashing";
    case "read":
      return "Marking read";
    case "unread":
      return "Marking unread";
    default:
      return "Working on";
  }
}

// formatICSDate renders a calendar-invite date-time as a human-readable local
// string. The backend resolves zoned/UTC invites to an absolute RFC3339 instant
// (e.g. "2026-07-31T07:00:00Z") — those are converted to the VIEWER's timezone,
// so a 09:00-CEST event shows 09:00 for a CEST user regardless of the
// organizer's zone. Raw iCal digits ("20260720T150000", all-day "20260731") have
// no zone, so they fall back to a wall-clock reading.
export function formatICSDate(raw: string): string {
  if (!raw) return raw;
  // Absolute instant from the backend (ISO-8601 has dashes in the date): render
  // in local time.
  if (raw.includes("-")) {
    const iso = new Date(raw);
    if (!Number.isNaN(iso.getTime())) {
      return iso.toLocaleString([], {
        weekday: "short",
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      });
    }
  }
  const v = raw.includes(":") ? raw.slice(raw.lastIndexOf(":") + 1) : raw;
  const m = v.match(/^(\d{4})(\d{2})(\d{2})(?:T(\d{2})(\d{2})(\d{2})?Z?)?/);
  if (!m) return raw;
  const [, y, mo, d, hh = "00", mm = "00"] = m;
  const dt = new Date(
    Number(y),
    Number(mo) - 1,
    Number(d),
    Number(hh),
    Number(mm),
  );
  if (Number.isNaN(dt.getTime())) return raw;
  return dt.toLocaleString([], {
    weekday: "short",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

// mixHex blends hex color `a` toward `b` by t∈[0,1]. Falls back to `a` when
// either isn't a parseable #rgb / #rrggbb string, so theme mapping never breaks.
export function mixHex(a: string, b: string, t: number): string {
  const parse = (h: string): [number, number, number] | null => {
    let s = h.trim().replace(/^#/, "");
    if (s.length === 3) s = s.replace(/(.)/g, "$1$1");
    if (s.length !== 6 || /[^0-9a-fA-F]/.test(s)) return null;
    return [
      parseInt(s.slice(0, 2), 16),
      parseInt(s.slice(2, 4), 16),
      parseInt(s.slice(4, 6), 16),
    ];
  };
  const ca = parse(a);
  const cb = parse(b);
  if (!ca || !cb) return a;
  const mix = (x: number, y: number) =>
    Math.round(x + (y - x) * t)
      .toString(16)
      .padStart(2, "0");
  return `#${mix(ca[0], cb[0])}${mix(ca[1], cb[1])}${mix(ca[2], cb[2])}`;
}

// cleanSubject strips Re:/Fwd: prefixes so a subject search matches the thread.
export function cleanSubject(subject: string): string {
  return subject.replace(/^(\s*(re|fwd|fw)\s*:\s*)+/i, "").trim() || subject;
}

// countMatches returns how many times query occurs in text (case-insensitive).
export function countMatches(text: string, query: string): number {
  if (!query) return 0;
  const q = query.toLowerCase();
  const t = text.toLowerCase();
  let n = 0;
  let i = t.indexOf(q);
  while (i !== -1) {
    n++;
    i = t.indexOf(q, i + q.length);
  }
  return n;
}

export function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const mins = Math.floor((Date.now() - d.getTime()) / 60000);
  if (mins < 1) return "now";
  if (mins < 60) return `${mins}m`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d`;
  return d.toLocaleDateString([], { month: "short", day: "numeric" });
}

export function formatFull(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString();
}

export function formatSize(bytes: number): string {
  if (bytes <= 0) return "";
  const units = ["B", "KB", "MB", "GB"];
  let n = bytes;
  let u = 0;
  while (n >= 1024 && u < units.length - 1) {
    n /= 1024;
    u++;
  }
  return `${n.toFixed(u === 0 ? 0 : 1)} ${units[u]}`;
}

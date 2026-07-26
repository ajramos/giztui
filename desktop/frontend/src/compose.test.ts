import { describe, it, expect } from "vitest";
import { replyInit, replyAllInit, forwardInit } from "./compose";
import type { MessageDetail } from "./api";

// Minimal MessageDetail factory — only the fields the builders read matter.
const msg = (o: Partial<MessageDetail>): MessageDetail =>
  ({
    id: "m1",
    from: "Ada <ada@x.com>",
    to: "me@x.com",
    cc: "",
    subject: "Hello",
    date: "2026-07-20T10:00:00Z",
    plainText: "body text",
    html: "",
    unread: false,
    ...o,
  }) as MessageDetail;

describe("replyInit", () => {
  it("replies to the sender, threaded on the original", () => {
    expect(replyInit(msg({ id: "abc", from: "Ada <ada@x.com>" }))).toEqual({
      mode: "reply",
      originalId: "abc",
      to: "Ada <ada@x.com>",
    });
  });
});

describe("replyAllInit", () => {
  it("adds the original To/Cc as Cc, de-duplicated", () => {
    const r = replyAllInit(
      msg({ from: "ada@x.com", to: "bob@x.com, carol@x.com", cc: "carol@x.com, dave@x.com" }),
    );
    expect(r.mode).toBe("reply");
    expect(r.to).toBe("ada@x.com");
    const cc = (r.cc || "").split(", ").sort();
    expect(cc).toEqual(["bob@x.com", "carol@x.com", "dave@x.com"]);
  });
  it("never re-adds the sender to Cc", () => {
    const r = replyAllInit(msg({ from: "ada@x.com", to: "ada@x.com, bob@x.com", cc: "" }));
    expect(r.cc).toBe("bob@x.com");
  });
  it("yields an empty Cc when there are no other recipients", () => {
    const r = replyAllInit(msg({ from: "ada@x.com", to: "", cc: "" }));
    expect(r.cc).toBe("");
  });
});

describe("forwardInit", () => {
  it("prefixes Fwd: once and quotes the original", () => {
    const r = forwardInit(msg({ subject: "Report", from: "Ada <ada@x.com>", plainText: "hi" }));
    expect(r.mode).toBe("new");
    expect(r.subject).toBe("Fwd: Report");
    expect(r.body).toContain("---------- Forwarded message ----------");
    expect(r.body).toContain("From: Ada <ada@x.com>");
    expect(r.body).toContain("hi");
  });
  it("does not double-prefix an already-forwarded subject", () => {
    expect(forwardInit(msg({ subject: "Fwd: Report" })).subject).toBe("Fwd: Report");
  });
});

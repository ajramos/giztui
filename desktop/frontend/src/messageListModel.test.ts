import { describe, it, expect } from "vitest";
import { freshPrefix, dedupeNew } from "./messageListModel";

const m = (id: string) => ({ id });

describe("freshPrefix", () => {
  it("returns all messages when none are known", () => {
    expect(freshPrefix([m("a"), m("b")], new Set())).toEqual([m("a"), m("b")]);
  });
  it("returns nothing when the first message is already known", () => {
    expect(freshPrefix([m("a"), m("b")], new Set(["a"]))).toEqual([]);
  });
  it("returns only the contiguous unknown prefix, stopping at the first known id", () => {
    expect(freshPrefix([m("n1"), m("n2"), m("k1")], new Set(["k1"]))).toEqual([
      m("n1"),
      m("n2"),
    ]);
  });
  it("does NOT treat a later unknown id past a known one as new (the scramble guard)", () => {
    // 'old' shifted onto page 1 after a delete: unknown to us, but it appears
    // AFTER a known message, so it must not be prepended as new mail.
    const page = [m("new1"), m("known"), m("old")];
    expect(freshPrefix(page, new Set(["known"]))).toEqual([m("new1")]);
  });
  it("handles an empty page", () => {
    expect(freshPrefix([], new Set(["a"]))).toEqual([]);
  });
});

describe("dedupeNew", () => {
  it("drops items already known", () => {
    expect(dedupeNew([m("a"), m("b"), m("c")], new Set(["b"]))).toEqual([m("a"), m("c")]);
  });
  it("keeps everything when nothing is known", () => {
    expect(dedupeNew([m("a")], new Set())).toEqual([m("a")]);
  });
  it("returns empty when all are known", () => {
    expect(dedupeNew([m("a"), m("b")], new Set(["a", "b"]))).toEqual([]);
  });
});

import { describe, it, expect } from "vitest";
import { activeAiPanel } from "./aiPanels";

const s = (o: Partial<{ hasSummary: boolean; hasPrompt: boolean; hasTouchUp: boolean }>) => ({
  hasSummary: false,
  hasPrompt: false,
  hasTouchUp: false,
  ...o,
});

describe("activeAiPanel", () => {
  it("regenerates the prompt when a prompt result is up and no summary", () => {
    expect(activeAiPanel(s({ hasPrompt: true }))).toBe("prompt");
  });
  it("prefers the summary when both a summary and a prompt are shown", () => {
    expect(activeAiPanel(s({ hasSummary: true, hasPrompt: true }))).toBe("summary");
  });
  it("re-reformats when only a touch-up is shown", () => {
    expect(activeAiPanel(s({ hasTouchUp: true }))).toBe("touchup");
  });
  it("does NOT pick touch-up if a prompt or summary is also up", () => {
    expect(activeAiPanel(s({ hasTouchUp: true, hasPrompt: true }))).toBe("prompt");
    expect(activeAiPanel(s({ hasTouchUp: true, hasSummary: true }))).toBe("summary");
  });
  it("defaults to summary when a summary is shown or nothing is", () => {
    expect(activeAiPanel(s({ hasSummary: true }))).toBe("summary");
    expect(activeAiPanel(s({}))).toBe("summary");
  });
});

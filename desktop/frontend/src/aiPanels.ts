// Pure decision logic for the AI panels, extracted from App.tsx. The stateful
// streaming (summarize/runPrompt/touchUp) and their refs stay in App by design:
// they're coupled to the shared openIdRef and to loadMessage's reset ordering
// (see docs/DESKTOP_REFACTOR_PLAN.md §3). What IS cleanly separable is the
// branching that decides which panel is "active" — the bit that's easy to get
// subtly wrong — so it lives here with tests.

export type AiPanelKind = "summary" | "prompt" | "touchup";

// activeAiPanel picks which panel a "regenerate" should re-run, given which
// panels currently hold content. Mirrors App's original order exactly:
//   - a prompt result (and no summary) → regenerate the prompt
//   - else a touch-up (and nothing else) → re-reformat
//   - else (a summary is shown, or nothing yet) → (re)generate the summary
export function activeAiPanel(s: {
  hasSummary: boolean;
  hasPrompt: boolean;
  hasTouchUp: boolean;
}): AiPanelKind {
  if (s.hasPrompt && !s.hasSummary) return "prompt";
  if (s.hasTouchUp && !s.hasSummary && !s.hasPrompt) return "touchup";
  return "summary";
}

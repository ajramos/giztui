import { test, expect } from "@playwright/test";
import {
  openApp,
  openMessageAt,
  promptPanel,
  runCommand,
  summaryPanel,
  waitForPanelDone,
} from "./helpers";

// These flows are the ones the refactor plan (§3 landmines) calls out as the
// source of the coupling bugs: an AI run must belong to the message it was
// launched on (summaryForId / promptForId / openIdRef), the per-message result
// is cached and restored on return, and a run started on one message must never
// paint into another. This is the primary net for the useAiPanels extraction.

test.describe("AI summary panel", () => {
  test("':summarize' streams a summary into the reader", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 0);

    await runCommand(page, "summarize");
    await expect(summaryPanel(page)).toBeVisible();
    await expect(summaryPanel(page).locator(".summary-head")).toContainText("AI summary");
    const text = await waitForPanelDone(summaryPanel(page));
    expect(text.length).toBeGreaterThan(0);
  });

  test("summary is per-message: cached on return, absent on a fresh message", async ({
    page,
  }) => {
    await openApp(page);

    // Summarize message A and let it finish streaming.
    await openMessageAt(page, 0);
    await runCommand(page, "summarize");
    const summaryA = await waitForPanelDone(summaryPanel(page));
    expect(summaryA.length).toBeGreaterThan(0);

    // Switch to message B — its summary was never generated, so no panel bleeds
    // over from A.
    await openMessageAt(page, 1);
    await expect(summaryPanel(page)).toBeHidden();

    // Return to A — the cached summary is restored, unchanged.
    await openMessageAt(page, 0);
    await expect(summaryPanel(page)).toBeVisible();
    await expect(summaryPanel(page).locator(".summary-text")).toHaveText(summaryA);
  });

  test("'dismiss' hides the open summary panel", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 0);
    await runCommand(page, "summarize");
    // Wait for the run to complete before dismissing (dismiss is a post-run action).
    await waitForPanelDone(summaryPanel(page));

    await runCommand(page, "dismiss");
    await expect(summaryPanel(page)).toBeHidden();
  });
});

test.describe("AI prompt panel", () => {
  test("applying a prompt shows a result that survives switch + return", async ({
    page,
  }) => {
    await openApp(page);
    await openMessageAt(page, 0);

    // ':prompt' opens the prompts picker; Enter runs the highlighted prompt.
    await runCommand(page, "prompt");
    await expect(page.locator(".modal-overlay")).toBeVisible();
    await page.keyboard.press("Enter");

    await expect(promptPanel(page)).toBeVisible();
    const promptA = await waitForPanelDone(promptPanel(page));
    expect(promptA.length).toBeGreaterThan(0);

    // Switch away — the prompt result must not paint onto message B.
    await openMessageAt(page, 1);
    await expect(promptPanel(page)).toBeHidden();

    // Return to A — the cached prompt result is restored.
    await openMessageAt(page, 0);
    await expect(promptPanel(page)).toBeVisible();
    await expect(promptPanel(page).locator(".summary-text")).toHaveText(promptA);
  });
});

test.describe("AI chat", () => {
  test("':chat' opens the chat panel; a message streams an assistant reply", async ({
    page,
  }) => {
    await openApp(page);
    await openMessageAt(page, 0);
    // Open via the default keymap shortcut ("X") — exercises the keymap wiring.
    await page.keyboard.press("X");

    const input = page.locator(".chat-input");
    await expect(input).toBeVisible();
    await input.fill("What are the action items?");
    await input.press("Enter");

    // The user turn shows immediately; the assistant reply streams in.
    await expect(page.locator(".chat-user").last()).toContainText(
      "What are the action items?",
    );
    await expect(page.locator(".chat-assistant").last()).toContainText(
      "mock answer",
    );
  });
});

test.describe("AI bulk prompt (background job)", () => {
  test("running a bulk prompt shows its result in the job dialog; Escape closes it", async ({
    page,
  }) => {
    await openApp(page);
    // ':select all' enters bulk mode with every row selected; ':prompt' then opens
    // the prompts picker (allowed in bulk mode), and Enter runs the highlighted one.
    await runCommand(page, "select all");
    await runCommand(page, "prompt");
    await expect(page.locator(".modal-overlay")).toBeVisible();
    await page.keyboard.press("Enter");

    // The bulk prompt now runs as a background job whose result surfaces in the
    // job dialog (the ✦-headed modal), not a per-message panel.
    const jobModal = page
      .locator(".modal-overlay")
      .filter({ has: page.locator("h3", { hasText: "✦" }) });
    await expect(jobModal).toBeVisible();
    await expect(jobModal.locator(".summary-text")).toContainText(
      "mock bulk prompt result",
    );

    // Closing the dialog dismisses the view (the job itself is already done here).
    await page.keyboard.press("Escape");
    await expect(jobModal).toBeHidden();
  });

  test("':jobs' lists the job and Enter re-opens its result", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "select all");
    await runCommand(page, "prompt");
    await expect(page.locator(".modal-overlay")).toBeVisible();
    await page.keyboard.press("Enter");
    const jobModal = page
      .locator(".modal-overlay")
      .filter({ has: page.locator("h3", { hasText: "✦" }) });
    await expect(jobModal.locator(".summary-text")).toContainText(
      "mock bulk prompt result",
    );
    // Close the result dialog, then browse jobs via ':jobs'.
    await page.keyboard.press("Escape");
    await expect(jobModal).toBeHidden();

    // Open via the default keymap shortcut ("J") — exercises the keymap wiring.
    await page.keyboard.press("J");
    const jobsPicker = page
      .locator(".modal-overlay")
      .filter({ has: page.locator("h3", { hasText: "AI jobs" }) });
    await expect(jobsPicker).toBeVisible();
    await expect(jobsPicker.locator(".prompt-row").first()).toContainText(
      "messages",
    );
    await expect(jobsPicker.locator('.job-status[data-status="done"]').first()).toBeVisible();

    // Enter re-opens the selected job's result in the ✦ dialog.
    await page.keyboard.press("Enter");
    await expect(jobsPicker).toBeHidden();
    await expect(jobModal.locator(".summary-text")).toContainText(
      "mock bulk prompt result",
    );
  });
});

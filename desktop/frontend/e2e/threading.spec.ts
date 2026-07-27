import { test, expect } from "@playwright/test";
import { openApp, openMessageAt, runCommand, summaryPanel } from "./helpers";

// Guards the useThreading extraction (F3.2): ':threads' toggles the conversation
// view, and ':thread-summary' is gated on the loaded thread (threadMsgs, now
// owned by the hook) before calling summarizeThread (which still lives in App).
// The mock returns a 3-message thread.
test.describe("threading", () => {
  test("':threads' toggles the conversation view", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 2); // "Re: Project roadmap Q3"

    await runCommand(page, "threads");
    await expect(page.locator(".reader-body")).toContainText(
      "Conversation · 3 messages",
    );

    await runCommand(page, "threads");
    await expect(page.locator(".reader-body")).not.toContainText(
      "Conversation · 3 messages",
    );
  });

  test("':thread-summary' summarizes the open thread", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 2);
    await runCommand(page, "threads");
    await expect(page.locator(".reader-body")).toContainText(
      "Conversation · 3 messages",
    );

    await runCommand(page, "thread-summary");
    await expect(summaryPanel(page)).toBeVisible();
    await expect(summaryPanel(page).locator(".summary-text")).toContainText(
      "roadmap",
    );
  });
});

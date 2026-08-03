import { test, expect } from "@playwright/test";
import { openApp, runCommand } from "./helpers";

// Display modals extracted from App.tsx (StatsModal / ConfigModal) — verify they
// still open via their command and close on Escape after the F4 extraction.
test.describe("display modals", () => {
  test("':prompt stats' opens the AI usage modal and Escape closes it", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "prompt stats");
    const modal = page.locator(".modal-overlay").filter({ hasText: "AI usage" });
    await expect(modal).toBeVisible();
    await expect(modal.locator(".stat-tile").first()).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(modal).toBeHidden();
  });

  test("':stats' opens the local usage-analytics dashboard and Escape closes it", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "stats");
    const modal = page.locator(".modal-overlay").filter({ hasText: "Usage analytics" });
    await expect(modal).toBeVisible();
    // The Actions section (outcomes + timing) is the telemetry-specific part.
    await expect(modal).toContainText("Actions (outcome");
    await page.keyboard.press("Escape");
    await expect(modal).toBeHidden();
  });

  test("':config' opens the configuration modal and Escape closes it", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "config");
    const modal = page.locator(".modal-overlay").filter({ hasText: "Configuration" });
    await expect(modal).toBeVisible();
    await expect(modal.locator(".config-row").first()).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(modal).toBeHidden();
  });

  test("':config migrate' runs the migration and toasts (no config modal)", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "config migrate");
    // Runs the backend migration and confirms with a toast — it must NOT open the
    // read-only Configuration modal (the parity bug this guards against).
    await expect(page.locator(".toast")).toContainText("Config");
    await expect(
      page.locator(".modal-overlay").filter({ hasText: "Configuration" }),
    ).toBeHidden();
  });

  test("':action-plan prompt' opens the analyzer-prompt preview; Escape closes it", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "action-plan prompt");
    const modal = page.locator(".modal-overlay").filter({ hasText: "Analyzer prompt" });
    await expect(modal).toBeVisible();
    await expect(modal.locator("pre.summary-text")).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(modal).toBeHidden();
  });

  test("':action-plan rules' opens the analyzer-rules modal and Escape closes it", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "action-plan rules");
    const modal = page.locator(".modal-overlay").filter({ hasText: "Analyzer rules" });
    await expect(modal).toBeVisible();
    await expect(modal.locator(".prompt-manage-row")).toHaveCount(2);
    await page.keyboard.press("ArrowDown"); // window-level list nav still works

    // Bare "d" deletes the highlighted rule via the shared usePickerCrud handler
    // (the list holds focus, so no Shift is needed here).
    await page.keyboard.press("d");
    await expect(modal.locator(".prompt-manage-row")).toHaveCount(1);

    await page.keyboard.press("Escape");
    await expect(modal).toBeHidden();
  });

  test("':savequery' opens the save-search dialog once a search is active", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "search from:test");
    await runCommand(page, "savequery");
    const modal = page.locator(".modal-overlay").filter({ hasText: "Save search" });
    await expect(modal).toBeVisible();
    await expect(modal.locator("input").first()).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(modal).toBeHidden();
  });

  test("':advanced' opens the builder and previews the query as you type", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "advanced");
    const modal = page.locator(".modal-overlay").filter({ hasText: "Advanced search" });
    await expect(modal).toBeVisible();
    // Typing a From value updates the live query preview.
    await modal.locator(".field", { hasText: "From" }).locator("input").fill("boss@x.com");
    await expect(modal.locator(".ro-value")).toHaveText("from:boss@x.com");
    await page.keyboard.press("Escape");
    await expect(modal).toBeHidden();
  });
});

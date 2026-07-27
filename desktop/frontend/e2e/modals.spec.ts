import { test, expect } from "@playwright/test";
import { openApp, runCommand } from "./helpers";

// Display modals extracted from App.tsx (StatsModal / ConfigModal) — verify they
// still open via their command and close on Escape after the F4 extraction.
test.describe("display modals", () => {
  test("':stats' opens the AI usage modal and Escape closes it", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "stats");
    const modal = page.locator(".modal-overlay").filter({ hasText: "AI usage" });
    await expect(modal).toBeVisible();
    await expect(modal.locator(".stat-tile").first()).toBeVisible();
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
    await expect(modal.locator(".prompt-manage-row").first()).toBeVisible();
    await page.keyboard.press("ArrowDown"); // window-level list nav still works
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
});

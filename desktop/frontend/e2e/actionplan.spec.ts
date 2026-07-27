import { test, expect } from "@playwright/test";
import { openApp, runCommand } from "./helpers";

// The inbox action plan (:plan / :action-plan). Establishes the safety net for
// extracting the (prop-heavy) plan modal: open → analyze → categories render →
// keyboard nav highlights a row → Escape closes.
test.describe("action plan", () => {
  test(":plan analyzes the inbox and lists actionable categories", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "plan");

    const modal = page.locator(".modal-overlay").filter({ hasText: "Inbox action plan" });
    await expect(modal).toBeVisible();

    // The mock analyzes in ~1.4s, then renders category rows.
    const cats = modal.locator(".plan-cat");
    await expect(cats.first()).toBeVisible({ timeout: 8000 });
    expect(await cats.count()).toBeGreaterThan(1);

    // Window-level list nav highlights a category row.
    await page.keyboard.press("ArrowDown");
    await expect(modal.locator(".plan-cat.nav-active")).toBeVisible();

    await page.keyboard.press("Escape");
    await expect(modal).toBeHidden();
  });

  test("expanding a category reveals its emails", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "plan");
    const modal = page.locator(".modal-overlay").filter({ hasText: "Inbox action plan" });
    const firstCat = modal.locator(".plan-cat").first();
    await expect(firstCat).toBeVisible({ timeout: 8000 });
    // Click the category header to expand it; the caret flips to ▾.
    await firstCat.locator(".plan-cat-main").click();
    await expect(firstCat.locator(".conv-caret")).toHaveText("▾");
    await page.keyboard.press("Escape");
    await expect(modal).toBeHidden();
  });
});

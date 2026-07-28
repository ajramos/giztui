import { test, expect } from "@playwright/test";
import { openApp, runCommand } from "./helpers";

// Drafts subsystem (useDrafts). Guards the F5.1 hook extraction: :drafts shows
// the drafts pane, opening a draft loads it into the composer, and "Back to
// inbox" restores the list.
test.describe("drafts", () => {
  test("':drafts' lists drafts; opening one loads the composer", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "drafts");

    // The left pane flips to drafts: a "Drafts" header + the mock's draft rows.
    await expect(page.locator(".bulk-count", { hasText: "Drafts" })).toBeVisible();
    const draftRows = page.locator(".list .row");
    await expect(draftRows.first()).toBeVisible();
    expect(await draftRows.count()).toBeGreaterThan(1);

    // Clicking a draft opens the composer in "Edit draft" mode.
    await draftRows.first().click();
    const composer = page.locator(".modal-overlay").filter({ hasText: "Edit draft" });
    await expect(composer).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(composer).toBeHidden();

    // "Back to inbox" leaves the drafts view.
    await page.locator("button", { hasText: "Back to inbox" }).click();
    await expect(page.locator(".bulk-count", { hasText: "Drafts" })).toBeHidden();
  });
});

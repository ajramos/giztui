import { test, expect } from "@playwright/test";
import { openApp, openMessageAt, rows, runCommand } from "./helpers";

// Star / unstar (issue #34). '*' toggles the STARRED flag on the highlighted
// message and the row shows a .star-flag indicator, reflected in place (no
// refetch). ':star' / ':unstar' commands do the same on the open message.
// Mock message 0 starts unstarred (see apiMockData: starred = i % 4 === 1).
test.describe("star", () => {
  test("'*' toggles the star indicator on the open message", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 0);
    const row0 = rows(page).nth(0);
    await expect(row0.locator(".star-flag")).toHaveCount(0);
    await page.keyboard.press("*");
    await expect(row0.locator(".star-flag")).toBeVisible();
    await page.keyboard.press("*");
    await expect(row0.locator(".star-flag")).toHaveCount(0);
  });

  test("':star' and ':unstar' flip the open message", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 0);
    const row0 = rows(page).nth(0);
    await runCommand(page, "star");
    await expect(row0.locator(".star-flag")).toBeVisible();
    await runCommand(page, "unstar");
    await expect(row0.locator(".star-flag")).toHaveCount(0);
  });
});

import { test, expect } from "@playwright/test";
import { openApp, openMessageAt, rows, runCommand } from "./helpers";

// Message mutations (useMailActions). Guards the extraction of the archive/trash/
// bulk cluster: a single archive removes the open row and advances the cursor;
// bulk select-all + archive clears the whole list.
test.describe("mail actions", () => {
  test("archiving the open message removes its row from the list", async ({ page }) => {
    await openApp(page);
    const before = await rows(page).count();
    await openMessageAt(page, 0);
    // 'a' archives the open message (keymap.archive); the row leaves the list.
    await page.keyboard.press("a");
    await expect(rows(page)).toHaveCount(before - 1);
  });

  test("bulk select-all then archive empties the inbox list", async ({ page }) => {
    await openApp(page);
    expect(await rows(page).count()).toBeGreaterThan(0);
    // ':select all' enters bulk mode with every row selected; the 'a' key then
    // archives the whole selection (bulkAction), clearing the list.
    await runCommand(page, "select all");
    await page.keyboard.press("a");
    await expect(page.locator(".placeholder", { hasText: "No messages" })).toBeVisible();
  });
});

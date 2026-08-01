import { test, expect } from "@playwright/test";
import { openApp, rows, runCommand } from "./helpers";

// Message-number column (TUI :numbers parity). ':numbers' (alias ':n') toggles a
// 1-based, right-aligned number column on each row so ':N' jumps like ':14' are
// easy to aim. The mock's ShowMessageNumbers() returns false, so the column
// starts hidden.
test.describe("numbers", () => {
  test("':numbers' toggles the message-number column", async ({ page }) => {
    await openApp(page);
    // Hidden by default (config-seeded off).
    await expect(rows(page).nth(0).locator(".row-num")).toHaveCount(0);

    // Toggle on: every row gets a number, the first row is "1".
    await runCommand(page, "numbers");
    await expect(rows(page).nth(0).locator(".row-num")).toHaveText("1");
    await expect(rows(page).nth(1).locator(".row-num")).toHaveText("2");

    // Toggle off again.
    await runCommand(page, "numbers");
    await expect(rows(page).nth(0).locator(".row-num")).toHaveCount(0);
  });

  test("':n' is an alias for ':numbers'", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "n");
    await expect(rows(page).nth(0).locator(".row-num")).toHaveText("1");
  });
});

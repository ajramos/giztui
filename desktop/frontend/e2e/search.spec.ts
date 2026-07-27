import { test, expect } from "@playwright/test";
import { openApp, rows, runCommand } from "./helpers";

test.describe("search scoping", () => {
  test("':search' scopes the list and ':inbox' restores it", async ({ page }) => {
    await openApp(page);
    await expect(rows(page)).toHaveCount(24);

    // The mock matches subject/from substrings; 4 of the 24 are invoices.
    await runCommand(page, "search invoice");
    await expect(rows(page)).toHaveCount(4);
    await expect(rows(page).first().locator(".subject")).toHaveText(
      "Invoice #2043 from Acme Corp",
    );

    await runCommand(page, "inbox");
    await expect(rows(page)).toHaveCount(24);
  });

  test("a search with no matches yields an empty list, and inbox recovers", async ({
    page,
  }) => {
    await openApp(page);
    await runCommand(page, "search zzz-no-such-subject");
    await expect(rows(page)).toHaveCount(0);

    await runCommand(page, "inbox");
    await expect(rows(page)).toHaveCount(24);
  });
});

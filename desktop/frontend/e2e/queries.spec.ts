import { test, expect } from "@playwright/test";
import { openApp, runCommand } from "./helpers";

test.describe("saved queries picker", () => {
  test("':queries' groups by category and @cat narrows", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "queries");

    const picker = page
      .locator(".modal-overlay")
      .filter({ has: page.locator("h3", { hasText: "Saved searches" }) });
    await expect(picker).toBeVisible();

    // Named groups sort alphabetically; the uncategorised query falls under
    // "Default", which sorts last.
    await expect(picker.locator(".query-group-head")).toHaveText([
      "Finance",
      "Work",
      "Default",
    ]);

    // "@work" narrows to just that category group.
    await picker.locator(".label-filter").fill("@work");
    await expect(picker.locator(".query-group-head")).toHaveText(["Work"]);
    await expect(picker.locator(".query-row")).toHaveCount(1);
    await expect(picker.locator(".query-row")).toContainText("Unread from team");
  });
});

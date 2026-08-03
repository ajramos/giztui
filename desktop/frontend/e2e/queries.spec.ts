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

    // Shift+Backspace deletes the highlighted row (a bare key would type into the
    // filter). The only Work query goes, leaving no matches for "@work".
    await page.keyboard.press("Shift+Backspace");
    await expect(picker.locator(".query-row")).toHaveCount(0);
    await expect(picker.locator(".placeholder")).toHaveText("No matches");
  });

  test("editing a saved query renames it in place", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "queries");
    const picker = page
      .locator(".modal-overlay")
      .filter({ has: page.locator("h3", { hasText: "Saved searches" }) });
    await expect(picker).toBeVisible();

    // Open the edit dialog for "Invoices" via its pencil button.
    await picker
      .locator(".query-row")
      .filter({ hasText: "Invoices" })
      .getByTitle("Edit")
      .click();
    const edit = page
      .locator(".modal-overlay")
      .filter({ has: page.locator("h3", { hasText: "Edit saved search" }) });
    await expect(edit).toBeVisible();

    // Rename and save; the picker reflects the new name.
    await edit.locator("input").first().fill("Unpaid invoices");
    await edit.getByRole("button", { name: "Save" }).click();
    await expect(edit).toBeHidden();
    await expect(
      picker.locator(".query-row").filter({ hasText: "Unpaid invoices" }),
    ).toHaveCount(1);
  });

  test("Shift+E opens the edit dialog for the highlighted query", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "queries");
    const picker = page
      .locator(".modal-overlay")
      .filter({ has: page.locator("h3", { hasText: "Saved searches" }) });
    await expect(picker).toBeVisible();

    // The filter input is focused, so a bare "e" would type — Shift+E edits the
    // highlighted row via the shared usePickerCrud handler.
    await page.keyboard.press("Shift+E");
    await expect(
      page
        .locator(".modal-overlay")
        .filter({ has: page.locator("h3", { hasText: "Edit saved search" }) }),
    ).toBeVisible();
  });

  test("Escape in the edit dialog closes only the modal, not the picker", async ({
    page,
  }) => {
    await openApp(page);
    await runCommand(page, "queries");
    const picker = page
      .locator(".modal-overlay")
      .filter({ has: page.locator("h3", { hasText: "Saved searches" }) });
    await expect(picker).toBeVisible();

    // Open the edit dialog for "Invoices" via its pencil button.
    await picker
      .locator(".query-row")
      .filter({ hasText: "Invoices" })
      .getByTitle("Edit")
      .click();
    const edit = page
      .locator(".modal-overlay")
      .filter({ has: page.locator("h3", { hasText: "Edit saved search" }) });
    await expect(edit).toBeVisible();

    // Escape must dismiss only the edit modal — the picker underneath must stay
    // open (regression: a single Escape used to close both).
    await page.keyboard.press("Escape");
    await expect(edit).toBeHidden();
    await expect(picker).toBeVisible();
  });
});

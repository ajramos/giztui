import { test, expect } from "@playwright/test";
import { openApp, runCommand } from "./helpers";

// Prompts are managed inline in the picker (edit/delete/new) — the same model as
// saved searches — so there is no separate manager surface anymore.
test.describe("prompts picker inline CRUD", () => {
  const picker = (page) =>
    page
      .locator(".modal-overlay")
      .filter({ has: page.locator("h3", { hasText: "Prompts" }) });
  const editModal = (page, title: string) =>
    page
      .locator(".modal-overlay")
      .filter({ has: page.locator("h3", { hasText: title }) });

  test("edit renames a prompt in place", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "prompt");
    await expect(picker(page)).toBeVisible();
    await expect(picker(page).locator(".query-row")).toHaveCount(4);

    await picker(page)
      .locator(".query-row")
      .filter({ hasText: "Extract action items" })
      .getByTitle("Edit")
      .click();
    const edit = editModal(page, "Edit prompt");
    await expect(edit).toBeVisible();
    await edit.locator("input").first().fill("Pull out tasks");
    await edit.getByRole("button", { name: "Save" }).click();
    await expect(edit).toBeHidden();
    await expect(
      picker(page).locator(".query-row").filter({ hasText: "Pull out tasks" }),
    ).toHaveCount(1);
  });

  test("trash deletes a prompt in place", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "prompt");
    await expect(picker(page).locator(".query-row")).toHaveCount(4);
    await picker(page)
      .locator(".query-row")
      .filter({ hasText: "Summarize concisely" })
      .getByTitle("Delete")
      .click();
    // Deleting now asks to confirm first.
    const confirm = page.locator(".modal.confirm");
    await expect(confirm).toBeVisible();
    await confirm.getByRole("button", { name: "Delete" }).click();
    await expect(picker(page).locator(".query-row")).toHaveCount(3);
    await expect(
      picker(page).locator(".query-row").filter({ hasText: "Summarize concisely" }),
    ).toHaveCount(0);
  });

  test("New prompt creates a row", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "prompt");
    await picker(page).locator(".modal-foot").getByRole("button", { name: "New" }).click();
    const create = editModal(page, "New prompt");
    await expect(create).toBeVisible();
    await create.locator("input").first().fill("Translate to English");
    await create.locator("textarea").fill("Translate:\n\n{{body}}");
    await create.getByRole("button", { name: "Save" }).click();
    await expect(create).toBeHidden();
    await expect(
      picker(page).locator(".query-row").filter({ hasText: "Translate to English" }),
    ).toHaveCount(1);
  });

  test("typing uppercase in the filter does NOT trigger edit/new/delete", async ({
    page,
  }) => {
    await openApp(page);
    await runCommand(page, "prompt");
    await expect(picker(page)).toBeVisible();
    // Enter filter mode and type a name with capitals (the old Shift+E/Shift+N
    // hack used to fire edit/new here).
    await page.keyboard.press("/");
    await page.keyboard.type("Extract New");
    await expect(picker(page).locator(".label-filter")).toHaveValue("Extract New");
    // No edit or new dialog opened — the keys were just text.
    await expect(
      page.locator(".modal-overlay").filter({ has: page.locator("h3", { hasText: "prompt" }) }),
    ).toHaveCount(1); // only the picker itself ("Prompts")
  });

  test("bare 'n' opens the New prompt dialog (list mode)", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "prompt");
    await expect(picker(page)).toBeVisible();
    // List mode on open: bare "n" creates; "/" would type into the filter instead.
    await page.keyboard.press("n");
    await expect(editModal(page, "New prompt")).toBeVisible();
  });

  test("Escape in the edit modal closes only the modal, not the picker", async ({
    page,
  }) => {
    await openApp(page);
    await runCommand(page, "prompt");
    await picker(page)
      .locator(".query-row")
      .filter({ hasText: "Extract action items" })
      .getByTitle("Edit")
      .click();
    const edit = editModal(page, "Edit prompt");
    await expect(edit).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(edit).toBeHidden();
    await expect(picker(page)).toBeVisible();
  });
});

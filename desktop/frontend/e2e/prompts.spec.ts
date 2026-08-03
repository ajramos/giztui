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
    await expect(picker(page).locator(".query-row")).toHaveCount(3);
    await expect(
      picker(page).locator(".query-row").filter({ hasText: "Summarize concisely" }),
    ).toHaveCount(0);
  });

  test("New prompt creates a row", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "prompt");
    await picker(page).getByRole("button", { name: "New prompt" }).click();
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

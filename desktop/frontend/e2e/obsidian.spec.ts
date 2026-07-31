import { test, expect } from "@playwright/test";
import { openApp, openMessageAt, runCommand } from "./helpers";

// The Obsidian ingest dialog (TUI parity): ":obsidian" opens a dialog for an
// optional comment (rendered into the note as "> **Note:** …"), then ingests.
// This guards the wiring that replaced the old fire-and-forget send, which
// dropped the user's comment entirely.
test.describe("Obsidian ingest dialog", () => {
  test("':obsidian' opens the dialog; Enter ingests with the comment and toasts", async ({
    page,
  }) => {
    await openApp(page);
    await openMessageAt(page, 0);

    await runCommand(page, "obsidian");
    const dialog = page
      .locator(".modal-overlay")
      .filter({ has: page.locator("h3", { hasText: "Send to Obsidian" }) });
    await expect(dialog).toBeVisible();

    const input = dialog.locator(".slack-premessage");
    await input.fill("follow up next week");
    await input.press("Enter");

    await expect(page.locator(".toast")).toContainText("Saved to Obsidian");
    await expect(dialog).toBeHidden();
  });

  test("Escape closes the Obsidian dialog without ingesting", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 0);
    await runCommand(page, "obsidian");
    const dialog = page
      .locator(".modal-overlay")
      .filter({ has: page.locator("h3", { hasText: "Send to Obsidian" }) });
    await expect(dialog).toBeVisible();

    await page.keyboard.press("Escape");
    await expect(dialog).toBeHidden();
  });
});

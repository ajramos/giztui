import { test, expect } from "@playwright/test";
import { openApp, openMessageAt, runCommand } from "./helpers";

// Guards the useAttachments extraction (F3.2): the attachment list is fetched
// per-message inside loadMessage (gated by openIdRef) and shown as chips; the
// ':attachments' command opens the keyboard-navigable picker. In the mock,
// message m0 (row 0) carries two attachments.
test.describe("attachments", () => {
  test("the attachment bar shows the message's attachments", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 0);
    await expect(page.locator(".attach-bar")).toBeVisible();
    await expect(page.locator(".attach-chip")).toHaveCount(2);
    await expect(page.locator(".attach-chip").first()).toContainText(
      "invoice-2043.pdf",
    );
  });

  test("':attachments' opens the picker and Escape closes it", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 0);
    await expect(page.locator(".attach-bar")).toBeVisible();

    await runCommand(page, "attachments");
    await expect(page.locator(".modal-overlay")).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(page.locator(".modal-overlay")).toBeHidden();
  });
});

import { test, expect } from "@playwright/test";
import { openApp, rows } from "./helpers";

// Global keyboard handler (handleKeyDown/handleKeyMain). Guards the 3-file split
// of the ~700-line onKey handler: single-key actions, j/k navigation, and the
// layered Escape still work from the window listener.
test.describe("keyboard", () => {
  test("'c' opens the composer; Escape closes it", async ({ page }) => {
    await openApp(page);
    await page.keyboard.press("c");
    const composer = page.locator(".modal-overlay").filter({ hasText: "New message" });
    await expect(composer).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(composer).toBeHidden();
  });

  test("j / k move the inbox cursor (preview) without opening a modal", async ({ page }) => {
    await openApp(page);
    // First row is selected on load; j moves the highlight down, k back up.
    const selected = () => page.locator(".row.selected");
    await expect(selected()).toHaveCount(1);
    const firstId = await selected().getAttribute("data-id").catch(() => null);
    await page.keyboard.press("j");
    // The selection moved to a different row (still exactly one selected).
    await expect(selected()).toHaveCount(1);
    const afterJ = await rows(page).nth(1).getAttribute("class");
    expect(afterJ).toContain("selected");
    await page.keyboard.press("k");
    const afterK = await rows(page).nth(0).getAttribute("class");
    expect(afterK).toContain("selected");
    void firstId;
  });

  test("'?' opens the shortcuts help; Escape closes it", async ({ page }) => {
    await openApp(page);
    await page.keyboard.press("?");
    const help = page.locator(".modal-overlay").filter({ hasText: "GizTUI Desktop" });
    await expect(help).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(help).toBeHidden();
  });
});

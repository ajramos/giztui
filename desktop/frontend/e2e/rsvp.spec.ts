import { test, expect } from "@playwright/test";
import { openApp, openMessageAt, runCommand } from "./helpers";

// Guards the useRsvp extraction (F3.2): the invite is fetched per-message inside
// loadMessage (gated by openIdRef + the enabled ref) and surfaced as the RSVP
// bar; ':rsvp' opens the picker; responding calls the backend and toasts. In the
// mock, message m0 (row 0) is a calendar invite.
test.describe("RSVP", () => {
  test("the RSVP bar shows for a calendar invite", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 0);
    await expect(page.locator(".rsvp-bar")).toBeVisible();
    await expect(page.locator(".rsvp-btn.accept")).toBeVisible();
    await expect(page.locator(".rsvp-btn.decline")).toBeVisible();
  });

  test("':rsvp' opens the RSVP picker and Escape closes it", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 0);
    await expect(page.locator(".rsvp-bar")).toBeVisible();

    await runCommand(page, "rsvp");
    await expect(page.locator(".modal-overlay")).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(page.locator(".modal-overlay")).toBeHidden();
  });

  test("responding to an invite confirms with a toast", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 0);
    await expect(page.locator(".rsvp-bar")).toBeVisible();

    await page.locator(".rsvp-btn.accept").click();
    await expect(page.locator(".toast")).toContainText("RSVP: accepted");
  });
});

import { test, expect } from "@playwright/test";
import { openApp, openMessageAt, runCommand } from "./helpers";

// The Slack forward picker (TUI parity): ":slack" opens a dialog to pick a
// channel and add an optional pre-message, then forwards. This guards the wiring
// that replaced the old fire-and-forget forward (which ignored channel choice and
// the configured format_style).
test.describe("Slack forward picker", () => {
  test("':slack' opens the channel picker; Enter forwards and toasts", async ({
    page,
  }) => {
    await openApp(page);
    await openMessageAt(page, 0);

    await runCommand(page, "slack");
    const picker = page
      .locator(".modal-overlay")
      .filter({ has: page.locator("h3", { hasText: "Forward to Slack" }) });
    await expect(picker).toBeVisible();
    // Both configured channels are listed, default first.
    await expect(picker.locator(".prompt-row").first()).toContainText("team-updates");
    await expect(picker.locator(".prompt-row")).toHaveCount(2);

    // Add a pre-message and send.
    await picker.locator(".slack-premessage").fill("heads up on this");
    await picker.locator(".slack-premessage").press("Enter");

    await expect(page.locator(".toast")).toContainText("Forwarded to Slack");
    await expect(picker).toBeHidden();
  });

  test("Escape closes the Slack picker without forwarding", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 0);
    await runCommand(page, "slack");
    const picker = page
      .locator(".modal-overlay")
      .filter({ has: page.locator("h3", { hasText: "Forward to Slack" }) });
    await expect(picker).toBeVisible();

    await page.keyboard.press("Escape");
    await expect(picker).toBeHidden();
  });
});

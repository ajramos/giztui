import { test, expect } from "@playwright/test";
import { openApp, openMessageAt, runCommand } from "./helpers";

// Pickers are keyboard-first and driven by window-level listeners (WKWebView
// won't focus a bare div). The refactor must keep: open via command, arrow
// navigation updating the active row, and Escape closing from anywhere.

test.describe("pickers", () => {
  test("':labels' opens the labels picker and Escape closes it", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 0);

    await runCommand(page, "labels");
    await expect(page.locator(".modal-overlay")).toBeVisible();

    await page.keyboard.press("Escape");
    await expect(page.locator(".modal-overlay")).toBeHidden();
  });

  test("arrow keys move the active row in a picker", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 0);

    await runCommand(page, "labels");
    await expect(page.locator(".modal-overlay")).toBeVisible();

    const active = page.locator(".modal-overlay .nav-active");
    await expect(active).toHaveCount(1);
    const first = (await active.textContent())?.trim();

    await page.keyboard.press("ArrowDown");
    const moved = (await page.locator(".modal-overlay .nav-active").textContent())?.trim();
    expect(moved).not.toBe(first);

    await page.keyboard.press("Escape");
    await expect(page.locator(".modal-overlay")).toBeHidden();
  });

  test("':theme' opens the theme picker and Escape closes it", async ({ page }) => {
    await openApp(page);

    await runCommand(page, "theme");
    await expect(page.locator(".modal-overlay")).toBeVisible();

    await page.keyboard.press("Escape");
    await expect(page.locator(".modal-overlay")).toBeHidden();
  });
});

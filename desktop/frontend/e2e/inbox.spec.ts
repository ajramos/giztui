import { test, expect } from "@playwright/test";
import { openApp, openMessageAt, rows, runCommand } from "./helpers";

test.describe("inbox + reader", () => {
  test("renders the mock inbox and opens a message in the reader", async ({ page }) => {
    await openApp(page);
    await expect(rows(page)).toHaveCount(24);

    const subject = await openMessageAt(page, 0);
    expect(subject).toBe("Welcome to GizTUI Desktop");
    // Reader body renders the full message content from the mock.
    await expect(page.locator(".reader-body")).toBeVisible();
    await expect(page.locator(".reader-body")).toContainText(subject);
  });

  test("switching rows swaps the reader content", async ({ page }) => {
    await openApp(page);
    await openMessageAt(page, 0);
    await expect(page.locator(".reader-head h2")).toHaveText("Welcome to GizTUI Desktop");

    await openMessageAt(page, 2);
    await expect(page.locator(".reader-head h2")).toHaveText("Re: Project roadmap Q3");
  });

  test("the command palette opens with ':' and closes on Escape", async ({ page }) => {
    await openApp(page);
    await page.keyboard.press(":");
    await expect(page.locator(".cmd-bar")).toBeVisible();
    // The input auto-focuses so typing filters immediately.
    await expect(page.locator(".cmd-input")).toBeFocused();
    await page.keyboard.press("Escape");
    await expect(page.locator(".cmd-bar")).toBeHidden();
  });

  test("'load more' is a no-op on the mock but does not break the list", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "inbox");
    await expect(rows(page)).toHaveCount(24);
  });
});

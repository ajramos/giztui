import { test, expect, type Page } from "@playwright/test";
import { openApp, runCommand } from "./helpers";

// Guards the useTheme extraction (F3.1): the startup init applies the configured
// theme, and ':theme <name>' maps that theme's palette onto the CSS custom
// properties the stylesheet reads. The mock resolves an empty/unknown name to
// "slate-blue" and defines "dracula".
const cssVar = (page: Page, name: string) =>
  page.evaluate(
    (n) => getComputedStyle(document.documentElement).getPropertyValue(n).trim(),
    name,
  );

test.describe("theme", () => {
  test("startup applies the configured theme's palette", async ({ page }) => {
    await openApp(page);
    // Empty name resolves to slate-blue in the mock backend.
    expect(await cssVar(page, "--bg")).toBe("#0f172a");
  });

  test("':theme <name>' applies that theme's colors", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "theme dracula");
    expect(await cssVar(page, "--bg")).toBe("#282a36");
    expect(await cssVar(page, "--accent")).toBe("#bd93f9");
  });

  test("':theme' with no arg opens the picker; Escape closes it", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "theme");
    await expect(page.locator(".modal-overlay")).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(page.locator(".modal-overlay")).toBeHidden();
  });
});

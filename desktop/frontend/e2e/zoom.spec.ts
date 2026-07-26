import { test, expect } from "@playwright/test";
import { openApp, rows, runCommand } from "./helpers";

// Guards the useZoom extraction (F3.1): zoom is CSS `zoom` on the document root,
// clamped/rounded, and persisted to localStorage across reloads.
const rootZoom = (page: import("@playwright/test").Page) =>
  page.evaluate(() => document.documentElement.style.zoom || "");

test.describe("UI zoom", () => {
  test("zoom commands change the root CSS zoom", async ({ page }) => {
    await openApp(page);

    await runCommand(page, "zoom-reset");
    expect(await rootZoom(page)).toBe("1");

    await runCommand(page, "zoom-in");
    expect(await rootZoom(page)).toBe("1.1");

    await runCommand(page, "zoom-out");
    expect(await rootZoom(page)).toBe("1");

    // ':zoom <n>' sets an absolute level; ':zoom' with no arg resets.
    await runCommand(page, "zoom 1.5");
    expect(await rootZoom(page)).toBe("1.5");
    await runCommand(page, "zoom");
    expect(await rootZoom(page)).toBe("1");
  });

  test("zoom persists across a reload", async ({ page }) => {
    await openApp(page);
    await runCommand(page, "zoom 1.3");
    expect(await rootZoom(page)).toBe("1.3");

    await page.reload();
    await expect(rows(page).first()).toBeVisible();
    expect(await rootZoom(page)).toBe("1.3");

    // Restore so a reused context doesn't leak a zoom into later work.
    await runCommand(page, "zoom-reset");
  });
});

import { type Page, type Locator, expect } from "@playwright/test";

// Shared helpers for driving the desktop app through its command palette and
// message list — the two stable entry points that survive the App.tsx
// decomposition. Prefer commands over raw shortcuts: the COMMANDS table is the
// documented contract and is far less brittle than key-by-key bindings.

export const rows = (page: Page): Locator => page.locator(".row");
export const summaryPanel = (page: Page): Locator =>
  page.locator(".summary-panel:not(.prompt-panel)");
export const promptPanel = (page: Page): Locator =>
  page.locator(".summary-panel.prompt-panel");

// openApp navigates to the app and waits for the mock inbox to render.
export async function openApp(page: Page): Promise<void> {
  await page.goto("/");
  await expect(rows(page).first()).toBeVisible();
}

// runCommand opens the ":" command palette, types a command, and submits it.
// Assumes no text input currently holds focus (the normal state between flows).
export async function runCommand(page: Page, command: string): Promise<void> {
  await page.keyboard.press(":");
  const input = page.locator(".cmd-input");
  await expect(input).toBeVisible();
  await input.fill(command);
  await page.keyboard.press("Enter");
  // The palette closes itself on submit; wait for it so the next window-level
  // keypress (a following ":" or shortcut) isn't swallowed by the closing modal.
  await expect(page.locator(".cmd-bar")).toBeHidden();
}

// openMessageAt clicks the Nth message row and waits for the reader to show it.
export async function openMessageAt(page: Page, index: number): Promise<string> {
  const row = rows(page).nth(index);
  const subject = (await row.locator(".subject").first().textContent())?.trim() ?? "";
  await row.click();
  await expect(page.locator(".reader-head h2")).toHaveText(subject);
  return subject;
}

// readerSubject returns the subject currently shown in the reading pane.
export async function readerSubject(page: Page): Promise<string> {
  return (await page.locator(".reader-head h2").textContent())?.trim() ?? "";
}

// waitForPanelDone waits until an AiPanel has finished streaming. AiPanel only
// renders its regenerate/dismiss actions once `text !== null && !generating`, so
// the presence of the "regenerate" button is the reliable "done" signal — the
// streaming caret (▍) is gone and the text is final.
export async function waitForPanelDone(panel: Locator): Promise<string> {
  await expect(
    panel.locator(".summary-head-actions button", { hasText: "regenerate" }),
  ).toBeVisible();
  return (await panel.locator(".summary-text").textContent())?.trim() ?? "";
}

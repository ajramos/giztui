import { defineConfig } from "@playwright/test";

// Playwright integration suite for the desktop frontend.
//
// This is the regression net that guards the F3 refactor (extracting the
// coupled subsystem hooks out of App.tsx). It drives the REAL React app against
// the in-browser api.ts mock backend — the same mock `npm run dev` uses — so it
// exercises the coupled runtime behavior (window-level key listeners, streaming
// order, focus, the summary/prompt ForId gating) that jsdom unit tests cannot.
//
// Chromium comes from the pre-installed browser (PLAYWRIGHT_BROWSERS_PATH is set
// in this environment); we never download one. Run with `npm run test:e2e`.
const PORT = 5199;

export default defineConfig({
  testDir: "./e2e",
  // The flows share one mock backend and mutate it (labels, trash); keep them
  // serial and deterministic rather than racing parallel workers.
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  timeout: 30_000,
  expect: { timeout: 8_000 },
  reporter: [["list"]],
  use: {
    baseURL: `http://localhost:${PORT}`,
    trace: "retain-on-failure",
    viewport: { width: 1280, height: 850 },
  },
  projects: [{ name: "chromium", use: { browserName: "chromium" } }],
  webServer: {
    command: `npm run dev -- --port ${PORT} --strictPort`,
    url: `http://localhost:${PORT}`,
    reuseExistingServer: !process.env.CI,
    timeout: 90_000,
  },
});

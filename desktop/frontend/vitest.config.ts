import { defineConfig } from "vitest/config";

// Unit-test config, separate from vite.config.ts (the build). Pure logic lives
// in small modules extracted from App.tsx; jsdom gives us DOM types (e.g.
// KeyboardEvent) without a browser. Integration flows are covered separately by
// Playwright against the api.ts mock.
export default defineConfig({
  test: {
    environment: "jsdom",
    include: ["src/**/*.test.ts"],
  },
});

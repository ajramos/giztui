import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Wails serves the contents of dist/ from its embedded asset server, so we emit
// a relative-base build that works when loaded from the app's internal origin.
export default defineConfig({
  plugins: [react()],
  base: "./",
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});

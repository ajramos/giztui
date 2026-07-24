import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";

// Vite tags module scripts/styles with `crossorigin`. Under macOS WKWebView the
// Wails asset server is served from an internal scheme, and crossorigin requests
// against it fail CORS — leaving the window blank. Stripping the attribute makes
// the bundle load. It is harmless in a normal browser (same-origin).
function stripCrossorigin(): Plugin {
  return {
    name: "strip-crossorigin",
    transformIndexHtml(html) {
      return html.replace(/\s+crossorigin(=".*?")?/g, "");
    },
  };
}

export default defineConfig({
  plugins: [react(), stripCrossorigin()],
  base: "./",
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});

import { useCallback, useState } from "react";
import { backend } from "./api";
import { mixHex } from "./format";

// useTheme owns the desktop theme subsystem: whether theming is enabled, the
// available theme names, the current one, the picker's open state, and applying
// a theme's palette onto the CSS custom properties the stylesheet reads.
// Extracted from App.tsx unchanged (F3.1) — no behavior change; applyTheme and
// the startup init are the same logic, just moved behind a hook.
export interface Theme {
  themesOn: boolean;
  themePickerOpen: boolean;
  setThemePickerOpen: (v: boolean) => void;
  themeNames: string[];
  currentTheme: string;
  // applyTheme fetches a theme's palette from the backend and maps it onto the
  // CSS custom properties. Empty name = the configured (current) theme.
  applyTheme: (name: string) => Promise<void>;
  // initTheme resolves enablement + names and applies the configured theme;
  // called once from the App's startup effect.
  initTheme: () => Promise<void>;
}

export function useTheme(): Theme {
  const [themesOn, setThemesOn] = useState(false);
  const [themePickerOpen, setThemePickerOpen] = useState(false);
  const [themeNames, setThemeNames] = useState<string[]>([]);
  const [currentTheme, setCurrentTheme] = useState("");

  const applyTheme = useCallback(async (name: string) => {
    try {
      const c = await backend.GetThemeColors(name);
      if (!c) return;
      const root = document.documentElement.style;
      const set = (k: string, v: string) => {
        if (v) root.setProperty(k, v);
      };
      // Map the theme palette onto the CSS custom properties the stylesheet
      // reads. Elevated/hover surfaces are derived from the base background so
      // the UI keeps its layering even when a theme only defines a few colors.
      // The accent (focus color) — never the title color — drives buttons, so
      // primary buttons stay in the theme's accent hue instead of turning into
      // a loud title color (e.g. bright green).
      const bg = c.bg || "#14161b";
      const fg = c.fg || "#e6e8ec";
      const accent = c.accent || "#6ea8fe";
      const distinct = (v: string) =>
        v && v.toLowerCase() !== bg.toLowerCase() ? v : "";
      const elev = distinct(c.inputBg) || mixHex(bg, fg, 0.06);
      const rowHover = distinct(c.selectionBg) || mixHex(bg, fg, 0.1);
      const selected = distinct(c.selectionBg) || mixHex(bg, accent, 0.22);
      set("--bg", bg);
      set("--bg-elev", elev);
      set("--bg-row", bg);
      set("--bg-row-hover", rowHover);
      set("--bg-selected", selected);
      set("--border", c.border || mixHex(bg, fg, 0.16));
      set("--text", fg);
      set("--text-muted", c.muted || mixHex(fg, bg, 0.4));
      // Readable secondary text (keyboard hints, ghost buttons, loading) —
      // always derived from fg/bg so it stays legible regardless of how faint a
      // theme's own muted color is. t is the weight toward bg, so a smaller value
      // than --text-muted's 0.4 keeps this closer to the text colour (higher
      // contrast).
      set("--text-dim", mixHex(fg, bg, 0.2));
      set("--accent", accent);
      set("--accent-strong", accent);
      set("--danger", c.danger || "#ff6b6b");
      set("--chip-bg", mixHex(bg, fg, 0.12));
      set("--chip-text", fg);
      set("--unread-dot", accent);
      if (c.name) setCurrentTheme(c.name);
    } catch {
      /* non-fatal: keep the default palette */
    }
  }, []);

  const initTheme = useCallback(async () => {
    try {
      const on = await backend.ThemesEnabled();
      setThemesOn(on);
      if (on) {
        setThemeNames(await backend.ListThemes());
        await applyTheme(""); // apply the configured theme
      }
    } catch {
      /* non-fatal */
    }
  }, [applyTheme]);

  return {
    themesOn,
    themePickerOpen,
    setThemePickerOpen,
    themeNames,
    currentTheme,
    applyTheme,
    initTheme,
  };
}

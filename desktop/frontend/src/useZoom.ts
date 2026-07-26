import { useCallback, useEffect, useState } from "react";

// useZoom owns the UI zoom level. WKWebView doesn't honour Cmd/Ctrl +/-/0 to
// scale the app, so we drive it ourselves via CSS `zoom` on the document root
// and remember the choice in localStorage. Extracted from App.tsx unchanged
// (F3.1) — behavior is identical: same clamp range, same rounding, same key.
const MIN_ZOOM = 0.6;
const MAX_ZOOM = 2.4;

export interface Zoom {
  // setZoom sets an absolute level. Callers validate the range (e.g. the ":zoom
  // <n>" command), so this is the raw setter to keep behavior byte-identical.
  setZoom: (n: number) => void;
  // bumpZoom nudges the level by delta, clamped and rounded to one decimal.
  bumpZoom: (delta: number) => void;
  // resetZoom returns to 1.0.
  resetZoom: () => void;
}

export function useZoom(): Zoom {
  const [uiZoom, setUiZoom] = useState(() => {
    const v = Number(localStorage.getItem("giztui.zoom"));
    return v >= MIN_ZOOM && v <= MAX_ZOOM ? v : 1;
  });
  useEffect(() => {
    (document.documentElement.style as unknown as { zoom: string }).zoom =
      String(uiZoom);
    localStorage.setItem("giztui.zoom", String(uiZoom));
  }, [uiZoom]);
  const bumpZoom = useCallback((delta: number) => {
    setUiZoom((z) =>
      Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, Math.round((z + delta) * 10) / 10)),
    );
  }, []);
  const resetZoom = useCallback(() => setUiZoom(1), []);
  return { setZoom: setUiZoom, bumpZoom, resetZoom };
}

import { useCallback, useEffect, useState, type Dispatch, type SetStateAction } from "react";
import { backend } from "./api";

// useAutoRefresh owns the background inbox poll: whether it's on, the interval,
// and the toggle (which persists the on/off choice to localStorage). The actual
// poll (checkNewMail) stays in App and is passed as onTick, since it reads the
// message/search subsystem. Extracted from App.tsx unchanged. setAutoRefresh /
// setAutoRefreshSecs are exposed so init can seed them from config.
export interface AutoRefresh {
  autoRefresh: boolean;
  setAutoRefresh: Dispatch<SetStateAction<boolean>>;
  autoRefreshSecs: number;
  setAutoRefreshSecs: Dispatch<SetStateAction<number>>;
  toggleAutoRefresh: () => void;
}

export function useAutoRefresh(deps: {
  showToast: (m: string) => void;
  onTick: () => void;
}): AutoRefresh {
  const { showToast, onTick } = deps;
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [autoRefreshSecs, setAutoRefreshSecs] = useState(60);

  const toggleAutoRefresh = useCallback(() => {
    setAutoRefresh((v) => {
      const next = !v;
      // Persist to config.json — the single source of truth, read back on startup.
      // The desktop never sends the Slack new-mail digest, but writing enabled:false
      // here also stops any TUI launched later from re-arming it from a stale
      // enabled:true. Fire-and-forget: the local toggle already took effect.
      void backend.SetAutoRefreshEnabled(next).catch(() => {});
      showToast(next ? "Auto-refresh on" : "Auto-refresh off");
      return next;
    });
  }, [showToast]);

  useEffect(() => {
    if (!autoRefresh) return;
    const ms = Math.max(30, autoRefreshSecs) * 1000;
    const timer = setInterval(() => onTick(), ms);
    return () => clearInterval(timer);
  }, [autoRefresh, autoRefreshSecs, onTick]);

  return {
    autoRefresh,
    setAutoRefresh,
    autoRefreshSecs,
    setAutoRefreshSecs,
    toggleAutoRefresh,
  };
}

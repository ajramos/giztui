import { useCallback, useEffect, useState, type Dispatch, type SetStateAction } from "react";

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
      localStorage.setItem("giztui.autorefresh", next ? "on" : "off");
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

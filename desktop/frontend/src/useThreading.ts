import {
  useCallback,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";
import { backend, type MessageDetail } from "./api";

// useThreading owns the conversation-view subsystem: whether threading is
// enabled, the loaded thread messages, which are collapsed, the loading flag,
// and toggling the thread open/closed. Extracted from App.tsx unchanged (F3.2).
//
// NOTE: summarizeThread stays in App.tsx on purpose — it writes the AI summary
// panel state (setSummarizing/setSummary), which belongs to the AI-panels
// subsystem (F3.4), not here. This hook only owns the thread list itself.
export interface Threading {
  threadingOn: boolean;
  setThreadingOn: Dispatch<SetStateAction<boolean>>;
  threadMsgs: MessageDetail[] | null;
  setThreadMsgs: Dispatch<SetStateAction<MessageDetail[] | null>>;
  collapsedMsgs: Set<string>;
  setCollapsedMsgs: Dispatch<SetStateAction<Set<string>>>;
  loadingThread: boolean;
  toggleThread: () => Promise<void>;
}

export function useThreading(
  detail: MessageDetail | null,
  deps: { setError: (e: string) => void },
): Threading {
  const { setError } = deps;
  const [threadingOn, setThreadingOn] = useState(false);
  const [threadMsgs, setThreadMsgs] = useState<MessageDetail[] | null>(null);
  const [collapsedMsgs, setCollapsedMsgs] = useState<Set<string>>(new Set());
  const [loadingThread, setLoadingThread] = useState(false);

  const toggleThread = useCallback(async () => {
    if (!detail) return;
    if (threadMsgs) {
      setThreadMsgs(null);
      return;
    }
    setLoadingThread(true);
    setError("");
    try {
      const msgs = await backend.GetThread(detail.threadId);
      setThreadMsgs(msgs);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoadingThread(false);
    }
  }, [detail, threadMsgs, setError]);

  return {
    threadingOn,
    setThreadingOn,
    threadMsgs,
    setThreadMsgs,
    collapsedMsgs,
    setCollapsedMsgs,
    loadingThread,
    toggleThread,
  };
}

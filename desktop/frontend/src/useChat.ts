import { useCallback, useRef, useState } from "react";
import { backend, chatStream } from "./api";

export interface ChatTurn {
  role: "user" | "assistant";
  text: string;
}

// useChat owns the "chat with this email" panel: a per-message conversation
// transcript, the input box, and the streaming send. History is kept per message
// id (both here for instant re-display and on the backend, which grounds each
// turn on the email + prior turns). The panel is scoped to the open message via
// forId so a chat opened on one email never shows under another.
export function useChat(deps: {
  detailId: () => string | null;
  setError: (e: string) => void;
}) {
  const { detailId, setError } = deps;
  const [chatOpen, setChatOpen] = useState(false);
  const [chatForId, setChatForId] = useState<string | null>(null);
  const [chatInput, setChatInput] = useState("");
  const [chatStreaming, setChatStreaming] = useState(false);
  const [chatStreamingText, setChatStreamingText] = useState("");
  const [chatTurns, setChatTurns] = useState<ChatTurn[]>([]);
  // Per-message transcripts, so switching emails and returning restores the chat.
  const store = useRef<Map<string, ChatTurn[]>>(new Map());

  const openChat = useCallback(() => {
    const id = detailId();
    if (!id) return;
    setChatForId(id);
    setChatTurns(store.current.get(id) ?? []);
    setChatOpen(true);
  }, [detailId]);

  const closeChat = useCallback(() => setChatOpen(false), []);

  const resetChat = useCallback(() => {
    const id = chatForId;
    if (!id) return;
    store.current.set(id, []);
    setChatTurns([]);
    void backend.ChatReset(id);
  }, [chatForId]);

  const sendChat = useCallback(async () => {
    const id = chatForId;
    const msg = chatInput.trim();
    if (!id || !msg || chatStreaming) return;
    const base = store.current.get(id) ?? [];
    const withUser: ChatTurn[] = [...base, { role: "user", text: msg }];
    store.current.set(id, withUser);
    setChatTurns(withUser);
    setChatInput("");
    setChatStreaming(true);
    setChatStreamingText("");
    try {
      let acc = "";
      const final = await chatStream(id, msg, (tok) => {
        acc += tok;
        setChatStreamingText(acc);
      });
      const withReply: ChatTurn[] = [...withUser, { role: "assistant", text: final }];
      store.current.set(id, withReply);
      // Only paint into the visible transcript if this chat is still the open one.
      if (chatForId === id) setChatTurns(withReply);
    } catch (e) {
      setError(String(e));
    } finally {
      setChatStreaming(false);
      setChatStreamingText("");
    }
  }, [chatForId, chatInput, chatStreaming, setError]);

  return {
    chatOpen,
    chatForId,
    chatTurns,
    chatInput,
    setChatInput,
    chatStreaming,
    chatStreamingText,
    openChat,
    closeChat,
    sendChat,
    resetChat,
  };
}

export type ChatBundle = ReturnType<typeof useChat>;

import { useCallback, useState, type Dispatch, type SetStateAction } from "react";
import { backend } from "./api";

// useIntegrations owns the optional outbound integrations (Obsidian, Slack):
// whether each is enabled (from backend config) and the fire-and-forget send
// actions. Extracted from App.tsx unchanged. Call refresh() during init to pull
// the enabled flags.
export interface Integrations {
  obsidianOn: boolean;
  slackOn: boolean;
  refresh: () => Promise<void>;
  // Obsidian ingest now goes through a dialog (optional comment, TUI parity):
  // openObsidian toggles it; sendObsidian performs the ingest with the comment.
  obsidianOpen: boolean;
  setObsidianOpen: Dispatch<SetStateAction<boolean>>;
  openObsidian: () => void;
  sendObsidian: (id: string, comment: string) => void;
  // Slack forward now goes through a picker (channel + pre-message, TUI parity):
  // openSlackForward toggles the dialog; forwardSlack performs the actual send.
  slackForwardOpen: boolean;
  setSlackForwardOpen: Dispatch<SetStateAction<boolean>>;
  openSlackForward: () => void;
  forwardSlack: (
    id: string,
    channelID: string,
    userMessage: string,
    format: string,
  ) => void;
}

export function useIntegrations(deps: {
  showToast: (m: string) => void;
  setError: (e: string) => void;
}): Integrations {
  const { showToast, setError } = deps;
  const [obsidianOn, setObsidianOn] = useState(false);
  const [slackOn, setSlackOn] = useState(false);
  const [slackForwardOpen, setSlackForwardOpen] = useState(false);
  const [obsidianOpen, setObsidianOpen] = useState(false);

  const refresh = useCallback(async () => {
    setObsidianOn(await backend.ObsidianEnabled());
    setSlackOn(await backend.SlackEnabled());
  }, []);

  const openObsidian = useCallback(() => setObsidianOpen(true), []);

  const sendObsidian = useCallback(
    (id: string, comment: string) => {
      setObsidianOpen(false);
      showToast("Sending to Obsidian…");
      void backend
        .SendToObsidian(id, comment)
        .then((p) => showToast(p ? `Saved to Obsidian: ${p}` : "Saved to Obsidian"))
        .catch((e) => setError(String(e)));
    },
    [showToast, setError],
  );

  const openSlackForward = useCallback(() => setSlackForwardOpen(true), []);

  const forwardSlack = useCallback(
    (id: string, channelID: string, userMessage: string, format: string) => {
      setSlackForwardOpen(false);
      showToast("Forwarding to Slack…");
      void backend
        .ForwardToSlack(id, channelID, userMessage, format)
        .then(() => showToast("Forwarded to Slack"))
        .catch((e) => setError(String(e)));
    },
    [showToast, setError],
  );

  return {
    obsidianOn,
    slackOn,
    refresh,
    obsidianOpen,
    setObsidianOpen,
    openObsidian,
    sendObsidian,
    slackForwardOpen,
    setSlackForwardOpen,
    openSlackForward,
    forwardSlack,
  };
}

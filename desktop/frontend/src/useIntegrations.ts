import { useCallback, useState } from "react";
import { backend } from "./api";

// useIntegrations owns the optional outbound integrations (Obsidian, Slack):
// whether each is enabled (from backend config) and the fire-and-forget send
// actions. Extracted from App.tsx unchanged. Call refresh() during init to pull
// the enabled flags.
export interface Integrations {
  obsidianOn: boolean;
  slackOn: boolean;
  refresh: () => Promise<void>;
  sendObsidian: (id: string) => void;
  forwardSlack: (id: string) => void;
}

export function useIntegrations(deps: {
  showToast: (m: string) => void;
  setError: (e: string) => void;
}): Integrations {
  const { showToast, setError } = deps;
  const [obsidianOn, setObsidianOn] = useState(false);
  const [slackOn, setSlackOn] = useState(false);

  const refresh = useCallback(async () => {
    setObsidianOn(await backend.ObsidianEnabled());
    setSlackOn(await backend.SlackEnabled());
  }, []);

  const sendObsidian = useCallback(
    (id: string) => {
      showToast("Sending to Obsidian…");
      void backend
        .SendToObsidian(id)
        .then((p) => showToast(p ? `Saved to Obsidian: ${p}` : "Saved to Obsidian"))
        .catch((e) => setError(String(e)));
    },
    [showToast, setError],
  );

  const forwardSlack = useCallback(
    (id: string) => {
      showToast("Forwarding to Slack…");
      void backend
        .ForwardToSlack(id)
        .then(() => showToast("Forwarded to Slack"))
        .catch((e) => setError(String(e)));
    },
    [showToast, setError],
  );

  return { obsidianOn, slackOn, refresh, sendObsidian, forwardSlack };
}

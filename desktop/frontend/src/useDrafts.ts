import { useCallback, useState, type Dispatch, type SetStateAction } from "react";
import { backend, type DraftSummary, type MessageDetail } from "./api";
import type { ComposeInit } from "./Compose";

// useDrafts owns the drafts subsystem: the drafts list, whether the left pane is
// showing drafts, and the load/open actions. Extracted from App.tsx unchanged.
// Opening a draft hands a ComposeInit to App's compose state; entering the
// drafts view clears the open message. Those cross-subsystem setters are passed
// in as deps.
export interface Drafts {
  draftsView: boolean;
  setDraftsView: Dispatch<SetStateAction<boolean>>;
  drafts: DraftSummary[];
  loadingDrafts: boolean;
  loadDrafts: () => Promise<void>;
  openDrafts: () => void;
  openDraft: (d: DraftSummary) => Promise<void>;
}

export function useDrafts(deps: {
  setError: (e: string) => void;
  setSelectedId: Dispatch<SetStateAction<string | null>>;
  setDetail: Dispatch<SetStateAction<MessageDetail | null>>;
  setCompose: Dispatch<SetStateAction<ComposeInit | null>>;
}): Drafts {
  const { setError, setSelectedId, setDetail, setCompose } = deps;
  const [draftsView, setDraftsView] = useState(false);
  const [drafts, setDrafts] = useState<DraftSummary[]>([]);
  const [loadingDrafts, setLoadingDrafts] = useState(false);

  const loadDrafts = useCallback(async () => {
    setLoadingDrafts(true);
    setError("");
    try {
      setDrafts(await backend.ListDrafts());
    } catch (e) {
      setError(String(e));
    } finally {
      setLoadingDrafts(false);
    }
  }, [setError]);

  const openDrafts = useCallback(() => {
    setDraftsView(true);
    setSelectedId(null);
    setDetail(null);
    void loadDrafts();
  }, [loadDrafts, setSelectedId, setDetail]);

  const openDraft = useCallback(
    async (d: DraftSummary) => {
      setError("");
      try {
        const det = await backend.GetDraft(d.id);
        setCompose({
          mode: "draft",
          draftId: det.id,
          to: det.to,
          cc: det.cc,
          subject: det.subject,
          body: det.body,
        });
      } catch (e) {
        setError(String(e));
      }
    },
    [setError, setCompose],
  );

  return {
    draftsView,
    setDraftsView,
    drafts,
    loadingDrafts,
    loadDrafts,
    openDrafts,
    openDraft,
  };
}

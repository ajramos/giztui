import {
  useCallback,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";
import { backend, type Attachment, type MessageDetail } from "./api";

// useAttachments owns the current message's attachment list, the picker's open
// state, and downloading an attachment. Extracted from App.tsx unchanged (F3.2).
// The list is still fetched inside loadMessage (gated by openIdRef) via
// setAttachments; this hook owns the state and the download action.
export interface Attachments {
  attachments: Attachment[];
  setAttachments: Dispatch<SetStateAction<Attachment[]>>;
  attachmentsOpen: boolean;
  setAttachmentsOpen: Dispatch<SetStateAction<boolean>>;
  downloadAttachment: (att: Attachment) => Promise<void>;
}

export function useAttachments(
  detail: MessageDetail | null,
  deps: {
    setBusy: (v: boolean) => void;
    setError: (e: string) => void;
    showToast: (m: string) => void;
  },
): Attachments {
  const { setBusy, setError, showToast } = deps;
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [attachmentsOpen, setAttachmentsOpen] = useState(false);

  const downloadAttachment = useCallback(
    async (att: Attachment) => {
      if (!detail) return;
      setBusy(true);
      try {
        const path = await backend.DownloadAttachment(
          detail.id,
          att.attachmentId,
          att.filename,
        );
        showToast(`Saved to ${path}`);
      } catch (e) {
        setError(String(e));
      } finally {
        setBusy(false);
      }
    },
    [detail, setBusy, setError, showToast],
  );

  return {
    attachments,
    setAttachments,
    attachmentsOpen,
    setAttachmentsOpen,
    downloadAttachment,
  };
}

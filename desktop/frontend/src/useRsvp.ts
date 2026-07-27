import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
} from "react";
import { backend, type Invite } from "./api";

// useRsvp owns the calendar-invite (RSVP) subsystem: whether RSVP is enabled,
// the current message's invite, the in-flight response state, and the picker.
// Extracted from App.tsx unchanged (F3.2). The per-message invite is still
// fetched inside loadMessage (gated by openIdRef) via setInvite + the enabled
// ref — this hook just owns the state and the respondInvite action.
export interface Rsvp {
  rsvpEnabled: boolean;
  setRsvpEnabled: Dispatch<SetStateAction<boolean>>;
  // Ref mirror so loadMessage (stable, no deps) can read the latest value.
  rsvpEnabledRef: MutableRefObject<boolean>;
  invite: Invite | null;
  setInvite: Dispatch<SetStateAction<Invite | null>>;
  rsvpBusy: string;
  rsvpPickerOpen: boolean;
  setRsvpPickerOpen: Dispatch<SetStateAction<boolean>>;
  respondInvite: (
    id: string,
    status: "accepted" | "tentative" | "declined",
  ) => Promise<void>;
}

export function useRsvp(deps: {
  setError: (e: string) => void;
  showToast: (m: string) => void;
}): Rsvp {
  const { setError, showToast } = deps;
  const [rsvpEnabled, setRsvpEnabled] = useState(false);
  const [invite, setInvite] = useState<Invite | null>(null);
  const [rsvpBusy, setRsvpBusy] = useState("");
  // The RSVP bar auto-shows for invites; V opens a keyboard-navigable picker.
  const [rsvpPickerOpen, setRsvpPickerOpen] = useState(false);
  const rsvpEnabledRef = useRef(false);
  useEffect(() => {
    rsvpEnabledRef.current = rsvpEnabled;
  }, [rsvpEnabled]);

  // respondInvite sends an RSVP (accepted/tentative/declined) for the open
  // message's calendar invite.
  const respondInvite = useCallback(
    async (id: string, status: "accepted" | "tentative" | "declined") => {
      setRsvpBusy(status);
      setError("");
      try {
        await backend.RespondInvite(id, status);
        showToast(`RSVP: ${status}`);
      } catch (e) {
        setError(String(e));
      } finally {
        setRsvpBusy("");
      }
    },
    [setError, showToast],
  );

  return {
    rsvpEnabled,
    setRsvpEnabled,
    rsvpEnabledRef,
    invite,
    setInvite,
    rsvpBusy,
    rsvpPickerOpen,
    setRsvpPickerOpen,
    respondInvite,
  };
}

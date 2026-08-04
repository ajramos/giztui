import { useState, type ReactNode } from "react";
import ConfirmDialog from "./ConfirmDialog";

// useConfirm centralizes the "are you sure?" step before a destructive action so
// every picker confirms deletes the same way. `ask` opens the dialog; `node` is
// the dialog element to render inside the picker; `open` lets the picker disable
// its own CRUD keys while the confirm is up (pass no items to usePickerCrud).
export function useConfirm(): {
  ask: (message: string, onConfirm: () => void, confirmLabel?: string) => void;
  node: ReactNode;
  open: boolean;
} {
  const [req, setReq] = useState<{
    message: string;
    onConfirm: () => void;
    confirmLabel?: string;
  } | null>(null);

  return {
    ask: (message, onConfirm, confirmLabel) =>
      setReq({ message, onConfirm, confirmLabel }),
    open: req !== null,
    node: req ? (
      <ConfirmDialog
        message={req.message}
        confirmLabel={req.confirmLabel}
        onConfirm={() => {
          req.onConfirm();
          setReq(null);
        }}
        onCancel={() => setReq(null)}
      />
    ) : null,
  };
}

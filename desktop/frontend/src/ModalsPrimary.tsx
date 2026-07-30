import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import type {
  MessageDetail,
  Prompt,
  SavedQuery,
  Attachment,
  Invite,
} from "./apiTypes";
import type { ComposeInit } from "./Compose";
import { formatICSDate } from "./format";
import Compose from "./Compose";
import LabelsPicker from "./LabelsPicker";
import PromptsPicker from "./PromptsPicker";
import PromptManager from "./PromptManager";
import LinksPicker from "./LinksPicker";
import SuggestPicker from "./SuggestPicker";
import AttachmentsPicker from "./AttachmentsPicker";
import SavedQueriesPicker from "./SavedQueriesPicker";
import RSVPPicker from "./RSVPPicker";
import SlackPicker from "./SlackPicker";
import SaveQueryModal from "./SaveQueryModal";

// The compose window + the picker-style modals (labels, prompts, links, suggest,
// attachments, saved queries, RSVP, save-query). A behavior-preserving lift of
// the first half of App's modal stack — every value/handler stays in App and is
// passed through, so the JSX is byte-identical to what lived inline.
type AiCache = MutableRefObject<
  Map<
    string,
    {
      summary?: string;
      touchUp?: string;
      promptResults?: Record<number, { text: string; label: string }>;
      lastPromptId?: number;
    }
  >
>;

export default function ModalsPrimary(p: {
  compose: ComposeInit | null;
  setCompose: Dispatch<SetStateAction<ComposeInit | null>>;
  showToast: (m: string) => void;
  draftsView: boolean;
  loadDrafts: () => Promise<void>;
  labelsFor: string | null;
  setLabelsFor: Dispatch<SetStateAction<string | null>>;
  applyLabelChange: (ids: Set<string>, change: { added?: string; removed?: string }) => void;
  bulkLabels: boolean;
  setBulkLabels: Dispatch<SetStateAction<boolean>>;
  selected: Set<string>;
  promptsOpen: boolean;
  setPromptsOpen: Dispatch<SetStateAction<boolean>>;
  runPrompt: (prompt: Prompt, force?: boolean) => Promise<void>;
  aiPromptsEnabled: boolean;
  setPromptManagerOpen: Dispatch<SetStateAction<boolean>>;
  promptManagerOpen: boolean;
  aiEnabled: boolean;
  aiCache: AiCache;
  setPromptResult: Dispatch<SetStateAction<string | null>>;
  linksFor: string | null;
  setLinksFor: Dispatch<SetStateAction<string | null>>;
  suggestFor: string | null;
  setSuggestFor: Dispatch<SetStateAction<string | null>>;
  suggestions: string[];
  loadingSuggest: boolean;
  applySuggestion: (name: string) => void;
  attachmentsOpen: boolean;
  setAttachmentsOpen: Dispatch<SetStateAction<boolean>>;
  attachments: Attachment[];
  busy: boolean;
  downloadAttachment: (att: Attachment) => Promise<void>;
  queriesOpen: boolean;
  setQueriesOpen: Dispatch<SetStateAction<boolean>>;
  savedQueries: SavedQuery[];
  activeQuery: string;
  runQuery: (q: SavedQuery) => void;
  deleteQuery: (id: number) => Promise<void>;
  setSaveQueryOpen: Dispatch<SetStateAction<boolean>>;
  rsvpPickerOpen: boolean;
  setRsvpPickerOpen: Dispatch<SetStateAction<boolean>>;
  detail: MessageDetail | null;
  invite: Invite | null;
  rsvpBusy: string;
  respondInvite: (id: string, status: "accepted" | "tentative" | "declined") => Promise<void>;
  saveQueryOpen: boolean;
  saveQueryName: string;
  setSaveQueryName: Dispatch<SetStateAction<string>>;
  doSaveQuery: () => void;
  slackForwardOpen: boolean;
  setSlackForwardOpen: Dispatch<SetStateAction<boolean>>;
  forwardSlack: (id: string, channelID: string, userMessage: string) => void;
}) {
  const {
    compose, setCompose, showToast, draftsView, loadDrafts,
    labelsFor, setLabelsFor, applyLabelChange, bulkLabels, setBulkLabels, selected,
    promptsOpen, setPromptsOpen, runPrompt, aiPromptsEnabled, setPromptManagerOpen,
    promptManagerOpen, aiEnabled, aiCache, setPromptResult,
    linksFor, setLinksFor,
    suggestFor, setSuggestFor, suggestions, loadingSuggest, applySuggestion,
    attachmentsOpen, setAttachmentsOpen, attachments, busy, downloadAttachment,
    queriesOpen, setQueriesOpen, savedQueries, activeQuery, runQuery, deleteQuery, setSaveQueryOpen,
    rsvpPickerOpen, setRsvpPickerOpen, detail, invite, rsvpBusy, respondInvite,
    saveQueryOpen, saveQueryName, setSaveQueryName, doSaveQuery,
    slackForwardOpen, setSlackForwardOpen, forwardSlack,
  } = p;
  return (
    <>
      {compose && (
        <Compose
          init={compose}
          onClose={() => setCompose(null)}
          onSent={(msg) => {
            setCompose(null);
            showToast(msg);
            if (draftsView) void loadDrafts();
          }}
        />
      )}
      {labelsFor && (
        <LabelsPicker
          messageId={labelsFor}
          onClose={() => setLabelsFor(null)}
          onChanged={(c) => applyLabelChange(new Set([labelsFor]), c)}
        />
      )}
      {bulkLabels && (
        <LabelsPicker
          bulkIds={[...selected]}
          onClose={() => setBulkLabels(false)}
          onChanged={(c) => applyLabelChange(new Set(selected), c)}
        />
      )}
      {promptsOpen && (
        <PromptsPicker
          onClose={() => setPromptsOpen(false)}
          onPick={(prompt) => void runPrompt(prompt)}
          onManage={
            aiPromptsEnabled
              ? () => {
                  setPromptsOpen(false);
                  setPromptManagerOpen(true);
                }
              : undefined
          }
        />
      )}
      {promptManagerOpen && (
        <PromptManager
          aiEnabled={aiEnabled}
          onClose={() => setPromptManagerOpen(false)}
          onChanged={() => {
            // A prompt was created/edited/deleted. Drop cached prompt results so a
            // re-run regenerates with the new text (the backend already cleared
            // the DB copies for edited/deleted prompts).
            for (const e of aiCache.current.values()) {
              e.promptResults = {};
              e.lastPromptId = undefined;
            }
            setPromptResult(null);
          }}
        />
      )}
      {linksFor && (
        <LinksPicker messageId={linksFor} onClose={() => setLinksFor(null)} />
      )}
      {suggestFor && (
        <SuggestPicker
          suggestions={suggestions}
          loading={loadingSuggest}
          onApply={applySuggestion}
          onClose={() => setSuggestFor(null)}
        />
      )}
      {attachmentsOpen && (
        <AttachmentsPicker
          attachments={attachments}
          busy={busy}
          onDownload={(att) => void downloadAttachment(att)}
          onClose={() => setAttachmentsOpen(false)}
        />
      )}
      {queriesOpen && (
        <SavedQueriesPicker
          queries={savedQueries}
          canSaveCurrent={!!activeQuery}
          onRun={runQuery}
          onDelete={(id) => void deleteQuery(id)}
          onSaveCurrent={() => {
            setQueriesOpen(false);
            setSaveQueryOpen(true);
          }}
          onClose={() => setQueriesOpen(false)}
        />
      )}
      {rsvpPickerOpen && detail && invite?.isInvite && (
        <RSVPPicker
          summary={invite.summary || ""}
          when={invite.dtStart ? formatICSDate(invite.dtStart) : ""}
          busy={rsvpBusy}
          onRespond={(status) => {
            void respondInvite(detail.id, status);
            setRsvpPickerOpen(false);
          }}
          onClose={() => setRsvpPickerOpen(false)}
        />
      )}
      {slackForwardOpen && detail && (
        <SlackPicker
          onSend={(channelID, message) => forwardSlack(detail.id, channelID, message)}
          onClose={() => setSlackForwardOpen(false)}
        />
      )}
      {saveQueryOpen && (
        <SaveQueryModal
          name={saveQueryName}
          onNameChange={setSaveQueryName}
          query={activeQuery}
          onSave={doSaveQuery}
          onClose={() => setSaveQueryOpen(false)}
        />
      )}
    </>
  );
}

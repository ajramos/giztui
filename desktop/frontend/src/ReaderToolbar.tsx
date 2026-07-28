import type { MessageDetail } from "./api";
import { Icon, IconBtn } from "./Icons";
import MoreMenu from "./MoreMenu";

// The reader action bar: primary actions (reply/forward/labels/archive/trash/
// read/html/thread) plus the "⋯" overflow menu. Presentational — App owns all
// state and passes plain onX handlers, so the AI/obsidian/slack coupling and
// the aiCache landmine stay in App. Behavior-preserving extraction of the
// `<div className="actions">` block.
export default function ReaderToolbar({
  detail,
  busy,
  aiEnabled,
  aiPromptsEnabled,
  obsidianOn,
  slackOn,
  threadingOn,
  hasThread,
  viewHtml,
  summarizing,
  promptRunning,
  generatingReply,
  touchingUp,
  touchUpShown,
  headersHidden,
  headersExpanded,
  onReply,
  onForward,
  onLabels,
  onArchive,
  onTrash,
  onToggleRead,
  onToggleHtml,
  onToggleThread,
  onSummarize,
  onApplyPrompt,
  onDraftReply,
  onTouchUp,
  onSuggestLabels,
  onMove,
  onSearchSender,
  onLinks,
  onObsidian,
  onSlack,
  onSave,
  onSaveRaw,
  onToggleHeaderBlock,
  onToggleFullHeaders,
  onOpenGmail,
}: {
  detail: MessageDetail;
  busy: boolean;
  aiEnabled: boolean;
  aiPromptsEnabled: boolean;
  obsidianOn: boolean;
  slackOn: boolean;
  threadingOn: boolean;
  hasThread: boolean;
  viewHtml: boolean;
  summarizing: boolean;
  promptRunning: boolean;
  generatingReply: boolean;
  touchingUp: boolean;
  touchUpShown: boolean;
  headersHidden: boolean;
  headersExpanded: boolean;
  onReply: () => void;
  onForward: () => void;
  onLabels: () => void;
  onArchive: () => void;
  onTrash: () => void;
  onToggleRead: () => void;
  onToggleHtml: () => void;
  onToggleThread: () => void;
  onSummarize: () => void;
  onApplyPrompt: () => void;
  onDraftReply: () => void;
  onTouchUp: () => void;
  onSuggestLabels: () => void;
  onMove: () => void;
  onSearchSender: () => void;
  onLinks: () => void;
  onObsidian: () => void;
  onSlack: () => void;
  onSave: () => void;
  onSaveRaw: () => void;
  onToggleHeaderBlock: () => void;
  onToggleFullHeaders: () => void;
  onOpenGmail: () => void;
}) {
  return (
    <div className="actions">
      {/* Primary actions stay visible; everything else collapses into the "⋯"
          overflow so the toolbar never wraps. The whole bar is optional
          (keyboard-first) — hide it from the topbar ▤ toggle or :toolbar. */}
      <IconBtn icon={Icon.reply} label="Reply" primary onClick={onReply} />
      <IconBtn icon={Icon.forward} label="Forward" onClick={onForward} />
      <IconBtn icon={Icon.label} label="Labels" onClick={onLabels} />
      <span className="actions-sep" />
      <IconBtn
        icon={Icon.archive}
        label="Archive"
        disabled={busy}
        onClick={onArchive}
      />
      <IconBtn
        icon={Icon.trash}
        label="Trash"
        danger
        disabled={busy}
        onClick={onTrash}
      />
      <IconBtn
        icon={detail.unread ? Icon.mailOpen : Icon.mail}
        label={detail.unread ? "Mark read" : "Mark unread"}
        disabled={busy}
        onClick={onToggleRead}
      />
      {detail.html && detail.html.trim() && (
        <IconBtn
          icon={viewHtml ? Icon.text : Icon.code}
          label={viewHtml ? "Show plain text" : "Show HTML"}
          onClick={onToggleHtml}
        />
      )}
      {threadingOn && (
        <IconBtn
          icon={Icon.thread}
          label={hasThread ? "Hide conversation" : "Show conversation"}
          primary={hasThread}
          onClick={onToggleThread}
        />
      )}
      <span className="actions-sep" />
      <MoreMenu
        items={[
          {
            icon: Icon.summarize,
            label: summarizing ? "Summarizing…" : "Summarize (AI)",
            disabled: summarizing,
            hidden: !aiEnabled,
            onClick: onSummarize,
          },
          {
            icon: Icon.prompt,
            label: promptRunning ? "Running…" : "Apply a prompt",
            disabled: promptRunning,
            hidden: !aiPromptsEnabled,
            onClick: onApplyPrompt,
          },
          {
            icon: Icon.reply,
            label: generatingReply ? "Drafting…" : "Draft reply (AI)",
            disabled: generatingReply,
            hidden: !aiEnabled,
            onClick: onDraftReply,
          },
          {
            icon: Icon.summarize,
            label: touchingUp
              ? "Reformatting…"
              : touchUpShown
                ? "Show original"
                : "Touch-up (AI)",
            disabled: touchingUp,
            hidden: !aiEnabled,
            onClick: onTouchUp,
          },
          {
            icon: Icon.tag2,
            label: "Suggest labels (AI)",
            hidden: !aiEnabled,
            onClick: onSuggestLabels,
          },
          {
            icon: Icon.folder,
            label: "Move to…",
            onClick: onMove,
          },
          {
            icon: Icon.search,
            label: "Search from sender",
            onClick: onSearchSender,
          },
          {
            icon: Icon.link,
            label: "Links",
            onClick: onLinks,
          },
          {
            icon: Icon.obsidian,
            label: "Send to Obsidian",
            hidden: !obsidianOn,
            onClick: onObsidian,
          },
          {
            icon: Icon.slack,
            label: "Forward to Slack",
            hidden: !slackOn,
            onClick: onSlack,
          },
          {
            icon: Icon.save,
            label: "Save to file",
            onClick: onSave,
          },
          {
            icon: Icon.save,
            label: "Save raw (.eml)",
            onClick: onSaveRaw,
          },
          {
            icon: Icon.text,
            label: headersHidden
              ? "Show header block (h)"
              : "Hide header block (h)",
            onClick: onToggleHeaderBlock,
          },
          {
            icon: Icon.text,
            label: headersExpanded ? "Hide full headers" : "Show full headers",
            onClick: onToggleFullHeaders,
          },
          {
            icon: Icon.external,
            label: "Open in Gmail",
            onClick: onOpenGmail,
          },
        ]}
      />
    </div>
  );
}

import type { Dispatch, RefObject, SetStateAction } from "react";
import type {
  ActionPlanResult,
  PlanCategory,
  MessageSummary,
  MessageDetail,
  Label,
  UsageStats,
  TelemetrySummary,
  ConfigInfo,
  KeyMap,
} from "./apiTypes";
import type { PlanNode } from "./planNodes";
import type { MoveTarget } from "./planMove";
import { buildMoveTargets } from "./planMove";
import type { AdvFilters } from "./advancedSearch";
import { COMMANDS } from "./commands";
import { Icon } from "./Icons";
import { displayName, emailAddr, formatFull } from "./format";
import Markdown from "./Markdown";
import ActionPlanModal from "./ActionPlanModal";
import PlanMovePicker from "./PlanMovePicker";
import RulesManager from "./RulesManager";
import AnalyzerRulesModal from "./AnalyzerRulesModal";
import PromptPreviewModal from "./PromptPreviewModal";
import CommandBar from "./CommandBar";
import ThemePicker from "./ThemePicker";
import MovePicker from "./MovePicker";
import AdvancedSearchModal from "./AdvancedSearchModal";
import StatsModal from "./StatsModal";
import TelemetryModal from "./TelemetryModal";
import ConfigModal from "./ConfigModal";
import Help from "./Help";
import AiJobsPicker from "./AiJobsPicker";
import type { AiJob } from "./useAiJobs";

// The dialog-style modals: bulk-prompt result, action plan (+ move/preview),
// analyzer/deterministic rules, prompt preview, command palette, theme/move
// pickers, advanced search, stats, config, help. A behavior-preserving lift of
// the second half of App's modal stack; every value/handler stays in App and is
// passed through, so the JSX is byte-identical to what lived inline.
export default function ModalsSecondary(p: {
  bulkPromptText: string | null;
  setBulkPromptText: Dispatch<SetStateAction<string | null>>;
  bulkPromptLabel: string;
  bulkJobRunning: boolean;
  jobs: AiJob[];
  jobsPickerOpen: boolean;
  setJobsPickerOpen: Dispatch<SetStateAction<boolean>>;
  openJob: (id: string) => void;
  removeJob: (id: string) => void;
  clearFinished: () => void;
  planOpen: boolean;
  analyzing: boolean;
  analyzeCount: number;
  analyzeProgress: { done: number; total: number } | null;
  analyzeElapsed: number;
  plan: ActionPlanResult | null;
  planNodes: PlanNode[];
  planActiveNode: PlanNode | undefined;
  planNav: { listRef: RefObject<HTMLDivElement>; setActiveHover: (i: number) => void };
  expandedCats: Set<string>;
  setExpandedCats: Dispatch<SetStateAction<Set<string>>>;
  planExcluded: Set<string>;
  setPlanExcluded: Dispatch<SetStateAction<Set<string>>>;
  applyingAll: boolean;
  rulesEnabled: boolean;
  messages: MessageSummary[];
  applyCategory: (cat: PlanCategory, asMove?: boolean) => Promise<void>;
  dispatchPromptCategory: (cat: PlanCategory) => Promise<void>;
  applyAllCategories: () => Promise<void>;
  setPlanOpen: Dispatch<SetStateAction<boolean>>;
  openMessage: (m: MessageSummary) => void;
  openRules: () => Promise<void>;
  viewAnalyzerPrompt: () => Promise<void>;
  planMove: { kind: "email" | "category"; catIdx: number; id?: string } | null;
  setPlanMove: Dispatch<SetStateAction<{ kind: "email" | "category"; catIdx: number; id?: string } | null>>;
  doPlanMove: (target: MoveTarget) => void;
  planPreview: MessageDetail | null;
  planPreviewLoading: boolean;
  setPlanPreview: Dispatch<SetStateAction<MessageDetail | null>>;
  detRulesOpen: boolean;
  setDetRulesOpen: Dispatch<SetStateAction<boolean>>;
  runDeterministicRules: () => Promise<void>;
  rulesOpen: boolean;
  rules: { id: number; text: string }[];
  newRule: string;
  setNewRule: Dispatch<SetStateAction<string>>;
  addRule: () => Promise<void>;
  deleteRule: (id: number) => Promise<void>;
  setRulesOpen: Dispatch<SetStateAction<boolean>>;
  promptPreview: string | null;
  setPromptPreview: Dispatch<SetStateAction<string | null>>;
  cmdOpen: boolean;
  executeCommand: (raw: string) => void;
  setCmdOpen: Dispatch<SetStateAction<boolean>>;
  themePickerOpen: boolean;
  themeNames: string[];
  currentTheme: string;
  applyTheme: (name: string) => Promise<void>;
  setThemePickerOpen: (v: boolean) => void;
  showToast: (m: string) => void;
  moveFor: string | null;
  labels: Label[];
  doMove: (id: string, name: string) => Promise<void>;
  setMoveFor: Dispatch<SetStateAction<string | null>>;
  bulkMove: boolean;
  selected: Set<string>;
  doBulkMove: (name: string) => Promise<void>;
  setBulkMove: Dispatch<SetStateAction<boolean>>;
  advOpen: boolean;
  adv: AdvFilters;
  setAdv: Dispatch<SetStateAction<AdvFilters>>;
  setLocalFilter: Dispatch<SetStateAction<boolean>>;
  setQuery: Dispatch<SetStateAction<string>>;
  load: (q: string) => Promise<void>;
  setAdvOpen: Dispatch<SetStateAction<boolean>>;
  statsOpen: boolean;
  stats: UsageStats | null;
  setStatsOpen: Dispatch<SetStateAction<boolean>>;
  telemetryOpen: boolean;
  telemetry: TelemetrySummary | null;
  setTelemetryOpen: Dispatch<SetStateAction<boolean>>;
  resetTelemetry: () => Promise<void>;
  configOpen: boolean;
  configInfo: ConfigInfo | null;
  clearCaches: () => Promise<void>;
  setConfigOpen: Dispatch<SetStateAction<boolean>>;
  showHelp: boolean;
  keymap: KeyMap;
  aiEnabled: boolean;
  aiPromptsEnabled: boolean;
  obsidianOn: boolean;
  slackOn: boolean;
  threadingOn: boolean;
  savedQueriesOn: boolean;
  actionPlanOn: boolean;
  rsvpEnabled: boolean;
  themesOn: boolean;
  appVersion: string;
  setShowHelp: Dispatch<SetStateAction<boolean>>;
}) {
  const {
    bulkPromptText, setBulkPromptText, bulkPromptLabel, bulkJobRunning,
    jobs, jobsPickerOpen, setJobsPickerOpen, openJob, removeJob, clearFinished,
    planOpen, analyzing, analyzeCount, analyzeProgress, analyzeElapsed, plan,
    planNodes, planActiveNode, planNav, expandedCats, setExpandedCats,
    planExcluded, setPlanExcluded, applyingAll, rulesEnabled, messages,
    applyCategory, dispatchPromptCategory, applyAllCategories, setPlanOpen,
    openMessage, openRules, viewAnalyzerPrompt, planMove, setPlanMove, doPlanMove,
    planPreview, planPreviewLoading, setPlanPreview, detRulesOpen, setDetRulesOpen,
    runDeterministicRules, rulesOpen, rules, newRule, setNewRule, addRule,
    deleteRule, setRulesOpen, promptPreview, setPromptPreview, cmdOpen,
    executeCommand, setCmdOpen, themePickerOpen, themeNames, currentTheme,
    applyTheme, setThemePickerOpen, showToast, moveFor, labels, doMove, setMoveFor,
    bulkMove, selected, doBulkMove, setBulkMove, advOpen, adv, setAdv,
    setLocalFilter, setQuery, load, setAdvOpen, statsOpen, stats, setStatsOpen,
    telemetryOpen, telemetry, setTelemetryOpen, resetTelemetry,
    configOpen, configInfo, clearCaches, setConfigOpen, showHelp, keymap, aiEnabled,
    aiPromptsEnabled, obsidianOn, slackOn, threadingOn, savedQueriesOn,
    actionPlanOn, rsvpEnabled, themesOn, appVersion, setShowHelp,
  } = p;
  return (
    <>
      {jobsPickerOpen && (
        <AiJobsPicker
          jobs={jobs}
          onClose={() => setJobsPickerOpen(false)}
          onOpen={openJob}
          onRemove={removeJob}
          onClear={clearFinished}
        />
      )}
      {bulkPromptText !== null && (
        <div className="modal-overlay" onClick={() => setBulkPromptText(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-head">
              <h3>✦ {bulkPromptLabel}</h3>
              <button className="ghost" onClick={() => setBulkPromptText(null)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              {bulkJobRunning && !bulkPromptText ? (
                <div className="placeholder">Generating…</div>
              ) : (
                <div className="summary-text">
                  <Markdown text={bulkPromptText || ""} />
                  {bulkJobRunning && <span className="caret">▍</span>}
                </div>
              )}
            </div>
            <div className="modal-foot">
              <button onClick={() => setBulkPromptText(null)}>Done</button>
            </div>
          </div>
        </div>
      )}
      {planOpen && (
        <ActionPlanModal
          analyzing={analyzing}
          analyzeCount={analyzeCount}
          analyzeProgress={analyzeProgress}
          analyzeElapsed={analyzeElapsed}
          plan={plan}
          planNodes={planNodes}
          planActiveNode={planActiveNode}
          listRef={planNav.listRef}
          setActiveHover={planNav.setActiveHover}
          expandedCats={expandedCats}
          setExpandedCats={setExpandedCats}
          planExcluded={planExcluded}
          setPlanExcluded={setPlanExcluded}
          applyingAll={applyingAll}
          rulesEnabled={rulesEnabled}
          messages={messages}
          onApplyCategory={(c, move) => void applyCategory(c, move)}
          onDispatchPrompt={(c) => void dispatchPromptCategory(c)}
          onApplyAll={() => void applyAllCategories()}
          onOpenMessage={(m) => {
            setPlanOpen(false);
            void openMessage(m);
          }}
          onOpenRules={() => void openRules()}
          onViewPrompt={() => void viewAnalyzerPrompt()}
          onClose={() => setPlanOpen(false)}
        />
      )}
      {planMove && plan && (
        <PlanMovePicker
          title={
            planMove.kind === "email"
              ? "Move email to…"
              : `Move "${plan.categories[planMove.catIdx]?.name ?? ""}" (${
                  // The selected (non-deselected) count — what will actually move.
                  (plan.categories[planMove.catIdx]?.messageIds ?? []).filter(
                    (id) => !planExcluded.has(id),
                  ).length
                }) to…`
          }
          targets={buildMoveTargets(
            plan.categories,
            plan.categories[planMove.catIdx]?.name ?? "",
          )}
          onChoose={doPlanMove}
          onClose={() => setPlanMove(null)}
        />
      )}
      {(planPreview || planPreviewLoading) && (
        <div className="modal-overlay" onClick={() => setPlanPreview(null)}>
          <div
            className="modal plan-preview"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="modal-head">
              <h3>{planPreview?.subject || "Quick view"}</h3>
              <button className="ghost" onClick={() => setPlanPreview(null)}>
                {Icon.x}
              </button>
            </div>
            <div className="modal-body">
              {planPreviewLoading || !planPreview ? (
                <div className="placeholder">Loading…</div>
              ) : (
                <>
                  <div className="qv-meta muted">
                    <div>
                      <strong>{displayName(planPreview.from)}</strong>{" "}
                      {emailAddr(planPreview.from)}
                    </div>
                    <div>{formatFull(planPreview.date)}</div>
                  </div>
                  <pre className="qv-body">
                    {planPreview.plainText?.trim() ||
                      "(no plain-text preview — open in reader for the full HTML)"}
                  </pre>
                </>
              )}
            </div>
            <div className="modal-foot">
              <span className="foot-hint">Esc back to plan</span>
              {planPreview && (
                <button
                  className="ghost"
                  onClick={() => {
                    const m = messages.find((x) => x.id === planPreview.id);
                    setPlanPreview(null);
                    setPlanOpen(false);
                    if (m) void openMessage(m);
                  }}
                >
                  Open in reader
                </button>
              )}
            </div>
          </div>
        </div>
      )}
      {detRulesOpen && (
        <RulesManager
          onClose={() => setDetRulesOpen(false)}
          onRun={() => void runDeterministicRules()}
        />
      )}
      {rulesOpen && (
        <AnalyzerRulesModal
          rules={rules}
          newRule={newRule}
          onNewRuleChange={setNewRule}
          onAddRule={() => void addRule()}
          onDeleteRule={(id) => void deleteRule(id)}
          onClose={() => setRulesOpen(false)}
        />
      )}
      {promptPreview !== null && (
        <PromptPreviewModal
          text={promptPreview}
          onClose={() => setPromptPreview(null)}
        />
      )}
      {cmdOpen && (
        <CommandBar
          commands={COMMANDS}
          onRun={executeCommand}
          onClose={() => setCmdOpen(false)}
        />
      )}
      {themePickerOpen && (
        <ThemePicker
          themes={themeNames}
          current={currentTheme}
          onPick={(name) => {
            void applyTheme(name);
            setThemePickerOpen(false);
            showToast(`Theme: ${name}`);
          }}
          onClose={() => setThemePickerOpen(false)}
        />
      )}
      {moveFor && (
        <MovePicker
          labels={labels}
          onMove={(name) => void doMove(moveFor, name)}
          onClose={() => setMoveFor(null)}
        />
      )}
      {bulkMove && (
        <MovePicker
          labels={labels}
          count={selected.size}
          onMove={(name) => void doBulkMove(name)}
          onClose={() => setBulkMove(false)}
        />
      )}
      {advOpen && (
        <AdvancedSearchModal
          adv={adv}
          onChange={setAdv}
          onSearch={(q) => {
            if (!q) return;
            setLocalFilter(false);
            setQuery(q);
            setAdvOpen(false);
            void load(q);
          }}
          onClose={() => setAdvOpen(false)}
        />
      )}
      {statsOpen && (
        <StatsModal stats={stats} onClose={() => setStatsOpen(false)} />
      )}
      {telemetryOpen && (
        <TelemetryModal
          summary={telemetry}
          onReset={() => void resetTelemetry()}
          onClose={() => setTelemetryOpen(false)}
        />
      )}
      {configOpen && (
        <ConfigModal
          info={configInfo}
          onClearCaches={() => void clearCaches()}
          onClose={() => setConfigOpen(false)}
        />
      )}
      {showHelp && (
        <Help
          keymap={keymap}
          flags={{
            ai: aiEnabled,
            prompts: aiPromptsEnabled,
            obsidian: obsidianOn,
            slack: slackOn,
            threading: threadingOn,
            savedQueries: savedQueriesOn,
            actionPlan: actionPlanOn,
            rsvp: rsvpEnabled,
            themes: themesOn,
            rules: rulesEnabled,
          }}
          version={appVersion}
          onClose={() => setShowHelp(false)}
        />
      )}

    </>
  );
}

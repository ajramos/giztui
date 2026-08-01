import { useCallback, useEffect } from "react";
import { backend } from "./api";
import type { AccountInfo, KeyMap, Label, MessageDetail } from "./apiTypes";

// useBootstrap owns app startup + account switching: runInit (wait for the
// backend session, detect missing credentials, load config flags/keymap/labels/
// theme, then the inbox), the credentials import/retry flow, and switchAccount
// (reset per-account UI state then reload). Pure setState orchestration lifted
// verbatim from App; all the cross-subsystem setters arrive via deps.
export function useBootstrap(deps: {
  load: (q: string) => Promise<void>;
  initTheme: () => Promise<void>;
  refreshIntegrations: () => Promise<void>;
  setConnecting: (v: boolean) => void;
  setInitError: (v: string) => void;
  setNeedCreds: (v: boolean) => void;
  setAuthUrl: (v: string) => void;
  setCredsPath: (v: string) => void;
  setError: (e: string) => void;
  setAccount: (v: string) => void;
  setAiEnabled: (v: boolean) => void;
  setAiPromptsEnabled: (v: boolean) => void;
  setJobsNotify: (v: boolean) => void;
  setAccounts: (v: AccountInfo[]) => void;
  setKeymap: (v: KeyMap) => void;
  setAppVersion: (v: string) => void;
  setThreadingOn: (v: boolean) => void;
  setShowNumbers: (v: boolean) => void;
  setSavedQueriesOn: (v: boolean) => void;
  setActionPlanOn: (v: boolean) => void;
  setRulesEnabled: (v: boolean) => void;
  setLabels: (v: Label[]) => void;
  setRsvpEnabled: (v: boolean) => void;
  setAutoRefreshSecs: (v: number) => void;
  setAutoRefresh: (v: boolean) => void;
  setImportErr: (v: string) => void;
  setImporting: (v: boolean) => void;
  setSwitching: (v: boolean) => void;
  setSelectedId: (v: string | null) => void;
  setDetail: (v: MessageDetail | null) => void;
  setSummary: (v: string | null) => void;
  setPromptResult: (v: string | null) => void;
  setBulkMode: (v: boolean) => void;
  setSelected: (v: Set<string>) => void;
  setQuery: (v: string) => void;
}) {
  const {
    load, initTheme, refreshIntegrations, setConnecting, setInitError, setNeedCreds,
    setAuthUrl, setCredsPath, setError, setAccount, setAiEnabled, setAiPromptsEnabled, setJobsNotify,
    setAccounts, setKeymap, setAppVersion, setThreadingOn, setShowNumbers, setSavedQueriesOn, setActionPlanOn,
    setRulesEnabled, setLabels, setRsvpEnabled, setAutoRefreshSecs, setAutoRefresh, setImportErr,
    setImporting, setSwitching, setSelectedId, setDetail, setSummary, setPromptResult,
    setBulkMode, setSelected, setQuery,
  } = deps;
  const runInit = useCallback(async () => {
    // Reset to the connecting state (also covers retry-after-import).
    setConnecting(true);
    setInitError("");
    setNeedCreds(false);
    // The backend builds the Gmail/service session off the main thread so the
    // window paints immediately; wait for it to be ready before the first
    // calls (up to ~45s for a cold OAuth) instead of erroring.
    for (let i = 0; i < 300; i++) {
      try {
        if (await backend.Ready()) break;
      } catch {
        break; // mock / no backend
      }
      // Surface the sign-in URL (first-run OAuth) so the modal can offer a
      // button instead of the user hunting for the URL in the logs.
      try {
        setAuthUrl(await backend.PendingAuthURL());
      } catch {
        /* mock backend has no pending auth */
      }
      await new Promise((r) => setTimeout(r, 150));
    }
    setAuthUrl("");
    setConnecting(false);
    try {
      const ie = await backend.InitError();
      if (ie) {
        // Distinguish "no credentials.json yet" (first-run) from other errors so
        // the UI can offer an import flow instead of a dead-end message.
        try {
          if (await backend.NeedsCredentials()) {
            setNeedCreds(true);
            setCredsPath(await backend.CredentialsPath());
          }
        } catch {
          /* older/mock backend without these methods */
        }
        setInitError(ie);
        return;
      }
    } catch {
      /* mock backend never errors here */
    }
      try {
        setAccount(await backend.AccountEmail());
      } catch {
        /* non-fatal */
      }
      try {
        setAiEnabled(await backend.AIEnabled());
      } catch {
        /* non-fatal */
      }
      try {
        setAiPromptsEnabled(await backend.PromptsEnabled());
      } catch {
        /* non-fatal */
      }
      try {
        setJobsNotify(await backend.JobsNotifyOnComplete());
      } catch {
        /* non-fatal */
      }
      try {
        setAccounts(await backend.ListAccounts());
      } catch {
        /* non-fatal */
      }
      try {
        setKeymap(await backend.KeyMap());
      } catch {
        /* non-fatal — defaults already set */
      }
      try {
        setAppVersion(await backend.Version());
      } catch {
        /* non-fatal */
      }
      try {
        await refreshIntegrations();
        setThreadingOn(await backend.ThreadingEnabled());
        setShowNumbers(await backend.ShowMessageNumbers());
        setSavedQueriesOn(await backend.SavedQueriesEnabled());
        setActionPlanOn(await backend.ActionPlanEnabled());
        setRulesEnabled(await backend.AnalyzerRulesEnabled());
      } catch {
        /* non-fatal */
      }
      await initTheme();
      try {
        setLabels(await backend.ListLabels());
      } catch {
        /* non-fatal */
      }
      try {
        setRsvpEnabled(await backend.RSVPEnabled());
      } catch {
        /* non-fatal */
      }
      try {
        const ar = await backend.AutoRefreshSettings();
        if (ar.intervalSeconds > 0) setAutoRefreshSecs(ar.intervalSeconds);
        // config.json is the single source of truth (the toggle persists back to
        // it). A stale localStorage override used to win here, which made
        // auto_refresh.enabled:true show as OFF after the user had toggled it once.
        setAutoRefresh(ar.enabled);
      } catch {
        /* non-fatal */
      }
      void load("");
  }, [load, initTheme]);

  useEffect(() => {
    void runInit();
  }, [runInit]);

  // Import a credentials.json via the native file picker, then retry init.
  const importCreds = useCallback(async () => {
    setImportErr("");
    setImporting(true);
    try {
      await backend.ImportCredentials();
      // ImportCredentials returns "" if the user cancelled the dialog; in that
      // case NeedsCredentials stays true and runInit just shows the screen again.
      await runInit();
    } catch (e) {
      setImportErr(String((e as Error)?.message ?? e));
    } finally {
      setImporting(false);
    }
  }, [runInit]);

  // Re-run init after the user placed credentials.json manually.
  const retryInit = useCallback(async () => {
    setImportErr("");
    try {
      await backend.RetryInit();
    } catch {
      /* mock backend */
    }
    await runInit();
  }, [runInit]);

  const switchAccount = useCallback(
    async (a: AccountInfo) => {
      setSwitching(true);
      setError("");
      try {
        await backend.SwitchAccount(a.id);
        setSelectedId(null);
        setDetail(null);
        setSummary(null);
        setPromptResult(null);
        setBulkMode(false);
        setSelected(new Set());
        setQuery("");
        const [email, ai, prompts, notify, accs] = await Promise.all([
          backend.AccountEmail().catch(() => ""),
          backend.AIEnabled().catch(() => false),
          backend.PromptsEnabled().catch(() => false),
          backend.JobsNotifyOnComplete().catch(() => true),
          backend.ListAccounts().catch(() => [] as AccountInfo[]),
        ]);
        setAccount(email);
        setAiEnabled(ai);
        setAiPromptsEnabled(prompts);
        setJobsNotify(notify);
        if (accs.length) setAccounts(accs);
        await load("");
      } catch (e) {
        setError(String(e));
      } finally {
        setSwitching(false);
      }
    },
    [load],
  );

  return { importCreds, retryInit, switchAccount };
}

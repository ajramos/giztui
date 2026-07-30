import { matchesCombo, countMatches } from "./format";
import { handleKeyMain } from "./keydownMain";
import type { KeydownCtx } from "./keydownCtx";

// The keyboard handler prefix: UI zoom, advanced-search, the Ctrl/Cmd guard,
// help, the layered-Escape modal chain, action-plan shortcuts, the typing/
// drafts guards, the accounts toggle, and content-search n/N. Falls through to
// handleKeyMain for navigation + the chord/VIM dispatch. Verbatim from App.
export function handleKeyDown(e: KeyboardEvent, ctx: KeydownCtx) {
  const {
    accounts, accountsOpen, advOpen, applyCategory, bulkLabels, bulkMove,
    bulkPromptText, bumpZoom, cmdOpen, compose, configOpen, csOpen,
    csQuery, detRulesOpen, detail, draftsView, fullMessagesRef, keymap,
    labelsFor, linksFor, load, localFilter, moveFor, openRules,
    plan, planActiveRef, planMove, planNodesRef, planOpen, planPreview,
    promptManagerOpen, promptPreview, promptsOpen, queriesOpen, resetZoom, rsvpPickerOpen,
    rulesEnabled, rulesOpen, saveQueryOpen, searchRef, setAccountsOpen, setAdvOpen,
    setAttachmentsOpen, setBulkLabels, setBulkMove, setBulkPromptText, setCmdOpen, setCompose,
    setConfigOpen, setCsIndex, setDraftsView, setExpandedCats, setLabelsFor, setLinksFor,
    setLocalFilter, setMessages, setMoveFor, setPlanExcluded, setPlanMove, setPlanOpen,
    setPlanPreview, setPromptManagerOpen, setPromptPreview, setPromptsOpen, setQueriesOpen, setQuery,
    setRsvpPickerOpen, setRulesOpen, setSaveQueryOpen, setShowHelp, setStatsOpen, setSuggestFor,
    setThemePickerOpen, showHelp, statsOpen, suggestFor, themePickerOpen, viewAnalyzerPrompt,
    attachmentsOpen, activeQuery, jobsPickerOpen, setJobsPickerOpen,
    slackForwardOpen, setSlackForwardOpen,
  } = ctx;
      const tag = (e.target as HTMLElement | null)?.tagName;
      const typing = tag === "INPUT" || tag === "TEXTAREA";
      const chord = e.key === " " ? "space" : e.key;

      // UI zoom (Cmd/Ctrl +/-/0) — handled first so it works everywhere, even
      // over modals or while typing. WKWebView ignores native zoom.
      if ((e.metaKey || e.ctrlKey) && !e.altKey) {
        if (e.key === "=" || e.key === "+") {
          e.preventDefault();
          bumpZoom(0.1);
          return;
        }
        if (e.key === "-" || e.key === "_") {
          e.preventDefault();
          bumpZoom(-0.1);
          return;
        }
        if (e.key === "0") {
          e.preventDefault();
          resetZoom();
          return;
        }
      }

      // Advanced search builder (default Ctrl+F / Cmd+F, from search_advanced).
      // Handled before the generic Ctrl/Cmd early-return and the typing guard so
      // it opens from anywhere — list, reader, or the search box — like the TUI.
      // Only modifier combos are honored here (a bare key would hijack typing).
      if (keymap.searchAdvanced && !advOpen && matchesCombo(e, keymap.searchAdvanced)) {
        e.preventDefault();
        setAdvOpen(true);
        return;
      }

      // Never let OS/browser clipboard & navigation combos (Cmd/Ctrl+C, V, X, A,
      // Z…) fall through to single-key actions like compose ("c"). The only
      // Cmd/Ctrl combos we act on are zoom (handled above) and the accounts
      // switcher (Cmd/Ctrl+A, handled below), so let everything else reach the
      // browser — otherwise selecting text and pressing Cmd+C opened the composer.
      if (e.metaKey || e.ctrlKey) {
        const isAccounts =
          (e.key === "a" || e.key === "A") && accounts.length > 1;
        if (!isAccounts) return;
      }

      if (showHelp) {
        if (e.key === "Escape" || chord === keymap.help) {
          setShowHelp(false);
          e.preventDefault();
        }
        return;
      }
      const anyModal =
        compose ||
        labelsFor ||
        bulkLabels ||
        promptsOpen ||
        promptManagerOpen ||
        linksFor ||
        suggestFor ||
        cmdOpen ||
        queriesOpen ||
        saveQueryOpen ||
        planOpen ||
        themePickerOpen ||
        rulesOpen ||
        promptPreview !== null ||
        advOpen ||
        statsOpen ||
        configOpen ||
        moveFor ||
        bulkMove ||
        rsvpPickerOpen ||
        jobsPickerOpen ||
        slackForwardOpen ||
        detRulesOpen ||
        accountsOpen ||
        attachmentsOpen ||
        bulkPromptText !== null;
      if (anyModal) {
        // Escape closes the topmost modal from the window (WKWebView won't focus
        // a bare div, so per-modal Escape handlers on divs are unreliable). Order
        // = last-opened first, so a sub-modal (e.g. rules over the plan) closes
        // before its parent. Pickers also self-close via their own listener;
        // double-closing is harmless.
        if (e.key === "Escape") {
          e.preventDefault();
          if (accountsOpen) setAccountsOpen(false);
          else if (attachmentsOpen) setAttachmentsOpen(false);
          else if (promptPreview !== null) setPromptPreview(null);
          else if (rulesOpen) setRulesOpen(false);
          else if (bulkPromptText !== null) setBulkPromptText(null);
          else if (saveQueryOpen) setSaveQueryOpen(false);
          else if (moveFor) setMoveFor(null);
          else if (bulkMove) setBulkMove(false);
          else if (suggestFor) setSuggestFor(null);
          else if (advOpen) setAdvOpen(false);
          else if (statsOpen) setStatsOpen(false);
          else if (configOpen) setConfigOpen(false);
          else if (planPreview) setPlanPreview(null);
          else if (planMove) setPlanMove(null);
          else if (planOpen) setPlanOpen(false);
          else if (themePickerOpen) setThemePickerOpen(false);
          else if (queriesOpen) setQueriesOpen(false);
          else if (rsvpPickerOpen) setRsvpPickerOpen(false);
          else if (jobsPickerOpen) setJobsPickerOpen(false);
          else if (slackForwardOpen) setSlackForwardOpen(false);
          else if (linksFor) setLinksFor(null);
          else if (bulkLabels) setBulkLabels(false);
          else if (labelsFor) setLabelsFor(null);
          else if (promptsOpen) setPromptsOpen(false);
          else if (promptManagerOpen) setPromptManagerOpen(false);
          else if (cmdOpen) setCmdOpen(false);
          else if (compose) setCompose(null);
          return;
        }
        // Ctrl/Cmd+A toggles the account menu closed too (it's what opened it),
        // since the menu is now part of the modal guard and swallows other keys.
        if (
          accountsOpen &&
          (e.ctrlKey || e.metaKey) &&
          (e.key === "a" || e.key === "A")
        ) {
          e.preventDefault();
          setAccountsOpen(false);
          return;
        }
        // Action-plan reachable-by-keyboard shortcuts for its header buttons.
        if (
          planOpen &&
          !rulesOpen &&
          !detRulesOpen &&
          promptPreview === null &&
          planMove === null &&
          planPreview === null &&
          bulkPromptText === null
        ) {
          if (e.key === "r") {
            e.preventDefault();
            if (rulesEnabled) void openRules();
            return;
          }
          if (e.key === "p") {
            e.preventDefault();
            void viewAnalyzerPrompt();
            return;
          }
          // l applies a label bucket as LABEL-ONLY (Enter does the move variant).
          if (e.key === "l") {
            const node = planNodesRef.current[planActiveRef.current];
            const c = node ? plan?.categories[node.catIdx] : undefined;
            if (c && c.action === "label") {
              e.preventDefault();
              void applyCategory(c, false);
              return;
            }
          }
          // m reassigns to another bucket: on an email node, that one email; on a
          // category node, the whole category (the TUI's action-plan move).
          if (e.key === "m") {
            const node = planNodesRef.current[planActiveRef.current];
            if (node) {
              e.preventDefault();
              if (node.type === "email") {
                setPlanMove({
                  kind: "email",
                  catIdx: node.catIdx,
                  id: node.id,
                });
              } else {
                setPlanMove({ kind: "category", catIdx: node.catIdx });
              }
              return;
            }
          }
          // Space toggles selection of the active email (deselect to exclude it
          // from the category's apply/move); on a category it expands/collapses.
          if (e.key === " ") {
            const node = planNodesRef.current[planActiveRef.current];
            if (node?.type === "email") {
              e.preventDefault();
              setPlanExcluded((prev) => {
                const n = new Set(prev);
                if (n.has(node.id)) n.delete(node.id);
                else n.add(node.id);
                return n;
              });
              return;
            }
            const cat = node ? plan?.categories[node.catIdx] : undefined;
            if (cat) {
              e.preventDefault();
              setExpandedCats((prev) => {
                const n = new Set(prev);
                if (n.has(cat.name)) n.delete(cat.name);
                else n.add(cat.name);
                return n;
              });
              return;
            }
          }
          // → expands the active category (or the parent of the active email);
          // ← collapses it.
          if (e.key === "ArrowRight" || e.key === "ArrowLeft") {
            const node = planNodesRef.current[planActiveRef.current];
            const cat = node ? plan?.categories[node.catIdx] : undefined;
            if (cat) {
              e.preventDefault();
              setExpandedCats((prev) => {
                const n = new Set(prev);
                if (e.key === "ArrowLeft") n.delete(cat.name);
                else n.add(cat.name);
                return n;
              });
              return;
            }
          }
        }
        return;
      }
      if (typing) {
        if (e.key === "Escape") {
          // preventDefault so macOS/WKWebView doesn't treat Escape as
          // "leave fullscreen"; we handle it ourselves.
          e.preventDefault();
          const el = e.target as HTMLElement;
          if (el === searchRef.current) {
            // TUI parity: Escape in the search box clears the filter and
            // returns to the default inbox, instead of just blurring.
            setQuery("");
            if (localFilter) {
              setLocalFilter(false);
              setMessages(fullMessagesRef.current);
            } else if (activeQuery) {
              void load("");
            }
          }
          el.blur();
        }
        return;
      }
      if (draftsView) {
        if (e.key === "Escape" || chord === keymap.drafts) {
          setDraftsView(false);
          e.preventDefault();
        } else if (chord === keymap.help) {
          setShowHelp(true);
        }
        return;
      }

      // Ctrl/Cmd+A opens the account switcher (TUI's accounts shortcut). Placed
      // after the typing guard so it never hijacks select-all inside inputs.
      if (
        (e.ctrlKey || e.metaKey) &&
        (e.key === "a" || e.key === "A") &&
        accounts.length > 1
      ) {
        e.preventDefault();
        setAccountsOpen((v) => !v);
        return;
      }

      // Content-search match navigation: once a find is active, n = next match,
      // N = previous (the TUI's in-message n/N). Typing in the search box is
      // handled by the typing guard above, so this only fires from the reader.
      if (csOpen && csQuery && detail && (chord === "n" || chord === "N")) {
        e.preventDefault();
        const total = countMatches(detail.plainText || "", csQuery);
        if (total > 0)
          setCsIndex((i) =>
            chord === "n" ? (i + 1) % total : (i - 1 + total) % total,
          );
        return;
      }

  handleKeyMain(e, ctx);
}

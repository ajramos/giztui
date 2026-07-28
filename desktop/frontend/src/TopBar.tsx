import type { RefObject } from "react";
import type { AccountInfo } from "./api";
import { isWails } from "./api";
import { Icon, IconBtn } from "./Icons";
import AccountSwitcher from "./AccountSwitcher";

// The top bar: brand, the search/filter box, the account switcher, and the
// action buttons (undo/compose/drafts/select/saved/toolbar/autorefresh/help/
// refresh). Presentational — App owns all state and passes values + plain
// handlers. Behavior-preserving extraction of the `<header className="topbar">`.
export default function TopBar({
  query,
  localFilter,
  searchRef,
  searchHint,
  activeQuery,
  onQueryChange,
  onSubmitSearch,
  onToggleFilterMode,
  onSearchEscape,
  onAdvanced,
  onClearSearch,
  accounts,
  account,
  switching,
  accountsOpen,
  onAccountsOpenChange,
  onSwitchAccount,
  undoLabel,
  onUndo,
  onCompose,
  draftsView,
  onToggleDrafts,
  bulkMode,
  onToggleBulk,
  savedQueriesOn,
  onOpenQueries,
  showToolbar,
  onToggleToolbar,
  autoRefresh,
  autoRefreshSecs,
  onToggleAutoRefresh,
  onHelp,
  onRefresh,
}: {
  query: string;
  localFilter: boolean;
  searchRef: RefObject<HTMLInputElement>;
  searchHint: string;
  activeQuery: string;
  onQueryChange: (v: string) => void;
  onSubmitSearch: () => void;
  onToggleFilterMode: () => void;
  onSearchEscape: () => void;
  onAdvanced: () => void;
  onClearSearch: () => void;
  accounts: AccountInfo[];
  account: string;
  switching: boolean;
  accountsOpen: boolean;
  onAccountsOpenChange: (open: boolean) => void;
  onSwitchAccount: (a: AccountInfo) => void;
  undoLabel: string;
  onUndo: () => void;
  onCompose: () => void;
  draftsView: boolean;
  onToggleDrafts: () => void;
  bulkMode: boolean;
  onToggleBulk: () => void;
  savedQueriesOn: boolean;
  onOpenQueries: () => void;
  showToolbar: boolean;
  onToggleToolbar: () => void;
  autoRefresh: boolean;
  autoRefreshSecs: number;
  onToggleAutoRefresh: () => void;
  onHelp: () => void;
  onRefresh: () => void;
}) {
  return (
    <header className="topbar">
      <div className="brand">
        <span className="logo">✦</span> GizTUI
        <span className="subtitle">Desktop</span>
      </div>
      <form
        className="searchbox"
        onSubmit={(e) => {
          e.preventDefault();
          onSubmitSearch();
        }}
      >
        <IconBtn
          icon={localFilter ? Icon.filter : Icon.search}
          label={
            localFilter
              ? "Local filter — click for Gmail search"
              : "Gmail search — click to filter loaded list"
          }
          primary={localFilter}
          onClick={onToggleFilterMode}
        />
        <input
          ref={searchRef}
          type="text"
          placeholder={
            localFilter
              ? "Filter loaded messages…"
              : `Search mail (${searchHint} · Ctrl+F advanced) — from:, has:attachment…`
          }
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
          onKeyDown={(e) => {
            // Escape from the search box exits the search entirely (back to the
            // default inbox), matching the TUI — not just a blur.
            if (e.key === "Escape") {
              e.preventDefault();
              onSearchEscape();
              (e.target as HTMLElement).blur();
            }
          }}
        />
        {!localFilter && (
          <button
            type="submit"
            className="icon-btn primary"
            aria-label="Search"
            data-tip="Search"
          >
            {Icon.search}
          </button>
        )}
        <IconBtn
          icon={Icon.sliders}
          label="Advanced search"
          onClick={onAdvanced}
        />
        {(activeQuery || (localFilter && query)) && (
          <IconBtn icon={Icon.x} label="Clear search" onClick={onClearSearch} />
        )}
      </form>
      <div className="account">
        {!isWails() && <span className="badge">mock</span>}
        <AccountSwitcher
          accounts={accounts}
          email={account}
          switching={switching}
          onSwitch={onSwitchAccount}
          open={accountsOpen}
          onOpenChange={onAccountsOpenChange}
        />
        {/* Same IconBtn format as the reader/bulk toolbars for one consistent
            button language across the app. */}
        <div className="actions topbar-actions">
          {undoLabel && (
            <IconBtn
              icon={Icon.undo}
              label={`Undo ${undoLabel} (U)`}
              onClick={onUndo}
            />
          )}
          <IconBtn
            icon={Icon.edit}
            label="Compose (c)"
            primary
            onClick={onCompose}
          />
          <IconBtn
            icon={Icon.drafts}
            label="Drafts (D)"
            primary={draftsView}
            onClick={onToggleDrafts}
          />
          <IconBtn
            icon={Icon.checkAll}
            label="Select mode (v)"
            primary={bulkMode}
            onClick={onToggleBulk}
          />
          {savedQueriesOn && (
            <IconBtn
              icon={Icon.bookmark}
              label="Saved searches (Q)"
              onClick={onOpenQueries}
            />
          )}
          <IconBtn
            icon={Icon.layout}
            label={
              showToolbar
                ? "Hide reader toolbar (:toolbar)"
                : "Show reader toolbar (:toolbar)"
            }
            primary={showToolbar}
            onClick={onToggleToolbar}
          />
          <IconBtn
            icon={Icon.clock}
            label={
              autoRefresh
                ? `Auto-refresh on (${autoRefreshSecs}s) — :autorefresh`
                : "Auto-refresh off — :autorefresh"
            }
            primary={autoRefresh}
            onClick={onToggleAutoRefresh}
          />
          <IconBtn icon={Icon.help} label="Shortcuts (?)" onClick={onHelp} />
          <IconBtn icon={Icon.refresh} label="Refresh (R)" onClick={onRefresh} />
        </div>
      </div>
    </header>
  );
}

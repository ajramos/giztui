## Gmail TUI — Unified Search UX & Roadmap

### Overview
Objective: Implement a unified email search with two modes (remote/local) and an advanced form, integrated in the same screen as the message list, with consistent focus and UX patterns (similar to Labels and AI Summary panels).

### Desired User Experience
- Search overlay on top of the message list (same screen, not a separate page).
- Simple mode (25% overlay height):
  - Dynamic title with mode: “🔍 Gmail Search — Remote” or “🔍 Gmail Search — Local”.
  - Single input centered vertically and full width.
  - Placeholders/help in English:
    - Remote: explains operators (from:, to:, subject:, label:, has:attachment, is:unread, older_than:7d, newer_than:1m…). Help footer with “ESC to back”.
    - Local: explains it filters Subject, From, To, Snippet; multi-word AND. Help with “ESC to back”.
  - Ctrl+T toggles Remote/Local; title and placeholder update live.
  - Enter executes:
    - Remote: runs Gmail query and renders results in the list.
    - Local: filters in-memory and updates the list.
  - ESC closes the overlay and returns focus to the message list. If a search was active, reset to Inbox (no filters).
- Advanced mode (Ctrl+F):
  - Replaces the list area to gain space (within the same container), showing a form with fields:
    - From, To, Subject, Has the words, Doesn’t have (text with placeholders).
    - Size: one compact field like “<2MB”.
    - Date within: one compact field like “1m”, “2d”, “3w”, “4h”, “6y”.
    - Has attachment: checkbox.
    - Search: a selector that opens an integrated picker.
      - Picker lists special folders first (All Mail, Inbox, Sent, Spam, Trash, Starred, Important, Drafts…), then user labels alphabetically.
      - Supports filter and type-to-jump.
      - Enter selects and returns to the form; ESC returns without selecting.
  - Execution via Ctrl+Enter or Ctrl+S (builds the exact Gmail query and runs remote search).
  - ESC returns to the simple overlay (25%); a second ESC closes the overlay entirely.
- Navigation & focus:
  - Tab cycles through visible components in order: search (if visible) → list → message content → labels (if visible) → AI summary (if visible).
  - When focus is on inputs/dropdowns/pickers, global shortcuts must not interfere.
  - Focus indicators: only the focused pane shows yellow border; others gray. When search has focus, the list’s border is gray (not focused).
- Results:
  - Message list title shows the user’s exact query: “Search Results (N) — {query}”.
  - Pagination continues to work (e.g., load more).
  - Closing search (ESC) restores Inbox and its standard title.

### High-level Roadmap
1) Search container
   - Create/ensure an overlay container above the list (25% for simple). Allow expanding to occupy the list region for the advanced form.
2) Simple mode
   - Centered full-width input; dynamic title and placeholders; Enter/ESC/Ctrl+T handling; Inbox reset on close.
3) Advanced form
   - Fields: From, To, Subject, Has, Doesn’t have, Size (compact operator+value+unit), Date within (compact), Has attachment, Search (selector).
   - Single panel title (no duplicated headers). Ctrl+Enter/Ctrl+S execute; ESC returns to simple.
4) “Search” picker
   - Integrated in the panel (no full-screen modals). Special folders + user labels (alphabetical). Filter, type-to-jump, Enter/ESC, explicit focus.
5) Execute search & render results
   - Build Gmail query from the advanced form and execute via EmailService. Simple local: AND across Subject/From/To/Snippet using service layer filtering. Update title and list through service; keep pagination.
6) Focus & indicators
   - Tab cycle over all visible components. Yellow/gray borders per focus. Ignore global shortcuts when focus is in inputs/dropdowns/pickers.
7) State & reset
   - Clean state when closing overlays. Restore Inbox and default title.
8) UX copy
   - Placeholders/help in English; standardized “ESC to back”.
9) Manual testing
   - Open/close simple & advanced flows; toggle remote/local; use picker; execute searches; ESC at every level; tab across components; paginate; return to Inbox.
10) Polish
   - Performance for local filtering through service layer; friendly error messages via ErrorHandler.

### Status — Implemented
- Container & layout
  - Search overlay integrated in the same screen above the message list (25% simple). Advanced form expands to replace the list region.
- Simple mode UX
  - Dynamic title and border; input full width & vertically centered; English placeholders/help; Ctrl+T toggles Remote/Local; Enter executes; ESC closes and restores list. Remote search updates results through EmailService with safe UI updates via ErrorHandler.
  - Default `in:inbox` appended when the user query lacks `in:`/`label:`.
  - Title shows exact query: “Search Results (N) — {query}”.
- Local search
  - Multi-word AND across Subject, From, To, Snippet through service layer. Runs off-UI-thread and updates via QueueUpdateDraw. Title and list update using service methods.
- Advanced form
  - Fields present: From, To, Subject, Has, Doesn’t have, Size (single field), Date within (single field), Has attachment (checkbox), Search (selector). ESC returns to simple. Ctrl+Enter/Ctrl+S trigger search.
- “Search” picker
  - Integrated inside the search panel with the same UX pattern as “Browse all labels”: filter input + list, Enter selects and returns, ESC returns, explicit focus on filter. Special folders first, then user labels alphabetically. Pending final validation.
- Focus & indicators
  - Tab cycles through search, list, message content container, labels panel, AI summary. When search has focus, list border is gray.
  - Global shortcuts are ignored when focus is on InputField/DropDown/Form/List within the search UI.
- State & reset
  - Closing the search overlay via ESC resets Inbox for active remote searches and clears search state (mode, query, local filter, pagination).
- UX copy
  - Placeholders & help in English; “ESC to back” standardized.

### Status — Pending
- Type-to-jump buffer/backspace behavior fully robust in picker(s).
- Results pagination affordances inside search context (visuals/controls).
- Search-term highlighting in the message list/content.
- Additional empty/error states polish in the search UI.



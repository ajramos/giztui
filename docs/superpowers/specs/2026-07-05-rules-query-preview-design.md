# Rules: query preview + create-from-search — Design

Date: 2026-07-05 · Status: approved by user · Branch: feat/deterministic-rules

Two halves of the same idea: **the message list is the preview** — validate what a
rule's query matches by looking at real messages, before the rule exists.

## A. Preview from the rule form (Mac feedback item 4)

- The add/edit rule form gains a **Preview** button (order: Preview, Save, Cancel).
- Pressing it runs the form's current Query as a normal Gmail search and shows the
  results in the **main message list**, exactly like a user-typed search. The status
  bar reports the match count.
- The form **stays open** with its values intact; Tab reaches the list to inspect
  results and Tab returns to the form. Preview can be re-run after editing the query.
- The query is searched **verbatim** — no hidden scoping (no implicit `in:inbox`).
  What you see is what the query means.
- On form close (Save or Cancel), if a preview replaced the list, the list is
  **restored** to what was shown before the form opened (inbox or prior search).
- Empty query → warning via ErrorHandler, no search.

## B. Create a rule from any active search (Mac feedback item 5)

- Whenever the list is showing search results (quick `/`, `:search`, advanced
  search — any origin), a shortcut opens the rules panel **directly in the New rule
  form** with Query pre-filled with the active search query. The search results
  already on screen serve as the preview.
- Preferred key: Ctrl+S if free at implementation time; otherwise propose an
  alternative to the user before binding.
- Command parity: `:rules new` — with an active search it pre-fills its query;
  without one it opens a blank New rule form from anywhere.

## Out of scope (deliberate)

- No "apply rule now to these results" on save — `:rp` immediately after covers it.
- No new config options (nothing to migrate).
- Importing existing Gmail filters as rules: separate future design.

## Definition of done

- Command parity (`:rules new`), in-app `:help` updated, unit + smoke tests,
  `make pre-commit-check` green.

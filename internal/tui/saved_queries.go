package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ajramos/giztui/internal/services"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// queryItem represents a query item for the picker
type queryItem struct {
	id          int64
	name        string
	description string
	category    string
	query       string
	useCount    int
}

// parseSavedQuerySaveArgs splits ":save-query" arguments into a name and an
// optional category. A token starting with "@" sets the category (case as typed,
// minus the "@"); everything else joins into the name. E.g.
// ["Unpaid", "invoices", "@finance"] → ("Unpaid invoices", "finance"). With no
// "@" token the category is empty, landing the query in the picker's "Default"
// group. Mirrors the "@category" filter so save and browse share one convention.
func parseSavedQuerySaveArgs(args []string) (name, category string) {
	nameParts := make([]string, 0, len(args))
	for _, a := range args {
		if strings.HasPrefix(a, "@") && len(a) > 1 {
			category = strings.TrimSpace(strings.TrimPrefix(a, "@"))
			continue
		}
		nameParts = append(nameParts, a)
	}
	return strings.TrimSpace(strings.Join(nameParts, " ")), category
}

// savedQueryCategoryLabel is the display name of a query's category: the free-form
// category string, or "Default" for uncategorised entries. Grouping and the
// "@category" filter share this so they always agree.
func savedQueryCategoryLabel(c string) string {
	if strings.TrimSpace(c) == "" {
		return "Default"
	}
	return strings.TrimSpace(c)
}

// matchesSavedQueryFilter reports whether a query matches the picker filter. A
// filter beginning with "@" narrows by category (case-insensitive substring,
// e.g. "@work"); any other non-empty filter narrows by name. Empty matches all.
func matchesSavedQueryFilter(item queryItem, filter string) bool {
	f := strings.ToLower(strings.TrimSpace(filter))
	if f == "" {
		return true
	}
	if strings.HasPrefix(f, "@") {
		cat := strings.TrimSpace(f[1:])
		if cat == "" {
			return true
		}
		return strings.Contains(strings.ToLower(savedQueryCategoryLabel(item.category)), cat)
	}
	return strings.Contains(strings.ToLower(item.name), f)
}

// sortSavedQueriesByCategory orders queries by category — named groups
// alphabetically, the uncategorised "Default" group last — then by name, so the
// picker can render contiguous, headed groups.
func sortSavedQueriesByCategory(items []queryItem) {
	sort.SliceStable(items, func(i, j int) bool {
		ci, cj := savedQueryCategoryLabel(items[i].category), savedQueryCategoryLabel(items[j].category)
		di, dj := ci == "Default", cj == "Default"
		if di != dj {
			return !di // named categories before the Default group
		}
		if !strings.EqualFold(ci, cj) {
			return strings.ToLower(ci) < strings.ToLower(cj)
		}
		return strings.ToLower(items[i].name) < strings.ToLower(items[j].name)
	})
}

// showSavedQueriesPicker displays the saved queries picker interface using prompts-style picker
func (a *App) showSavedQueriesPicker() {
	// Get query service
	queryService := a.GetQueryService()
	if queryService == nil {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Query service not available - database may still be initializing")
		}()
		return
	}

	// Set account email if available (non-blocking)
	if queryServiceImpl, ok := queryService.(*services.QueryServiceImpl); ok {
		// Use a default account email for now - this should be set during app initialization
		queryServiceImpl.SetAccountEmail(a.getActiveAccountEmail())
	}

	// Create picker UI similar to prompts
	input := tview.NewInputField().
		SetLabel("🔍 Filter: ").
		SetFieldWidth(30).
		SetLabelColor(a.GetComponentColors("saved_queries").Title.Color()).
		SetFieldBackgroundColor(a.GetComponentColors("saved_queries").Background.Color()).
		SetFieldTextColor(a.GetComponentColors("saved_queries").Text.Color())
	input.SetPlaceholder("press / to filter · @cat by category")
	input.SetPlaceholderTextColor(a.getHintColor())
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)
	list.SetBackgroundColor(a.GetComponentColors("saved_queries").Background.Color()) // Component background

	// Apply component-specific selection colors
	queryColors := a.GetComponentColors("saved_queries")
	list.SetMainTextColor(queryColors.Text.Color())
	list.SetSelectedTextColor(queryColors.Background.Color())   // Use background for selected text (inverse)
	list.SetSelectedBackgroundColor(queryColors.Accent.Color()) // Use accent for selection highlight

	var all []queryItem
	var visible []queryItem
	// rowItems maps each list row to its index in `visible`, or -1 for a
	// non-selectable category header row, so number/delete keys and the current
	// selection map back to a real query even with headers interleaved.
	var rowItems []int
	// currentFilter mirrors the active filter text so an in-place refresh (after a
	// delete) can rebuild the same view without re-reading the DB.
	currentFilter := ""
	// deletePendingID arms the two-press delete confirmation (rules-manager shape):
	// first 'd' arms + shows a status prompt, a second 'd' on the same row deletes.
	deletePendingID := int64(0)
	clearDeletePending := func() {
		if deletePendingID != 0 {
			deletePendingID = 0
			go a.GetErrorHandler().ClearPersistentMessage()
		}
	}
	// firstRealRow returns the first non-header row so the cursor never starts on a
	// category header (headers are non-selectable, rowItems == -1).
	firstRealRow := func() int {
		for i, ri := range rowItems {
			if ri >= 0 {
				return i
			}
		}
		return 0
	}
	// selectedRealItem returns the query under the cursor, or false on a header row.
	selectedRealItem := func() (queryItem, bool) {
		cur := list.GetCurrentItem()
		if cur >= 0 && cur < len(rowItems) && rowItems[cur] >= 0 {
			return visible[rowItems[cur]], true
		}
		return queryItem{}, false
	}

	// Reload rebuilds the list from `all`, grouped by category with a header per
	// group. A bare filter narrows by name; a "@cat" filter narrows by category.
	reload := func(filter string) {
		list.Clear()
		visible = visible[:0]
		rowItems = rowItems[:0]
		lastCat := ""
		haveHeader := false
		for _, item := range all {
			if !matchesSavedQueryFilter(item, filter) {
				continue
			}

			// Emit a header row whenever the category group changes. Colour it with
			// the component Title colour (bold) via a tview tag so it stands apart
			// from the query rows (List honours color tags in the main text).
			cl := savedQueryCategoryLabel(item.category)
			if !haveHeader || cl != lastCat {
				list.AddItem(fmt.Sprintf("[%s::b]─ %s ─[-:-:-]", queryColors.Title.String(), cl), "", 0, nil)
				rowItems = append(rowItems, -1)
				lastCat = cl
				haveHeader = true
			}
			rowItems = append(rowItems, len(visible))
			visible = append(visible, item)

			// Category icon
			var icon string
			switch item.category {
			case "search":
				icon = "🔍"
			case "filter":
				icon = "🎯"
			case "advanced":
				icon = "⚙️"
			default:
				icon = "📚"
			}

			display := fmt.Sprintf("  %s %s", icon, item.name)
			if item.useCount > 0 {
				display += fmt.Sprintf(" (used %d times)", item.useCount)
			}

			// Capture variables for closure
			queryID := item.id // int64
			queryName := item.name
			queryText := item.query

			list.AddItem(display, item.query, 0, func() {
				// Execute query
				a.closeSavedQueriesPicker()

				// Record usage
				go func() {
					if err := queryService.RecordQueryUsage(a.ctx, queryID); err != nil {
						if a.logger != nil {
							a.logger.Printf("Failed to record query usage: %v", err)
						}
					}
				}()

				// Execute the query
				go a.performSearch(queryText)

				// Show what we're executing
				go func() {
					a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("🔍 Executing: %s", queryName))
				}()
			})
		}
	}

	// Load queries in background
	go func() {
		queries, err := queryService.ListQueries(a.ctx, "")
		if err != nil {
			go func() {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load saved queries: %v", err))
			}()
			return
		}

		if len(queries) == 0 {
			go func() {
				a.GetErrorHandler().ShowInfo(a.ctx, "No saved queries found. Save current search with 'Z' key.")
			}()
			return
		}

		// Convert to queryItem
		all = make([]queryItem, 0, len(queries))
		for _, q := range queries {
			all = append(all, queryItem{
				id:          q.ID,
				name:        q.Name,
				description: q.Description,
				category:    q.Category,
				query:       q.Query,
				useCount:    q.UseCount,
			})
		}
		sortSavedQueriesByCategory(all)

		a.QueueUpdateDraw(func() {
			// Set up filter field (list-first: '/' focuses this, typing filters live).
			input.SetChangedFunc(func(text string) {
				currentFilter = strings.TrimSpace(text)
				reload(currentFilter)
			})

			// Filter input: Down/PgDn return to the list; Esc clears the filter and
			// returns to the list (a second Esc on the list closes the picker).
			input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if a.pickerTabCycle(e) {
					return nil
				}
				switch e.Key() {
				case tcell.KeyDown, tcell.KeyPgDn:
					a.SetFocus(list)
					return nil
				case tcell.KeyEscape:
					input.SetText("")
					reload("")
					a.SetFocus(list)
					return nil
				}
				return e
			})

			// Enter in the filter executes the first visible match.
			input.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEnter && len(visible) > 0 {
					a.executeQueryItem(visible[0], queryService)
				}
			})

			// Handle list input capture (list-first CRUD; parity with the rules manager).
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if a.pickerTabCycle(e) {
					return nil
				}
				if e.Key() == tcell.KeyEscape {
					if deletePendingID != 0 {
						clearDeletePending() // cancel the armed delete only; panel stays open
						return nil
					}
					a.closeSavedQueriesPicker()
					return nil
				}
				// '/' enters filter mode (k9s-style; the picker is list-first).
				if e.Rune() == '/' {
					clearDeletePending()
					a.SetFocus(input)
					return nil
				}
				// Number keys for quick access (1-9): Nth visible query.
				if e.Rune() >= '1' && e.Rune() <= '9' {
					num := int(e.Rune() - '0')
					if num >= 1 && num <= len(visible) {
						a.executeQueryItem(visible[num-1], queryService)
						return nil
					}
				}
				// New query (configurable; default "n").
				if a.matchesConfiguredKey(e, a.Keys.SavedQueryNew) {
					clearDeletePending()
					a.showSavedQueryForm(nil, queryService)
					return nil
				}
				// Edit the selected query (configurable; default "e").
				if a.matchesConfiguredKey(e, a.Keys.SavedQueryEdit) {
					clearDeletePending()
					if item, ok := selectedRealItem(); ok {
						a.showSavedQueryForm(&item, queryService)
					}
					return nil
				}
				// Delete the selected query (configurable; default "d") — two-press
				// status-bar confirmation, the shape used across the app.
				if a.matchesConfiguredKey(e, a.Keys.SavedQueryDel) {
					item, ok := selectedRealItem()
					if !ok {
						return nil
					}
					if deletePendingID == item.id { // second press on the armed row → delete
						deletePendingID = 0
						a.performSavedQueryDelete(item, queryService, func() {
							// Drop the row locally and rebuild the same filtered view.
							for i := range all {
								if all[i].id == item.id {
									all = append(all[:i], all[i+1:]...)
									break
								}
							}
							a.QueueUpdateDraw(func() {
								reload(currentFilter)
								list.SetCurrentItem(firstRealRow())
							})
						})
						return nil
					}
					deletePendingID = item.id
					go a.GetErrorHandler().ShowPersistentMessage(a.ctx,
						fmt.Sprintf("Delete query '%s'? Press '%s' again to confirm, Esc cancels", item.name, a.Keys.SavedQueryDel),
						LogLevelInfo)
					return nil
				}
				return e
			})

			// Create container and show
			container := tview.NewFlex().SetDirection(tview.FlexRow)
			bgColor := a.GetComponentColors("saved_queries").Background.Color()
			container.SetBackgroundColor(bgColor) // Consistent background
			container.SetBorder(true)
			container.SetBorderColor(a.GetComponentColors("saved_queries").Border.Color()) // Set initial border color
			container.SetTitle(" 📚 Saved Queries ")
			container.SetTitleColor(a.GetComponentColors("saved_queries").Title.Color()) // Use component colors

			// Set background on child components as well
			input.SetBackgroundColor(bgColor)
			list.SetBackgroundColor(bgColor)

			// Add spacing like attachments picker (3 lines for input). List-first:
			// the list holds initial focus, the filter is reached with '/'.
			container.AddItem(input, 3, 0, false)
			container.AddItem(list, 0, 1, true)

			// Footer with instructions (standardized footer color)
			footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
			footer.SetText(" Enter/1-9 run · / filter (@cat) · n new · e edit · d delete · Esc close ")
			footer.SetTextColor(a.GetComponentColors("general").Text.Color()) // Standardized footer color like other pickers
			footer.SetBackgroundColor(bgColor)
			container.AddItem(footer, 1, 0, false)

			// Initial population; start the cursor on the first real (non-header) row.
			reload("")
			list.SetCurrentItem(firstRealRow())

			// Add to content split (like labels/prompts)
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				if a.labelsView != nil {
					split.RemoveItem(a.labelsView)
				}
				a.labelsView = container
				split.SetBackgroundColor(bgColor)
				split.AddItem(a.labelsView, 0, 1, true)
				split.ResizeItem(a.labelsView, 0, 1)
			}

			// Set focus and state (use "labels" for proper border highlighting)
			a.markFocus("labels")
			a.setActivePicker(PickerSavedQueries)

			// List-first: focus the list so n/e/d and 1-9 act immediately ('/' filters).
			a.SetFocus(list)
		})
	}()
}

// executeQueryItem executes a selected query item
func (a *App) executeQueryItem(item queryItem, queryService services.QueryService) {
	a.closeSavedQueriesPicker()

	// Record usage
	go func() {
		if err := queryService.RecordQueryUsage(a.ctx, item.id); err != nil {
			if a.logger != nil {
				a.logger.Printf("Failed to record query usage: %v", err)
			}
		}
	}()

	// Execute the query
	go a.performSearch(item.query)

	// Show what we're executing
	go func() {
		a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("🔍 Executing: %s", item.name))
	}()
}

// closeSavedQueriesPicker closes the saved queries picker
func (a *App) closeSavedQueriesPicker() {
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 0)
	}
	a.setActivePicker(PickerNone)
	a.restoreFocusAfterModal()
}

// closeSaveQueryPanel closes the save query input panel
func (a *App) closeSaveQueryPanel() {
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 0)
	}
	a.setActivePicker(PickerNone)
	a.restoreFocusAfterModal()
}

// performQuerySave saves the query with the provided details
func (a *App) performQuerySave(name, query, description, category string, queryService services.QueryService) {
	// Close panel immediately (like Obsidian does)
	a.QueueUpdateDraw(func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.labelsView, 0, 0)
		}
		a.setActivePicker(PickerNone)
		// Restore focus to message list
		a.SetFocus(a.views["list"])
		a.markFocus("list")
	})

	// Show progress
	a.GetErrorHandler().ShowProgress(a.ctx, "💾 Saving search query...")

	// Save query
	_, err := queryService.SaveQuery(a.ctx, name, query, description, category)
	a.GetErrorHandler().ClearProgress()

	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to save query: %v", err))
	} else {
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Saved query: %s", name))
	}
}

// showSaveCurrentQueryDialog shows input panel to save current search using Obsidian-style bottom-right panel
func (a *App) showSaveCurrentQueryDialog() {
	// Get current query from list title or last search
	currentQuery := a.getCurrentSearchQuery()
	if strings.TrimSpace(currentQuery) == "" {
		go func() {
			a.GetErrorHandler().ShowWarning(a.ctx, "No current search to save. Perform a search first.")
		}()
		return
	}

	// Get query service
	queryService := a.GetQueryService()
	if queryService == nil {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Query service not available - database may still be initializing")
		}()
		return
	}

	// Set account email if available (non-blocking)
	if queryServiceImpl, ok := queryService.(*services.QueryServiceImpl); ok {
		queryServiceImpl.SetAccountEmail(a.getActiveAccountEmail())
	}

	// Show save query input panel (following Obsidian pattern)
	go a.showSaveQueryInput(currentQuery, queryService)
}

// showSaveQueryInput shows input panel for saving a query using Obsidian-style bottom-right panel
func (a *App) showSaveQueryInput(query string, queryService services.QueryService) {
	// Create panel similar to Obsidian ingestion panel
	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(a.GetComponentColors("saved_queries").Background.Color())
	container.SetBorder(true)
	container.SetBorderColor(a.GetComponentColors("saved_queries").Border.Color()) // Set initial border color
	container.SetTitle(" 💾 Save Search Query ")
	container.SetTitleColor(a.GetComponentColors("saved_queries").Title.Color())

	// Show query preview with proper theming
	queryPreview := fmt.Sprintf(`🔍 QUERY PREVIEW

This search query will be saved:

"%s"

You can execute it later using the bookmarks picker (Q key) or the :bookmark command.`, query)
	queryView := tview.NewTextView()
	queryView.SetText(queryPreview).
		SetScrollable(true).
		SetWordWrap(true).
		SetTextColor(a.GetComponentColors("saved_queries").Text.Color()).            // Theme text color
		SetBackgroundColor(a.GetComponentColors("saved_queries").Background.Color()) // Theme background color
	queryView.SetBorder(false) // Set border separately

	// Name input label and field (following Obsidian pattern exactly)
	nameLabel := tview.NewTextView().SetText("💾 Query name:")
	nameLabel.SetTextColor(a.GetComponentColors("saved_queries").Title.Color())
	nameLabel.SetBackgroundColor(a.GetComponentColors("saved_queries").Background.Color())

	nameInput := tview.NewInputField()
	nameInput.SetLabel("") // No built-in label, using separate TextView like Obsidian
	nameInput.SetText("")
	nameInput.SetPlaceholder("Enter a descriptive name for this search query...")
	nameInput.SetFieldWidth(50)
	nameInput.SetBorder(false)                                                                  // No border for cleaner look
	nameInput.SetBackgroundColor(a.GetComponentColors("saved_queries").Background.Color())      // InputField container background
	nameInput.SetFieldBackgroundColor(a.GetComponentColors("saved_queries").Background.Color()) // Component background (not accent)
	nameInput.SetFieldTextColor(a.GetComponentColors("saved_queries").Text.Color())             // Component text color
	nameInput.SetPlaceholderTextColor(a.getHintColor())                                         // Consistent placeholder color

	// Generate default name
	if queryServiceImpl, ok := queryService.(*services.QueryServiceImpl); ok {
		defaultName := queryServiceImpl.GenerateQueryName(query)
		nameInput.SetText(defaultName)
	}

	// Instructions
	instructions := tview.NewTextView().SetTextAlign(tview.AlignRight)
	instructions.SetText("Enter to save | Esc to cancel")
	instructions.SetTextColor(a.GetComponentColors("general").Text.Color())
	instructions.SetBackgroundColor(a.GetComponentColors("saved_queries").Background.Color())

	// Create a horizontal flex for label and input alignment with controlled spacing
	nameRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	nameRow.SetBackgroundColor(a.GetComponentColors("saved_queries").Background.Color()) // Ensure container background matches
	nameRow.AddItem(nameLabel, 17, 0, false)                                             // Fixed width for label (17 chars for "💾 Query name:")
	nameRow.AddItem(nameInput, 50, 0, false)                                             // Fixed width for input (50 chars)

	// Spacer with proper background theming
	spacer := tview.NewBox()
	spacer.SetBackgroundColor(a.GetComponentColors("saved_queries").Background.Color())
	nameRow.AddItem(spacer, 0, 1, false) // Spacer takes remaining space

	// Add items to container with proper proportions
	container.AddItem(queryView, 0, 1, false)    // Query preview takes most space
	container.AddItem(nameRow, 2, 0, false)      // Name label and input in same row
	container.AddItem(instructions, 1, 0, false) // Instructions take minimal space

	// Add to content split like Obsidian
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = container
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}

	// Set focus and state (use "labels" for proper border highlighting)
	a.markFocus("labels")
	a.setActivePicker(PickerSavedQueries)

	// Configure input handling
	nameInput.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape {
			a.closeSaveQueryPanel()
			return nil
		}
		if e.Key() == tcell.KeyEnter {
			// Get name and save
			name := strings.TrimSpace(nameInput.GetText())
			if name == "" {
				go func() {
					a.GetErrorHandler().ShowWarning(a.ctx, "Query name cannot be empty")
				}()
				return nil
			}
			// Perform save with default values
			go a.performQuerySave(name, query, "", "general", queryService)
			return nil
		}
		return e
	})

	// Container-level input capture for Escape
	container.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape {
			a.closeSaveQueryPanel()
			return nil
		}
		return e
	})

	// Set focus to input immediately
	a.SetFocus(nameInput)
	a.QueueUpdateDraw(func() {
		a.SetFocus(nameInput)
	})
}

// getCurrentSearchQuery gets current search query from app state
func (a *App) getCurrentSearchQuery() string {
	currentQuery := a.GetCurrentQuery()

	if currentQuery != "" {
		// The currentQuery includes additional filters, try to get the original query
		// by parsing the title first, fallback to currentQuery
		if list, ok := a.views["list"].(*tview.Table); ok {
			title := list.GetTitle()

			// Try different title formats:

			// 1. Initial search: " 🔍 Searching: has:attachment "
			if strings.Contains(title, "🔍 Searching: ") {
				start := strings.Index(title, "🔍 Searching: ") + len("🔍 Searching: ")
				end := len(title)
				if end > start {
					query := strings.TrimSpace(title[start:end])
					query = strings.TrimSuffix(query, " ")
					if query != "" {
						return query
					}
				}
			}

			// 2. Completed search: " 🔍 Search Results (10) — has:attachment "
			if strings.Contains(title, "🔍 Search Results") && strings.Contains(title, " — ") {
				parts := strings.Split(title, " — ")
				if len(parts) >= 2 {
					query := strings.TrimSpace(parts[len(parts)-1])
					query = strings.TrimSuffix(query, " ")
					if query != "" {
						return query
					}
				}
			}

			// 3. Spinner during search: " ⠋ Searching… (5/10) — has:attachment "
			if strings.Contains(title, " — ") {
				parts := strings.Split(title, " — ")
				if len(parts) >= 2 {
					query := strings.TrimSpace(parts[len(parts)-1])
					query = strings.TrimSuffix(query, " ")
					if query != "" {
						return query
					}
				}
			}
		}

		// Fallback: return currentQuery but remove the auto-added filters
		query := currentQuery
		// Remove common auto-added filters to get the original user query
		query = strings.ReplaceAll(query, " -in:sent -in:draft -in:chat -in:spam -in:trash in:inbox", "")
		query = strings.TrimSpace(query)
		if query != "" {
			return query
		}
	}

	// No current search
	return ""
}

// OBLITERATED: unused editSavedQuery function eliminated! 💥

// OBLITERATED: unused deleteSavedQuery function eliminated! 💥

// performSavedQueryDelete deletes a query via QueryService and runs onDone (an
// in-place list refresh) on success. The persistent confirmation prompt is cleared
// first; the picker stays open so the refreshed list is shown.
func (a *App) performSavedQueryDelete(item queryItem, queryService services.QueryService, onDone func()) {
	go func() {
		a.GetErrorHandler().ClearPersistentMessage()
		if err := queryService.DeleteQuery(a.ctx, item.id); err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to delete query: %v", err))
			return
		}
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Deleted query: %s", item.name))
		if onDone != nil {
			onDone()
		}
	}()
}

// showSavedQueryForm opens an inline form to create (existing == nil) or edit a
// saved query's name, query and category, persisting via QueryService.SaveQuery /
// UpdateQuery. Reopens the picker (refreshed) on save or cancel.
func (a *App) showSavedQueryForm(existing *queryItem, queryService services.QueryService) {
	colors := a.GetComponentColors("saved_queries")

	name, query, category := "", "", ""
	if existing != nil {
		name, query, category = existing.name, existing.query, existing.category
	}

	form := tview.NewForm()
	form.SetBackgroundColor(colors.Background.Color())
	form.SetFieldBackgroundColor(colors.Background.Color())
	form.SetFieldTextColor(colors.Text.Color())
	form.SetLabelColor(colors.Title.Color())
	form.SetButtonBackgroundColor(colors.Background.Color())
	form.SetButtonTextColor(colors.Text.Color())
	form.AddInputField("Name", name, 44, nil, nil)
	form.AddInputField("Query", query, 44, nil, nil)
	form.AddInputField("Category", category, 44, nil, nil)

	field := func(label string) string {
		if fi, ok := form.GetFormItemByLabel(label).(*tview.InputField); ok {
			return strings.TrimSpace(fi.GetText())
		}
		return ""
	}
	reopen := func() {
		a.closeSavedQueriesPicker()
		a.showSavedQueriesPicker()
	}
	save := func() {
		n, q, cat := field("Name"), field("Query"), field("Category")
		if n == "" || q == "" {
			go a.GetErrorHandler().ShowWarning(a.ctx, "Name and query are required")
			return
		}
		go func() {
			var err error
			if existing == nil {
				_, err = queryService.SaveQuery(a.ctx, n, q, "", cat)
			} else {
				err = queryService.UpdateQuery(a.ctx, existing.id, n, q, "", cat)
			}
			switch {
			case err != nil:
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to save query: %v", err))
			case existing == nil:
				a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Created query: %s", n))
			default:
				a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Updated query: %s", n))
			}
		}()
		reopen()
	}
	form.AddButton("Save", save)
	form.AddButton("Cancel", reopen)
	form.SetCancelFunc(reopen) // Esc

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(colors.Background.Color())
	container.SetBorder(true)
	container.SetBorderColor(colors.Border.Color())
	title := " ✏️  Edit Saved Query "
	if existing == nil {
		title = " ➕ New Saved Query "
	}
	container.SetTitle(title)
	container.SetTitleColor(colors.Title.Color())
	container.AddItem(form, 0, 1, true)

	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetText(" Tab move · Enter on Save · Esc cancel ")
	footer.SetTextColor(a.GetComponentColors("general").Text.Color())
	footer.SetBackgroundColor(colors.Background.Color())
	container.AddItem(footer, 1, 0, false)

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = container
		split.SetBackgroundColor(colors.Background.Color())
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.markFocus("labels")
	a.setActivePicker(PickerSavedQueries)
	a.SetFocus(form)
}

// Helper function to show query by name (for command usage)
func (a *App) executeQueryByName(name string) {
	queryService := a.GetQueryService()
	if queryService == nil {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Query service not available")
		}()
		return
	}

	// Set account email if available
	if queryServiceImpl, ok := queryService.(*services.QueryServiceImpl); ok {
		if email := a.getActiveAccountEmail(); email != "" {
			queryServiceImpl.SetAccountEmail(email)
		}
	}

	go func() {
		query, err := queryService.GetQuery(a.ctx, name)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Query '%s' not found", name))
			return
		}

		// Record usage
		_ = queryService.RecordQueryUsage(a.ctx, query.ID)

		// Execute query
		a.performSearch(query.Query)

		// Show feedback
		go func() {
			a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("🔍 Executing: %s", query.Name))
		}()
	}()
}

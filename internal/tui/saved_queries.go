package tui

import (
	"fmt"
	"strings"

	"github.com/ajramos/gmail-tui/internal/services"
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
		SetLabel("🔍 Search: ").
		SetFieldWidth(30).
		SetLabelColor(a.getTitleColor()).
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetFieldTextColor(tview.Styles.PrimaryTextColor)
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)
	list.SetBackgroundColor(a.GetComponentColors("saved_queries").Background.Color()) // Component background

	var all []queryItem
	var visible []queryItem

	// Reload function for filtering
	reload := func(filter string) {
		list.Clear()
		visible = visible[:0]
		for _, item := range all {
			if filter != "" && !strings.Contains(strings.ToLower(item.name), strings.ToLower(filter)) {
				continue
			}
			visible = append(visible, item)

			// Category icon
			icon := "📚"
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

			display := fmt.Sprintf("%s %s", icon, item.name)
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

		a.QueueUpdateDraw(func() {
			// Set up input field
			input.SetChangedFunc(func(text string) { reload(strings.TrimSpace(text)) })

			// Allow navigation from input to list
			input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyDown || e.Key() == tcell.KeyUp || e.Key() == tcell.KeyPgDn || e.Key() == tcell.KeyPgUp {
					a.SetFocus(list)
					return e
				}
				return e
			})

			// Handle enter in input field (select first match)
			input.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					a.closeSavedQueriesPicker()
					return
				}
				if key == tcell.KeyEnter && list.GetItemCount() > 0 {
					// Execute first match
					filtered := a.filterQueriesByName(all, input.GetText())
					if len(filtered) > 0 {
						a.executeQueryItem(filtered[0], queryService)
					}
				}
			})

			// Handle list input capture
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyEscape {
					a.closeSavedQueriesPicker()
					return nil
				}
				// Allow up arrow from first item to go back to input
				if e.Key() == tcell.KeyUp && list.GetCurrentItem() == 0 {
					a.SetFocus(input)
					return nil
				}
				// Handle number keys for quick access (1-9)
				if e.Rune() >= '1' && e.Rune() <= '9' {
					num := int(e.Rune() - '0')
					filtered := a.filterQueriesByName(all, input.GetText())
					if num > 0 && num <= len(filtered) {
						item := filtered[num-1]
						a.executeQueryItem(item, queryService)
						return nil
					}
				}
				// Handle delete key
				if e.Rune() == 'd' || e.Rune() == 'D' {
					filtered := a.filterQueriesByName(all, input.GetText())
					currentItem := list.GetCurrentItem()
					if currentItem >= 0 && currentItem < len(filtered) {
						item := filtered[currentItem]
						a.deleteSavedQueryItem(item, queryService)
						return nil
					}
				}
				return e
			})

			// Create container and show
			container := tview.NewFlex().SetDirection(tview.FlexRow)
			container.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor) // Consistent background
			container.SetBorder(true)
			container.SetBorderColor(tview.Styles.BorderColor) // Set initial border color
			container.SetTitle(" 📚 Saved Queries ")
			container.SetTitleColor(a.GetComponentColors("saved_queries").Title.Color()) // Use component colors

			// Add spacing like attachments picker (3 lines for input)
			container.AddItem(input, 3, 0, true)
			container.AddItem(list, 0, 1, false)

			// Footer with instructions (standardized footer color)
			footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
			footer.SetText(" Enter/1-9 to execute | d/D to delete | Esc to cancel ")
			footer.SetTextColor(a.GetComponentColors("general").Text.Color()) // Standardized footer color like other pickers
			container.AddItem(footer, 1, 0, false)

			// Initial population
			reload("")

			// Add to content split (like labels/prompts)
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				if a.labelsView != nil {
					split.RemoveItem(a.labelsView)
				}
				a.labelsView = container
				split.AddItem(a.labelsView, 0, 1, true)
				split.ResizeItem(a.labelsView, 0, 1)
			}

			// Set focus and state (use "labels" for proper border highlighting)
			a.currentFocus = "labels"
			a.updateFocusIndicators("labels")
			a.labelsVisible = true

			// Set focus to input
			a.SetFocus(input)
		})
	}()
}

// filterQueriesByName filters queries by name for input field
func (a *App) filterQueriesByName(queries []queryItem, filterText string) []queryItem {
	if strings.TrimSpace(filterText) == "" {
		return queries
	}

	filterLower := strings.ToLower(strings.TrimSpace(filterText))
	var filtered []queryItem

	for _, query := range queries {
		// Search in name, description, query, and category
		searchableText := strings.ToLower(fmt.Sprintf("%s %s %s %s",
			query.name, query.description, query.query, query.category))

		if strings.Contains(searchableText, filterLower) {
			filtered = append(filtered, query)
		}
	}

	return filtered
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
	a.labelsVisible = false
	a.restoreFocusAfterModal()
}

// closeSaveQueryPanel closes the save query input panel
func (a *App) closeSaveQueryPanel() {
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 0)
	}
	a.labelsVisible = false
	a.restoreFocusAfterModal()
}

// performQuerySave saves the query with the provided details
func (a *App) performQuerySave(name, query, description, category string, queryService services.QueryService) {
	// Close panel immediately (like Obsidian does)
	a.QueueUpdateDraw(func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.labelsView, 0, 0)
		}
		a.labelsVisible = false
		// Restore focus to message list
		a.SetFocus(a.views["list"])
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
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
	container.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	container.SetBorder(true)
	container.SetBorderColor(tview.Styles.BorderColor) // Set initial border color
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
	nameLabel.SetTextColor(a.getTitleColor())

	nameInput := tview.NewInputField()
	nameInput.SetLabel("") // No built-in label, using separate TextView like Obsidian
	nameInput.SetText("")
	nameInput.SetPlaceholder("Enter a descriptive name for this search query...")
	nameInput.SetFieldWidth(50)
	nameInput.SetBorder(false)                                                                  // No border for cleaner look
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

	// Create a horizontal flex for label and input alignment with controlled spacing
	nameRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	nameRow.AddItem(nameLabel, 17, 0, false)     // Fixed width for label (17 chars for "💾 Query name:")
	nameRow.AddItem(nameInput, 50, 0, false)     // Fixed width for input (50 chars)
	nameRow.AddItem(tview.NewBox(), 0, 1, false) // Spacer takes remaining space

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
	a.currentFocus = "labels"
	a.updateFocusIndicators("labels")
	a.labelsVisible = true

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


// editSavedQuery shows edit dialog for a saved query (placeholder for future implementation)
func (a *App) editSavedQuery(query *services.SavedQueryInfo) {
	go func() {
		a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Edit feature coming soon. Query: %s", query.Name))
	}()
}

// deleteSavedQuery shows delete confirmation and deletes query (placeholder for future implementation)
func (a *App) deleteSavedQuery(query *services.SavedQueryInfo) {
	queryService := a.GetQueryService()
	if queryService == nil {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Query service not available")
		}()
		return
	}

	// Hide picker first
	a.closeSavedQueriesPicker()

	// Show confirmation and delete
	go func() {
		if err := queryService.DeleteQuery(a.ctx, query.ID); err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to delete query: %v", err))
		} else {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Deleted query: %s", query.Name))
		}
	}()
}

// deleteSavedQueryItem deletes a query item from the picker
func (a *App) deleteSavedQueryItem(item queryItem, queryService services.QueryService) {
	// Hide picker first
	a.closeSavedQueriesPicker()

	// Show confirmation and delete
	go func() {
		if err := queryService.DeleteQuery(a.ctx, item.id); err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to delete query: %v", err))
		} else {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Deleted query: %s", item.name))
		}
	}()
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
		queryService.RecordQueryUsage(a.ctx, query.ID)

		// Execute query
		a.performSearch(query.Query)

		// Show feedback
		go func() {
			a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("🔍 Executing: %s", query.Name))
		}()
	}()
}

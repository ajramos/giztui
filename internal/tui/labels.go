package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

// executeLabelAdd adds a label to the current message
func (a *App) executeLabelAdd(args []string) {
	labelName := strings.Join(args, " ")
	if labelName == "" {
		a.showError("Label name cannot be empty")
		return
	}

	messageID := a.getCurrentMessageID()
	if messageID == "" {
		a.showError("No message selected")
		return
	}

	go func() {
		label, err := a.Client.CreateLabel(labelName)
		if err != nil {
			labels, err := a.Client.ListLabels()
			if err != nil {
				a.showError(fmt.Sprintf("❌ Error creating/finding label: %v", err))
				return
			}
			for _, l := range labels {
				if l.Name == labelName {
					label = l
					break
				}
			}
			if label == nil {
				a.showError(fmt.Sprintf("❌ Error creating label: %v", err))
				return
			}
		}
		if err := a.Client.ApplyLabel(messageID, label.Id); err != nil {
			a.showError(fmt.Sprintf("❌ Error applying label: %v", err))
			return
		}
		a.showStatusMessage(fmt.Sprintf("🏷️  Applied label: %s", labelName))
	}()
}

// executeLabelRemove removes a label from the current message
func (a *App) executeLabelRemove(args []string) {
	labelName := strings.Join(args, " ")
	if labelName == "" {
		a.showError("Label name cannot be empty")
		return
	}
	messageID := a.getCurrentMessageID()
	if messageID == "" {
		a.showError("No message selected")
		return
	}
	go func() {
		labels, err := a.Client.ListLabels()
		if err != nil {
			a.showError(fmt.Sprintf("❌ Error loading labels: %v", err))
			return
		}
		var labelID string
		for _, l := range labels {
			if l.Name == labelName {
				labelID = l.Id
				break
			}
		}
		if labelID == "" {
			a.showError(fmt.Sprintf("❌ Label not found: %s", labelName))
			return
		}
		if err := a.Client.RemoveLabel(messageID, labelID); err != nil {
			a.showError(fmt.Sprintf("❌ Error removing label: %v", err))
			return
		}
		a.showStatusMessage(fmt.Sprintf("🏷️  Removed label: %s", labelName))
	}()
}

// manageLabels opens the labels management view for the currently selected message
func (a *App) manageLabels() {
	// Toggle contextual panel like AI Summary
	if a.labelsVisible {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.labelsView, 0, 0)
		}
		a.labelsVisible = false
		a.SetFocus(a.views["text"])
		a.currentFocus = "text"
		a.updateFocusIndicators("text")
		a.showStatusMessage("🙈 Labels ocultas")
		return
	}

	messageID := a.getCurrentMessageID()
	if messageID == "" {
		a.showError("❌ No message selected")
		return
	}

	// Ensure message content is shown without stealing focus
	a.showMessageWithoutFocus(messageID)

	// Show panel and load quick view
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.labelsVisible = true
	a.labelsExpanded = false
	a.currentFocus = "labels"
	a.updateFocusIndicators("labels")
	a.populateLabelsQuickView(messageID)
}

// showMessageLabelsView displays labels for a specific message
func (a *App) showMessageLabelsView(labels []*gmailapi.Label, message *gmailapi.Message) {
	// Backwards-compat modal; keep for command bar or future use
	// Create labels list view
	labelsList := tview.NewList()
	labelsList.SetBorder(true)
	labelsList.SetTitle(" 🏷️  Message Labels ")

	// Get current message labels
	currentLabels := make(map[string]bool)
	if message.LabelIds != nil {
		for _, labelID := range message.LabelIds {
			currentLabels[labelID] = true
		}
	}

	// Extract subject for display
	subject := "Unknown subject"
	if message.Payload != nil && message.Payload.Headers != nil {
		for _, header := range message.Payload.Headers {
			if header.Name == "Subject" {
				subject = header.Value
				break
			}
		}
	}

	// Partition applied vs not-applied and sort each group; applied first
	applied, notApplied := a.partitionAndSortLabels(labels, currentLabels)
	for _, label := range append(applied, notApplied...) {
		// Store label info for the callback (avoid capturing loop vars directly)
		labelID := label.Id
		labelName := label.Name

		// Determine current applied state
		isApplied := currentLabels[labelID]

		// Create display text with label info and status (no secondary text)
		var displayText string
		if isApplied {
			displayText = fmt.Sprintf("✅ %s", labelName)
		} else {
			displayText = fmt.Sprintf("○ %s", labelName)
		}

		labelsList.AddItem(displayText, "", 0, func() {
			// Capture current index and state at click time
			index := labelsList.GetCurrentItem()
			currentlyApplied := currentLabels[labelID]
			// Async toggle
			a.toggleLabelForMessage(message.Id, labelID, labelName, currentlyApplied, func(newApplied bool, err error) {
				if err != nil {
					return
				}
				// Update local state map
				currentLabels[labelID] = newApplied
				// Update UI immediately
				newText := fmt.Sprintf("○ %s", labelName)
				if newApplied {
					newText = fmt.Sprintf("✅ %s", labelName)
				}
				a.QueueUpdateDraw(func() {
					labelsList.SetItemText(index, newText, "")
				})
				// Update cached meta for main list (for UNREAD/star, etc.)
				a.updateCachedMessageLabels(message.Id, labelID, newApplied)
			})
		})
	}

	// Add "Create new label" option at the end
	labelsList.AddItem("➕ Create new label", "Press Enter to create", 0, func() {
		a.createNewLabelFromView()
	})

	// Ensure first item is selected to enable immediate arrow navigation
	if labelsList.GetItemCount() > 0 {
		labelsList.SetCurrentItem(0)
	}

	// Set up key bindings for the labels view
	labelsList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			// Return to main view
			a.Pages.SwitchToPage("main")
			a.restoreFocusAfterModal()
			// Refresh currently displayed message content to reflect new labels
			go a.refreshMessageContent(message.Id)
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'n' {
				// Create new label
				a.createNewLabelFromView()
				return nil
			}
			if event.Rune() == 'r' {
				// Refresh labels view
				go a.manageLabels()
				return nil
			}
		}
		return event
	})

	// Create the labels view page
	labelsView := tview.NewFlex().SetDirection(tview.FlexRow)

	// Title with message subject
	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetText(fmt.Sprintf("🏷️  Labels for: %s", subject))
	title.SetTextColor(tcell.ColorYellow)
	title.SetBorder(true)

	// Instructions
	instructions := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	instructions.SetText("Enter: Toggle label | n: Create new label | r: Refresh | ESC: Back")
	instructions.SetTextColor(tcell.ColorGray)

	labelsView.AddItem(title, 3, 0, false)
	labelsView.AddItem(labelsList, 0, 1, true)
	labelsView.AddItem(instructions, 2, 0, false)

	// Add labels view to pages
	a.Pages.AddPage("messageLabels", labelsView, true, true)
	a.Pages.SwitchToPage("messageLabels")
	a.SetFocus(labelsList)
}

// populateLabelsQuickView renders current labels + quick actions in the side panel
func (a *App) populateLabelsQuickView(messageID string) {
	go func() {
		msg, err := a.Client.GetMessage(messageID)
		if err != nil {
			a.showError("❌ Error loading message")
			return
		}
		labels, err := a.Client.ListLabels()
		if err != nil {
			a.showError("❌ Error loading labels")
			return
		}
		// Build quick view UI off-thread then apply
		current := make(map[string]bool)
		for _, lid := range msg.LabelIds {
			current[lid] = true
		}
		applied, notApplied := a.partitionAndSortLabels(labels, current)

		body := tview.NewList().ShowSecondaryText(false)
		body.SetBorder(false)
		// Current labels first (checked)
		for _, l := range applied {
			name := l.Name
			lid := l.Id
			body.AddItem("✅ "+name, "Enter: toggle off", 0, func() {
				a.toggleLabelForMessage(messageID, lid, name, true, func(newApplied bool, err error) {
					if err == nil {
						a.updateCachedMessageLabels(messageID, lid, newApplied)
						a.updateMessageCacheLabels(messageID, name, newApplied)
						a.populateLabelsQuickView(messageID)
						a.refreshMessageContent(messageID)
					}
				})
			})
		}
		// Quick actions: first N from notApplied
		maxQuick := 6
		for i, l := range notApplied {
			if i >= maxQuick {
				break
			}
			name := l.Name
			lid := l.Id
			body.AddItem("○ "+name, "Enter: apply", 0, func() {
				a.toggleLabelForMessage(messageID, lid, name, false, func(newApplied bool, err error) {
					if err == nil {
						a.updateCachedMessageLabels(messageID, lid, newApplied)
						a.updateMessageCacheLabels(messageID, name, newApplied)
						a.populateLabelsQuickView(messageID)
						a.refreshMessageContent(messageID)
					}
				})
			})
		}
		// Actions
		body.AddItem("🔍 Browse all labels…", "Expand panel", 0, func() {
			a.expandLabelsBrowse(messageID)
		})
		body.AddItem("➕ Add custom label…", "Create or apply", 0, func() {
			a.addCustomLabelInline(messageID)
		})

		// Capture ESC in quick view to close panel
		body.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
			if e.Key() == tcell.KeyEscape {
				if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
					split.ResizeItem(a.labelsView, 0, 0)
				}
				a.labelsVisible = false
				// Salir también de bulk si estaba activo
				if a.bulkMode {
					a.bulkMode = false
					a.selected = make(map[string]bool)
					a.reformatListItems()
					a.setStatusPersistent("")
					if l, ok := a.views["list"].(*tview.List); ok {
						l.SetSelectedTextColor(tcell.ColorWhite)
						l.SetSelectedBackgroundColor(tcell.ColorBlue)
					}
				}
				// Restore list navigation
				if l, ok := a.views["list"].(*tview.Table); ok {
					l.SetInputCapture(nil)
				}
				a.SetFocus(a.views["text"])
				a.currentFocus = "text"
				a.updateFocusIndicators("text")
				return nil
			}
			// Keep focus anchored in labels list when using Up/Down
			if e.Key() == tcell.KeyUp || e.Key() == tcell.KeyDown {
				a.currentFocus = "labels"
				a.updateFocusIndicators("labels")
				return e
			}
			return e
		})

		container := tview.NewFlex().SetDirection(tview.FlexRow)
		container.SetBorder(true)
		container.SetTitle(" 🏷️ Labels ")
		container.SetTitleColor(tcell.ColorYellow)
		container.AddItem(body, 0, 1, true)

		a.QueueUpdateDraw(func() {
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				// replace labelsView item with new container
				split.RemoveItem(a.labelsView)
				a.labelsView = container
				split.AddItem(a.labelsView, 0, 1, false)
			}
			// While labels son visibles, solo tragamos flechas en la lista
			// cuando el foco actual está en labels. Si el usuario cambia con
			// Tab a la lista, las flechas deben funcionar normalmente.
			if l, ok := a.views["list"].(*tview.List); ok {
				l.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
					if a.labelsVisible && a.currentFocus == "labels" {
						switch ev.Key() {
						case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn, tcell.KeyHome, tcell.KeyEnd:
							return nil
						}
					}
					return ev
				})
			}
			// Solo forzar foco si ya estamos en labels (toggle inicial)
			if a.currentFocus == "labels" {
				a.SetFocus(body)
				a.updateFocusIndicators("labels")
				// Preselect first label item for proper arrow navigation
				if body.GetItemCount() > 0 {
					body.SetCurrentItem(0)
				}
			}
		})
	}()
}

// expandLabelsBrowse shows full list with search inside the side panel
func (a *App) expandLabelsBrowse(messageID string) {
	a.expandLabelsBrowseWithMode(messageID, false)
}

// expandLabelsBrowseWithMode shows full list with search inside the side panel.
// If moveMode is true, selecting a label will move the message (apply + archive)
// and then close the panel.
func (a *App) expandLabelsBrowseWithMode(messageID string, moveMode bool) {
	a.labelsExpanded = true
	input := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldWidth(30)
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)

	// Loader
	type labelItem struct {
		id, name string
		applied  bool
	}
	var all []labelItem
	var visible []labelItem
	var reload func(filter string)
	reload = func(filter string) {
		list.Clear()
		visible = visible[:0]
		for _, it := range all {
			if filter != "" && !strings.Contains(strings.ToLower(it.name), strings.ToLower(filter)) {
				continue
			}
			visible = append(visible, it)
			display := "○ " + it.name
			if it.applied {
				display = "✅ " + it.name
			}
			id := it.id
			name := it.name
			applied := it.applied
			list.AddItem(display, "Enter: toggle", 0, func() {
				if !moveMode {
					a.toggleLabelForMessage(messageID, id, name, applied, func(newApplied bool, err error) {
						if err == nil {
							// Update local model then rerender
							for i := range all {
								if all[i].id == id {
									all[i].applied = newApplied
									break
								}
							}
							a.updateCachedMessageLabels(messageID, id, newApplied)
							a.updateMessageCacheLabels(messageID, name, newApplied)
							reload(strings.TrimSpace(input.GetText()))
							a.refreshMessageContent(messageID)
						}
					})
					return
				}
				// Move mode: aplicar etiqueta y archivar para todos los seleccionados (o el actual)
				go func() {
					// Construir conjunto de mensajes a mover
					idsToMove := []string{messageID}
					if a.bulkMode && len(a.selected) > 0 {
						idsToMove = idsToMove[:0]
						for sid := range a.selected {
							idsToMove = append(idsToMove, sid)
						}
					}
					// Aplicar etiqueta y archivar (Gmail ignora duplicados en ApplyLabel)
					failed := 0
					for _, mid := range idsToMove {
						if err := a.Client.ApplyLabel(mid, id); err != nil {
							failed++
						}
						if err := a.Client.ArchiveMessage(mid); err != nil {
							failed++
						}
					}
					// Actualizar UI y cerrar panel
					a.QueueUpdateDraw(func() {
						listView, ok := a.views["list"].(*tview.Table)
						if !ok {
							return
						}
						if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
							split.ResizeItem(a.labelsView, 0, 0)
						}
						a.labelsVisible = false
						a.labelsExpanded = false
						// Eliminar todos los movidos de la lista/ids/meta
						rm := make(map[string]struct{}, len(idsToMove))
						for _, mid := range idsToMove {
							rm[mid] = struct{}{}
						}
						i := 0
						for i < len(a.ids) {
							if _, ok := rm[a.ids[i]]; ok {
								a.ids = append(a.ids[:i], a.ids[i+1:]...)
								if i < len(a.messagesMeta) {
									a.messagesMeta = append(a.messagesMeta[:i], a.messagesMeta[i+1:]...)
								}
								if i < listView.GetRowCount() {
									listView.RemoveRow(i)
								}
								continue
							}
							i++
						}
						// Ajustar selección y contenido
						cur, _ := listView.GetSelection()
						if cur >= listView.GetRowCount() {
							cur = listView.GetRowCount() - 1
						}
						listView.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
						if cur >= 0 && cur < len(a.ids) {
							listView.Select(cur, 0)
							go a.showMessageWithoutFocus(a.ids[cur])
						} else if tv, ok := a.views["text"].(*tview.TextView); ok {
							tv.SetText("No messages")
							tv.ScrollToBeginning()
						}
						// Salir de bulk y quitar checkboxes
						a.selected = make(map[string]bool)
						a.bulkMode = false
						a.reformatListItems()
						a.setStatusPersistent("")
						a.SetFocus(a.views["list"])
						a.currentFocus = "list"
						a.updateFocusIndicators("list")
						if len(idsToMove) <= 1 && failed == 0 {
							a.showStatusMessage("📦 Moved to: " + name)
						} else {
							a.showStatusMessage(fmt.Sprintf("📦 Moved %d message(s) to %s", len(idsToMove), name))
						}
					})
				}()
			})
		}
	}

	go func() {
		msg, err := a.Client.GetMessage(messageID)
		if err != nil {
			a.showError("❌ Error loading message")
			return
		}
		labels, err := a.Client.ListLabels()
		if err != nil {
			a.showError("❌ Error loading labels")
			return
		}
		current := make(map[string]bool)
		for _, lid := range msg.LabelIds {
			current[lid] = true
		}
		filtered := a.filterAndSortLabels(labels)
		all = make([]labelItem, 0, len(filtered))
		for _, l := range filtered {
			all = append(all, labelItem{l.Id, l.Name, current[l.Id]})
		}

		a.QueueUpdateDraw(func() {
			input.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					if moveMode {
						// Close the panel completely
						a.labelsExpanded = false
						if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
							split.ResizeItem(a.labelsView, 0, 0)
						}
						a.labelsVisible = false
						if a.bulkMode {
							a.bulkMode = false
							a.selected = make(map[string]bool)
							a.reformatListItems()
							a.setStatusPersistent("")
						}
						a.SetFocus(a.views["list"])
						a.currentFocus = "list"
						a.updateFocusIndicators("list")
					} else {
						// back to quick view
						a.labelsExpanded = false
						if a.bulkMode {
							a.bulkMode = false
							a.selected = make(map[string]bool)
							a.reformatListItems()
							a.setStatusPersistent("")
						}
						a.populateLabelsQuickView(messageID)
					}
					return
				}
				if key == tcell.KeyEnter {
					// UX shortcut: if there is at least one visible result, apply/move the first one
					if len(visible) >= 1 {
						v := visible[0]
						if !moveMode {
							if !v.applied {
								a.toggleLabelForMessage(messageID, v.id, v.name, false, func(newApplied bool, err error) {
									if err == nil {
										for i := range all {
											if all[i].id == v.id {
												all[i].applied = newApplied
												break
											}
										}
										a.updateCachedMessageLabels(messageID, v.id, newApplied)
										a.updateMessageCacheLabels(messageID, v.name, newApplied)
										reload(strings.TrimSpace(input.GetText()))
										a.refreshMessageContent(messageID)
									}
								})
							} else {
								a.showStatusMessage("✔️ Label already applied: " + v.name)
							}
						} else {
							// Modo mover: reutiliza la misma lógica que el callback de la lista
							go func(id, name string) {
								idsToMove := []string{messageID}
								if a.bulkMode && len(a.selected) > 0 {
									idsToMove = idsToMove[:0]
									for sid := range a.selected {
										idsToMove = append(idsToMove, sid)
									}
								}
								failed := 0
								for _, mid := range idsToMove {
									if err := a.Client.ApplyLabel(mid, id); err != nil {
										failed++
									}
									if err := a.Client.ArchiveMessage(mid); err != nil {
										failed++
									}
								}
								a.QueueUpdateDraw(func() {
									listView, ok := a.views["list"].(*tview.Table)
									if !ok {
										return
									}
									if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
										split.ResizeItem(a.labelsView, 0, 0)
									}
									a.labelsVisible = false
									a.labelsExpanded = false
									rm := make(map[string]struct{}, len(idsToMove))
									for _, mid := range idsToMove {
										rm[mid] = struct{}{}
									}
									i := 0
									for i < len(a.ids) {
										if _, ok := rm[a.ids[i]]; ok {
											a.ids = append(a.ids[:i], a.ids[i+1:]...)
											if i < len(a.messagesMeta) {
												a.messagesMeta = append(a.messagesMeta[:i], a.messagesMeta[i+1:]...)
											}
											if i < listView.GetRowCount() {
												listView.RemoveRow(i)
											}
											continue
										}
										i++
									}
									cur, _ := listView.GetSelection()
									if cur >= listView.GetRowCount() {
										cur = listView.GetRowCount() - 1
									}
									listView.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
									if cur >= 0 && cur < len(a.ids) {
										listView.Select(cur, 0)
										go a.showMessageWithoutFocus(a.ids[cur])
									} else if tv, ok := a.views["text"].(*tview.TextView); ok {
										tv.SetText("No messages")
										tv.ScrollToBeginning()
									}
									a.selected = make(map[string]bool)
									a.bulkMode = false
									a.reformatListItems()
									a.setStatusPersistent("")
									a.SetFocus(a.views["list"])
									a.currentFocus = "list"
									a.updateFocusIndicators("list")
									if len(idsToMove) <= 1 && failed == 0 {
										a.showStatusMessage("📦 Moved to: " + name)
									} else {
										a.showStatusMessage(fmt.Sprintf("📦 Moved %d message(s) to %s", len(idsToMove), name))
									}
								})
							}(v.id, v.name)
						}
					}
					return
				}
			})
			input.SetChangedFunc(func(text string) { reload(strings.TrimSpace(text)) })
			input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyUp || e.Key() == tcell.KeyDown {
					// Redirect arrow keys to the list when in the search field
					a.SetFocus(list)
					a.currentFocus = "labels"
					a.updateFocusIndicators("labels")
					return e
				}
				return e
			})

			container := tview.NewFlex().SetDirection(tview.FlexRow)
			container.SetBorder(true)
			titleText := " 🏷️ › 🔎 Browse all labels… "
			if moveMode {
				count := 1
				if a.bulkMode && len(a.selected) > 0 {
					count = len(a.selected)
				}
				if count == 1 {
					titleText = " 📦 Move message to… "
				} else {
					titleText = fmt.Sprintf(" 📦 Move %d messages to… ", count)
				}
			}
			container.SetTitle(titleText)
			container.SetTitleColor(tcell.ColorYellow)
			container.AddItem(input, 3, 0, true)
			container.AddItem(list, 0, 1, true)
			// Footer hint (bottom-right)
			footer := tview.NewTextView().
				SetDynamicColors(true).
				SetTextAlign(tview.AlignRight)
			if moveMode {
				footer.SetText("Enter applies/moves the first visible match   •   ESC to cancel  ")
			} else {
				footer.SetText("Enter applies the first visible match   •   ESC to back  ")
			}
			footer.SetTextColor(tcell.ColorGray)
			container.AddItem(footer, 1, 0, false)

			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				split.RemoveItem(a.labelsView)
				a.labelsView = container
				split.AddItem(a.labelsView, 0, 1, true)
			}
			// ESC handling in list: back to quick view or close panel in move mode
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyEscape {
					a.labelsExpanded = false
					if moveMode {
						if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
							split.ResizeItem(a.labelsView, 0, 0)
						}
						a.labelsVisible = false
						if a.bulkMode {
							a.bulkMode = false
							a.selected = make(map[string]bool)
							a.reformatListItems()
							a.setStatusPersistent("")
						}
						a.SetFocus(a.views["list"])
						a.currentFocus = "list"
						a.updateFocusIndicators("list")
					} else {
						if a.bulkMode {
							a.bulkMode = false
							a.selected = make(map[string]bool)
							a.reformatListItems()
							a.setStatusPersistent("")
						}
						a.populateLabelsQuickView(messageID)
					}
					return nil
				}
				return e
			})
			reload("")
			a.SetFocus(input)
			a.currentFocus = "labels"
			a.updateFocusIndicators("labels")
		})
	}()
}

// openMovePanel opens the side panel directly in browse-all mode to move the message
func (a *App) openMovePanel() {
	messageID := a.getCurrentMessageID()
	if messageID == "" {
		a.showError("❌ No message selected")
		return
	}
	// Ensure panel is visible
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.labelsVisible = true
	a.currentFocus = "labels"
	a.updateFocusIndicators("labels")
	// Open browse in move mode
	a.expandLabelsBrowseWithMode(messageID, true)
}

// openMovePanelBulk opens the move panel in bulk mode with pluralized title
func (a *App) openMovePanelBulk() {
	// If nothing selected fallback to single
	if len(a.selected) == 0 {
		a.openMovePanel()
		return
	}
	// Ensure panel visible
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.labelsVisible = true
	a.currentFocus = "labels"
	a.updateFocusIndicators("labels")
	// Use any selected message to populate current labels; choose the current focus message if selected, else any
	mid := a.getCurrentMessageID()
	if mid == "" || !a.selected[mid] {
		for id := range a.selected {
			mid = id
			break
		}
	}
	// Reuse browse with moveMode; title inside will be adjusted by len(a.selected) later if needed
	a.expandLabelsBrowseWithMode(mid, true)
}

// addCustomLabelInline prompts for a name and applies/creates it
func (a *App) addCustomLabelInline(messageID string) {
	input := tview.NewInputField().
		SetLabel("Label name: ").
		SetFieldWidth(30)
	modal := tview.NewFlex().SetDirection(tview.FlexRow)
	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetBorder(true)
	title.SetText("Enter a label name | Enter=apply, ESC=cancel")
	modal.AddItem(title, 3, 0, false)
	modal.AddItem(input, 3, 0, true)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			a.populateLabelsQuickView(messageID)
			return
		}
		if key == tcell.KeyEnter {
			name := strings.TrimSpace(input.GetText())
			if name == "" {
				return
			}
			go func() {
				// Reuse label list to find existing
				labels, err := a.Client.ListLabels()
				if err != nil {
					a.showError("❌ Error loading labels")
					return
				}
				nameToID := make(map[string]string)
				for _, l := range labels {
					nameToID[l.Name] = l.Id
				}
				id, ok := nameToID[name]
				if !ok {
					// case-insensitive match
					for n, i := range nameToID {
						if strings.EqualFold(n, name) {
							id = i
							ok = true
							break
						}
					}
				}
				if !ok {
					created, err := a.Client.CreateLabel(name)
					if err != nil {
						a.showError("❌ Error creating label")
						return
					}
					id = created.Id
				}
				if err := a.Client.ApplyLabel(messageID, id); err != nil {
					a.showError("❌ Error applying label")
					return
				}
				a.updateCachedMessageLabels(messageID, id, true)
				a.showStatusMessage("✅ Applied: " + name)
				a.QueueUpdateDraw(func() { a.populateLabelsQuickView(messageID); a.refreshMessageContent(messageID) })
			}()
		}
	})

	a.QueueUpdateDraw(func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.RemoveItem(a.labelsView)
			a.labelsView = modal
			split.AddItem(a.labelsView, 0, 1, true)
		}
		a.SetFocus(input)
		a.currentFocus = "labels"
		a.updateFocusIndicators("text")
	})
}

// toggleLabelForMessage toggles a label asynchronously and invokes onDone when finished
func (a *App) toggleLabelForMessage(messageID, labelID, labelName string, isCurrentlyApplied bool, onDone func(newApplied bool, err error)) {
	go func() {
		if isCurrentlyApplied {
			if err := a.Client.RemoveLabel(messageID, labelID); err != nil {
				a.showError(fmt.Sprintf("❌ Error removing label %s: %v", labelName, err))
				onDone(isCurrentlyApplied, err)
				return
			}
			a.showStatusMessage(fmt.Sprintf("🏷️  Removed label: %s", labelName))
			onDone(false, nil)
			return
		}
		if err := a.Client.ApplyLabel(messageID, labelID); err != nil {
			a.showError(fmt.Sprintf("❌ Error applying label %s: %v", labelName, err))
			onDone(isCurrentlyApplied, err)
			return
		}
		a.showStatusMessage(fmt.Sprintf("🏷️  Applied label: %s", labelName))
		onDone(true, nil)
	}()
}

// showMessagesWithLabel shows messages that have a specific label
func (a *App) showMessagesWithLabel(labelID, labelName string) {
	// Search for messages with this label
	query := fmt.Sprintf("label:%s", labelName)
	messages, err := a.Client.SearchMessages(query, 50)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error searching messages with label %s: %v", labelName, err))
		return
	}

	// Create messages view for this label
	a.showMessagesForLabel(messages, labelName)
}

// showMessagesForLabel displays messages that have a specific label
func (a *App) showMessagesForLabel(messages []*gmailapi.Message, labelName string) {
	// Create messages list for this label
	messagesList := tview.NewList()
	messagesList.SetBorder(true)
	messagesList.SetTitle(fmt.Sprintf(" 📧 Messages with label: %s ", labelName))

	// Clear current IDs and populate with new messages
	a.ids = []string{}

	for i, msg := range messages {
		a.ids = append(a.ids, msg.Id)

		// Get message details
		message, err := a.Client.GetMessageWithContent(msg.Id)
		if err != nil {
			messagesList.AddItem(fmt.Sprintf("⚠️  Error loading message %d", i+1), "Failed to load", 0, nil)
			continue
		}

		subject := message.Subject
		if subject == "" {
			subject = "(No subject)"
		}

		// Format the display text
		displayText := fmt.Sprintf("%s", subject)
		secondaryText := fmt.Sprintf("From: %s | %s", message.From, formatRelativeTime(message.Date))

		messagesList.AddItem(displayText, secondaryText, 0, func() {
			// Show the selected message
			if len(a.ids) > 0 {
				go a.showMessage(a.ids[0])
			}
		})
	}

	if len(messages) == 0 {
		messagesList.AddItem("No messages with this label", "", 0, nil)
	}

	// Set up key bindings
	messagesList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			// Return to labels view
			a.Pages.SwitchToPage("labels")
			return nil
		case 't':
			// Toggle read/unread for selected message
			go a.toggleMarkReadUnread()
			return nil
		case 'd':
			// Trash selected message
			go a.trashSelected()
			return nil
		}
		return event
	})

	// Create the messages view page
	pageName := fmt.Sprintf("messages_%s", labelName)
	a.Pages.AddPage(pageName, messagesList, true, true)
	a.Pages.SwitchToPage(pageName)
	a.SetFocus(messagesList)
}

// createNewLabelFromView creates a new label from the labels view
func (a *App) createNewLabelFromView() {
	// Create input field for new label name
	inputField := tview.NewInputField().
		SetLabel("Label name: ").
		SetFieldWidth(30).
		SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
			return len(textToCheck) > 0 && len(textToCheck) < 50
		})

	// Create modal for new label
	modal := tview.NewFlex().SetDirection(tview.FlexRow)

	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetText("🏷️  Create New Label")
	title.SetTextColor(tcell.ColorYellow)
	title.SetBorder(true)

	instructions := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	instructions.SetText("Enter label name and press Enter | ESC to cancel")
	instructions.SetTextColor(tcell.ColorGray)

	modal.AddItem(title, 3, 0, false)
	modal.AddItem(inputField, 3, 0, true)
	modal.AddItem(instructions, 2, 0, false)

	// Handle input
	inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			labelName := strings.TrimSpace(inputField.GetText())
			if labelName != "" {
				go func() {
					_, err := a.Client.CreateLabel(labelName)
					if err != nil {
						a.showError(fmt.Sprintf("❌ Error creating label: %v", err))
						return
					}

					a.showStatusMessage(fmt.Sprintf("🏷️  Created label: %s", labelName))

					// Return to labels view and refresh
					a.QueueUpdateDraw(func() {
						a.Pages.SwitchToPage("labels")
						// Refresh the labels view
						go a.manageLabels()
					})
				}()
			}
		case tcell.KeyEscape:
			a.Pages.SwitchToPage("labels")
		}
	})

	// Add modal to pages
	a.Pages.AddPage("createLabel", modal, true, true)
	a.Pages.SwitchToPage("createLabel")
	a.SetFocus(inputField)
}

// deleteSelectedLabel deletes the selected label (placeholder for now)
func (a *App) deleteSelectedLabel(labelsList *tview.List) {
	a.showInfo("Delete label functionality not yet implemented")
}

// updateCachedMessageLabels updates the cached labels for a message ID
func (a *App) updateCachedMessageLabels(messageID, labelID string, applied bool) {
	// Find index
	var idx = -1
	for i, id := range a.ids {
		if id == messageID {
			idx = i
			break
		}
	}
	if idx < 0 || idx >= len(a.messagesMeta) || a.messagesMeta[idx] == nil {
		return
	}
	msg := a.messagesMeta[idx]
	if applied {
		// add if not exists
		exists := false
		for _, l := range msg.LabelIds {
			if l == labelID {
				exists = true
				break
			}
		}
		if !exists {
			msg.LabelIds = append(msg.LabelIds, labelID)
		}
	} else {
		// remove
		out := msg.LabelIds[:0]
		for _, l := range msg.LabelIds {
			if l != labelID {
				out = append(out, l)
			}
		}
		msg.LabelIds = out
	}
}

// updateMessageCacheLabels updates the cached full message labels (names) so the
// rendered header reflects changes without requiring a refetch.
func (a *App) updateMessageCacheLabels(messageID, labelName string, applied bool) {
	if m, ok := a.messageCache[messageID]; ok && m != nil {
		if applied {
			// Add if missing (case-insensitive)
			exists := false
			for _, ln := range m.Labels {
				if strings.EqualFold(ln, labelName) {
					exists = true
					break
				}
			}
			if !exists {
				m.Labels = append(m.Labels, labelName)
			}
		} else {
			// Remove if present
			out := m.Labels[:0]
			for _, ln := range m.Labels {
				if !strings.EqualFold(ln, labelName) {
					out = append(out, ln)
				}
			}
			m.Labels = out
		}
	}
}

// moveSelected opens the labels picker to choose a destination label, applies it, then archives the message
func (a *App) moveSelected() {
	// Get the current message ID
	messageID := a.getCurrentMessageID()
	if messageID == "" {
		a.showError("❌ No message selected")
		return
	}

	// Load available labels and message metadata
	labels, err := a.Client.ListLabels()
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error loading labels: %v", err))
		return
	}
	message, err := a.Client.GetMessage(messageID)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error getting message: %v", err))
		return
	}

	a.showMoveLabelsView(labels, message)
}

// showMoveLabelsView lets user choose a label to apply and then archives the message (move semantics)
func (a *App) showMoveLabelsView(labels []*gmailapi.Label, message *gmailapi.Message) {
	picker := tview.NewList().ShowSecondaryText(false)
	picker.SetBorder(true)
	picker.SetTitle(" 📦 Move to label ")

	// Build list of candidate labels with applied first
	curr := make(map[string]bool)
	for _, l := range message.LabelIds {
		curr[l] = true
	}
	applied, notApplied := a.partitionAndSortLabels(labels, curr)
	for _, label := range append(applied, notApplied...) {
		// Store values for closure
		labelID := label.Id
		labelName := label.Name
		picker.AddItem(labelName, "", 0, func() {
			go func() {
				// Apply label if not already present
				has := false
				for _, l := range message.LabelIds {
					if l == labelID {
						has = true
						break
					}
				}
				if !has {
					if err := a.Client.ApplyLabel(message.Id, labelID); err != nil {
						a.showError(fmt.Sprintf("❌ Error applying label: %v", err))
						return
					}
					// Update cache
					a.updateCachedMessageLabels(message.Id, labelID, true)
				}
				// Archive (remove INBOX)
				if err := a.Client.ArchiveMessage(message.Id); err != nil {
					a.showError(fmt.Sprintf("❌ Error archiving: %v", err))
					return
				}
				a.showStatusMessage(fmt.Sprintf("📦 Moved to: %s", labelName))

				// Remove from current list (safe removal pattern) since we show INBOX only
				a.QueueUpdateDraw(func() {
					list, ok := a.views["list"].(*tview.List)
					if !ok {
						a.Pages.SwitchToPage("main")
						a.restoreFocusAfterModal()
						return
					}
					count := list.GetItemCount()
					if count == 0 {
						a.Pages.SwitchToPage("main")
						a.restoreFocusAfterModal()
						return
					}
					// Determine index of the moved message
					removeIndex := -1
					for i, id := range a.ids {
						if id == message.Id {
							removeIndex = i
							break
						}
					}
					if removeIndex < 0 || removeIndex >= count {
						removeIndex = list.GetCurrentItem()
						if removeIndex < 0 || removeIndex >= count {
							removeIndex = 0
						}
					}
					// Preselect a different index to avoid tview internal -1 during removal
					var next = -1
					if count > 1 {
						pre := removeIndex - 1
						if removeIndex == 0 {
							pre = 1
						}
						if pre < 0 {
							pre = 0
						}
						if pre >= count {
							pre = count - 1
						}
						list.SetCurrentItem(pre)
						next = pre
					}
					// Update caches using removeIndex
					if removeIndex >= 0 && removeIndex < len(a.ids) {
						a.ids = append(a.ids[:removeIndex], a.ids[removeIndex+1:]...)
					}
					if removeIndex >= 0 && removeIndex < len(a.messagesMeta) {
						a.messagesMeta = append(a.messagesMeta[:removeIndex], a.messagesMeta[removeIndex+1:]...)
					}
					// Visual removal
					if count == 1 {
						list.Clear()
						next = -1
					} else {
						if removeIndex >= 0 && removeIndex < list.GetItemCount() {
							list.RemoveItem(removeIndex)
						}
						if next >= 0 && next < list.GetItemCount() {
							list.SetCurrentItem(next)
						}
					}
					list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
					// Update message content pane similar to trash/archive
					if text, ok := a.views["text"].(*tview.TextView); ok {
						if next >= 0 && next < len(a.ids) {
							go a.showMessageWithoutFocus(a.ids[next])
						} else {
							text.SetText("No messages")
							text.ScrollToBeginning()
						}
					}
					// Return to main
					a.Pages.SwitchToPage("main")
					a.restoreFocusAfterModal()
				})
			}()
		})
	}

	// Basic keys
	picker.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			a.Pages.SwitchToPage("main")
			a.restoreFocusAfterModal()
			return nil
		}
		return event
	})

	// Container view
	v := tview.NewFlex().SetDirection(tview.FlexRow)
	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetBorder(true)
	title.SetText("Select destination label and press Enter. ESC to cancel")
	v.AddItem(title, 3, 0, false)
	v.AddItem(picker, 0, 1, true)

	a.Pages.AddPage("moveLabels", v, true, true)
	a.Pages.SwitchToPage("moveLabels")
	if picker.GetItemCount() > 0 {
		picker.SetCurrentItem(0)
	}
	a.SetFocus(picker)
}

// filterAndSortLabels filters out system labels and returns a name-sorted slice
func (a *App) filterAndSortLabels(labels []*gmailapi.Label) []*gmailapi.Label {
	filtered := make([]*gmailapi.Label, 0, len(labels))
	for _, l := range labels {
		if strings.HasPrefix(l.Id, "CATEGORY_") || l.Id == "INBOX" || l.Id == "SENT" || l.Id == "DRAFT" ||
			l.Id == "SPAM" || l.Id == "TRASH" || l.Id == "CHAT" || (strings.HasSuffix(l.Id, "_STARRED") && l.Id != "STARRED") {
			continue
		}
		filtered = append(filtered, l)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
	})
	return filtered
}

// partitionAndSortLabels returns two sorted slices: labels applied to current and the rest
func (a *App) partitionAndSortLabels(labels []*gmailapi.Label, current map[string]bool) ([]*gmailapi.Label, []*gmailapi.Label) {
	filtered := a.filterAndSortLabels(labels)
	applied := make([]*gmailapi.Label, 0)
	notApplied := make([]*gmailapi.Label, 0)
	for _, l := range filtered {
		if current[l.Id] {
			applied = append(applied, l)
		} else {
			notApplied = append(notApplied, l)
		}
	}
	// Already sorted by name from filterAndSortLabels; preserve order
	return applied, notApplied
}

// showAllLabelsPicker shows a list of all actionable labels to apply one to the message
func (a *App) showAllLabelsPicker(messageID string) {
	labels, err := a.Client.ListLabels()
	if err != nil {
		a.showError("❌ Error loading labels")
		return
	}
	// Get current message labels to mark applied ones
	msg, err := a.Client.GetMessage(messageID)
	if err != nil {
		a.showError("❌ Error loading message")
		return
	}
	current := make(map[string]bool, len(msg.LabelIds))
	for _, lid := range msg.LabelIds {
		current[lid] = true
	}
	// Build sorted actionable labels with applied first
	applied, notApplied := a.partitionAndSortLabels(labels, current)
	all := append(applied, notApplied...)

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true)
	list.SetTitle(" 🗂️  All Labels ")

	// Map name -> id
	nameToID := make(map[string]string, len(all))
	for _, l := range all {
		nameToID[l.Name] = l.Id
	}

	for _, l := range all {
		lbl := l.Name
		icon := "○ "
		if current[l.Id] {
			icon = "✅ "
		}
		display := icon + lbl
		list.AddItem(display, "", 0, func() {
			if id, ok := nameToID[lbl]; ok {
				a.applyLabelAndRefresh(messageID, id, lbl)
				a.showStatusMessage("✅ Applied: " + lbl)
				a.Pages.SwitchToPage("main")
				a.restoreFocusAfterModal()
			}
		})
	}

	list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape {
			a.Pages.SwitchToPage("aiLabelSuggestions")
			return nil
		}
		return e
	})

	v := tview.NewFlex().SetDirection(tview.FlexRow)
	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetBorder(true)
	title.SetText("Select a label to apply | Enter=apply, ESC=back")
	v.AddItem(title, 3, 0, false)
	v.AddItem(list, 0, 1, true)
	a.Pages.AddPage("aiAllLabels", v, true, true)
	a.Pages.SwitchToPage("aiAllLabels")
	if list.GetItemCount() > 0 {
		list.SetCurrentItem(0)
	}
	a.SetFocus(list)
}

// applyLabelAndRefresh applies a label to a message and refreshes its content
func (a *App) applyLabelAndRefresh(messageID, labelID, labelName string) {
	// We assume that we want to apply (not toggle off), so pass isCurrentlyApplied=false
	a.toggleLabelForMessage(messageID, labelID, labelName, false, func(newApplied bool, err error) {
		if err != nil {
			return
		}
		if newApplied {
			// Keep meta cache consistent
			a.updateCachedMessageLabels(messageID, labelID, true)
			// Refresh content from server to avoid desync
			a.refreshMessageContent(messageID)
		}
	})
}

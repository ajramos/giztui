package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// openAttachmentPicker shows a picker for selecting and downloading attachments from the current message
func (a *App) openAttachmentPicker() {
	messageID := a.GetCurrentMessageID()
	if messageID == "" {
		a.GetErrorHandler().ShowError(a.ctx, "No message selected")
		return
	}

	// Get attachment service
	_, _, _, _, _, _, _, _, _, attachmentService, _ := a.GetServices()
	if attachmentService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Attachment service not available")
		return
	}

	// Create picker UI similar to links picker
	input := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldWidth(30).
		SetLabelColor(a.getTitleColor()).
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetFieldTextColor(tview.Styles.PrimaryTextColor)
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)

	type attachmentItem struct {
		index        int
		attachmentID string
		filename     string
		mimeType     string
		size         int64
		type_        string
		inline       bool
	}

	var all []attachmentItem
	var visible []attachmentItem

	// Reload function for filtering
	reload := func(filter string) {
		list.Clear()
		visible = visible[:0]
		for _, item := range all {
			if filter != "" {
				filterLower := strings.ToLower(filter)
				if !strings.Contains(strings.ToLower(item.filename), filterLower) &&
					!strings.Contains(strings.ToLower(item.mimeType), filterLower) {
					// Check for special filters
					if strings.HasPrefix(filterLower, "type:") {
						typeFilter := strings.TrimPrefix(filterLower, "type:")
						if !strings.Contains(strings.ToLower(item.type_), typeFilter) {
							continue
						}
					} else if strings.HasPrefix(filterLower, "size:") {
						sizeFilter := strings.TrimPrefix(filterLower, "size:")
						// Simple size filtering (could be enhanced)
						if !strings.Contains(strings.ToLower(formatFileSize(item.size)), sizeFilter) {
							continue
						}
					} else {
						continue
					}
				}
			}
			visible = append(visible, item)

			// Category icon based on attachment type
			icon := a.getAttachmentIcon(item.type_)

			// Format: [n] filename.ext (size) - type
			sizeStr := formatFileSize(item.size)
			display := fmt.Sprintf("%s [%d] %s", icon, item.index, item.filename)
			if len(display) > 50 {
				display = display[:47] + "..."
			}

			// Show size and type in secondary text
			secondary := ""
			if sizeStr != "" && item.mimeType != "" {
				secondary = fmt.Sprintf("%s - %s", sizeStr, item.mimeType)
			} else if sizeStr != "" {
				secondary = sizeStr
			} else if item.mimeType != "" {
				// Always show MIME type, even if size is unknown
				secondary = item.mimeType
			}

			// If no secondary text, show a default indicator
			if secondary == "" {
				secondary = "attachment"
			}

			// Capture variables for closure
			attachmentID := item.attachmentID
			filename := item.filename
			fileType := item.type_

			list.AddItem(display, secondary, 0, func() {
				// Close picker first (synchronous)
				a.closeAttachmentPicker()

				// Download and open attachment asynchronously
				go func() {
					// Show status message asynchronously
					go func() {
						a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Downloading: %s", filename))
					}()

					// Download the attachment
					a.downloadAndOpenAttachment(messageID, attachmentID, filename, fileType)
				}()
			})
		}

		// Show count in input label
		if len(all) > 0 {
			input.SetLabel(fmt.Sprintf("🔍 Search (%d/%d): ", len(visible), len(all)))
		} else {
			input.SetLabel("🔍 Search: ")
		}
	}

	// Load attachments in background
	go func() {
		attachments, err := attachmentService.GetMessageAttachments(a.ctx, messageID)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load attachments: %v", err))
			return
		}

		if len(attachments) == 0 {
			a.GetErrorHandler().ShowInfo(a.ctx, "No attachments found in this message")
			return
		}

		// Convert to attachmentItem format
		all = make([]attachmentItem, 0, len(attachments))
		for _, attachment := range attachments {
			// Skip inline images without meaningful filenames (like embedded content)
			if attachment.Inline && attachment.Filename == "" {
				continue
			}

			all = append(all, attachmentItem{
				index:        attachment.Index,
				attachmentID: attachment.AttachmentID,
				filename:     attachment.Filename,
				mimeType:     attachment.MimeType,
				size:         attachment.Size,
				type_:        attachment.Type,
				inline:       attachment.Inline,
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
				// Support direct number input for quick access
				if e.Rune() >= '1' && e.Rune() <= '9' {
					num := int(e.Rune() - '0')
					if num <= len(visible) && num > 0 {
						item := visible[num-1]
						// Close picker first (synchronous)
						a.closeAttachmentPicker()

						// Download attachment asynchronously
						go func() {
							// Show status message asynchronously
							go func() {
								a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Downloading: %s", item.filename))
							}()

							// Download the attachment
							a.downloadAndOpenAttachment(messageID, item.attachmentID, item.filename, item.type_)
						}()
						return nil
					}
				}
				return e
			})

			// Handle enter in input field (select first match)
			input.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					a.closeAttachmentPicker()
					return
				}
				if key == tcell.KeyEnter {
					if len(visible) > 0 {
						item := visible[0]
						// Close picker first (synchronous)
						a.closeAttachmentPicker()

						// Download attachment asynchronously
						go func() {
							// Show status message asynchronously
							go func() {
								a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Downloading: %s", item.filename))
							}()

							// Download the attachment
							a.downloadAndOpenAttachment(messageID, item.attachmentID, item.filename, item.type_)
						}()
					}
				}
			})

			// Create container
			container := tview.NewFlex().SetDirection(tview.FlexRow)
			container.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
			container.SetBorder(true)
			container.SetTitle(" 📎 Attachments in Message ")
			container.SetTitleColor(a.GetComponentColors("attachments").Title.Color())
			container.AddItem(input, 3, 0, true)
			container.AddItem(list, 0, 1, true)

			// Footer with instructions
			footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
			footer.SetText(" Enter/1-9 to download | Ctrl+S to save as | Esc to cancel ")
			footer.SetTextColor(a.GetComponentColors("attachments").Text.Color()) // Standardized footer color
			container.AddItem(footer, 1, 0, false)

			// Handle navigation between input and list
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyUp && list.GetCurrentItem() == 0 {
					a.SetFocus(input)
					return nil
				}
				if e.Key() == tcell.KeyEscape {
					a.closeAttachmentPicker()
					return nil
				}
				// Show attachment details in status when navigating
				if e.Key() == tcell.KeyDown || e.Key() == tcell.KeyUp {
					// Small delay to let list update selection first
					go func() {
						// Get current selection after navigation
						currentItem := list.GetCurrentItem()
						if currentItem >= 0 && currentItem < len(visible) {
							item := visible[currentItem]
							details := fmt.Sprintf("%s - %s - %s", item.filename, formatFileSize(item.size), item.mimeType)
							// Show details in status bar asynchronously
							go func() {
								a.GetErrorHandler().ShowInfo(a.ctx, details)
							}()
						}
					}()
				}
				// Support save as with Ctrl+S
				if e.Key() == tcell.KeyCtrlS {
					currentItem := list.GetCurrentItem()
					if currentItem >= 0 && currentItem < len(visible) {
						item := visible[currentItem]
						a.saveAttachmentAs(messageID, item.attachmentID, item.filename)
					}
					return nil
				}
				// Quick number access
				if e.Rune() >= '1' && e.Rune() <= '9' {
					num := int(e.Rune() - '0')
					if num <= len(visible) && num > 0 {
						item := visible[num-1]
						// Close picker first (synchronous)
						a.closeAttachmentPicker()

						// Download attachment asynchronously
						go func() {
							// Show status message asynchronously
							go func() {
								a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Downloading: %s", item.filename))
							}()

							// Download the attachment
							a.downloadAndOpenAttachment(messageID, item.attachmentID, item.filename, item.type_)
						}()
						return nil
					}
				}
				return e
			})

			// Add to content split like other pickers
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				if a.labelsView != nil {
					split.RemoveItem(a.labelsView)
				}
				a.labelsView = container
				split.AddItem(a.labelsView, 0, 1, true)
				split.ResizeItem(a.labelsView, 0, 1)
			}
			a.SetFocus(input)
			a.currentFocus = "labels" // Reuse labels focus state for consistency
			a.updateFocusIndicators("labels")
			a.labelsVisible = true // Reuse labels visibility state

			// Initial load
			reload("")

			// Set first item as selected for better UX
			if list.GetItemCount() > 0 {
				list.SetCurrentItem(0)
				// Show first attachment details in status bar
				if len(visible) > 0 {
					details := fmt.Sprintf("%s - %s - %s", visible[0].filename, formatFileSize(visible[0].size), visible[0].mimeType)
					go func() {
						a.GetErrorHandler().ShowInfo(a.ctx, details)
					}()
				}
			}
		})
	}()
}

// closeAttachmentPicker closes the attachment picker and restores focus
func (a *App) closeAttachmentPicker() {
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 0)
	}
	a.labelsVisible = false
	// Restore focus to text view
	if text, ok := a.views["text"].(*tview.TextView); ok {
		a.SetFocus(text)
		a.currentFocus = "text"
		a.updateFocusIndicators("text")
	}
}

// downloadAndOpenAttachment downloads and optionally opens an attachment
func (a *App) downloadAndOpenAttachment(messageID, attachmentID, filename, fileType string) {
	// Get attachment service
	_, _, _, _, _, _, _, _, _, attachmentService, _ := a.GetServices()
	if attachmentService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Attachment service not available")
		return
	}

	// Download the attachment with the correct filename
	downloadPath, err := attachmentService.DownloadAttachmentWithFilename(a.ctx, messageID, attachmentID, "", filename)
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to download attachment: %v", err))
		return
	}

	// Show success message
	displayName := filepath.Base(downloadPath)
	a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Downloaded: %s", displayName))

	// Optionally open the file (based on config)
	if a.Config != nil && a.Config.Attachments.AutoOpen {
		if err := attachmentService.OpenAttachment(a.ctx, downloadPath); err != nil {
			// Don't show error for opening, just log it
			if a.logger != nil {
				a.logger.Printf("Failed to auto-open attachment: %v", err)
			}
		} else {
			a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Opened: %s", displayName))
		}
	}
}

// saveAttachmentAs opens input panel for custom save location
func (a *App) saveAttachmentAs(messageID, attachmentID, filename string) {
	// Close current attachment picker
	a.closeAttachmentPicker()

	// Get attachment service for default path
	_, _, _, _, _, _, _, _, _, attachmentService, _ := a.GetServices()
	defaultPath := ""
	if attachmentService != nil {
		defaultPath = filepath.Join(attachmentService.GetDefaultDownloadPath(), filename)
	}

	// Create title
	title := tview.NewTextView().
		SetText(fmt.Sprintf("Save Attachment: %s", filename)).
		SetTextAlign(tview.AlignCenter).
		SetTextColor(a.GetComponentColors("attachments").Title.Color())

	// Create path input with default value
	pathInput := tview.NewInputField().
		SetLabel("Save to: ").
		SetText(defaultPath).
		SetFieldWidth(0).
		SetLabelColor(a.GetComponentColors("attachments").Text.Color()).
		SetFieldBackgroundColor(a.GetComponentColors("attachments").Background.Color()).
		SetFieldTextColor(a.GetComponentColors("attachments").Text.Color())

	// Create instructions
	instructions := tview.NewTextView().
		SetText("Press Enter to save | Tab to edit | Esc to cancel").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(a.GetComponentColors("attachments").Accent.Color())

	// Handle input
	pathInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			a.closeSaveAsPanel()
			return
		}
		if key == tcell.KeyEnter {
			customPath := strings.TrimSpace(pathInput.GetText())
			a.closeSaveAsPanel()

			// Download with custom path
			go func() {
				go func() {
					a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Saving '%s'...", filename))
				}()

				if attachmentService != nil {
					downloadPath, err := attachmentService.DownloadAttachmentWithFilename(a.ctx, messageID, attachmentID, customPath, filename)
					if err != nil {
						a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to save: %v", err))
						return
					}

					displayName := filepath.Base(downloadPath)
					a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Saved: %s", displayName))
				}
			}()
		}
	})

	// Create container
	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(a.GetComponentColors("attachments").Background.Color())
	container.SetBorder(true)
	container.SetTitle(" 💾 Save Attachment As ")
	container.SetTitleColor(a.GetComponentColors("attachments").Title.Color())

	// Add components
	container.AddItem(title, 2, 0, false)
	container.AddItem(pathInput, 3, 0, true)
	container.AddItem(instructions, 1, 0, false)

	// Add to content split
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = container
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}

	a.SetFocus(pathInput)
	a.currentFocus = "labels"
	a.updateFocusIndicators("labels")
	a.labelsVisible = true
}

// closeSaveAsPanel closes the save as panel and restores focus
func (a *App) closeSaveAsPanel() {
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 0)
	}
	a.labelsVisible = false
	// Restore focus to text view
	if text, ok := a.views["text"].(*tview.TextView); ok {
		a.SetFocus(text)
		a.currentFocus = "text"
		a.updateFocusIndicators("text")
	}
}

// getAttachmentIcon returns an icon based on attachment type
func (a *App) getAttachmentIcon(attachmentType string) string {
	switch attachmentType {
	case "image":
		return "🖼️"
	case "document":
		return "📄"
	case "spreadsheet":
		return "📊"
	case "presentation":
		return "📽️"
	case "archive":
		return "📦"
	case "audio":
		return "🎵"
	case "video":
		return "🎬"
	case "calendar":
		return "📅"
	default:
		return "📎"
	}
}

// formatFileSize formats file size in human-readable format
func formatFileSize(size int64) string {
	if size == 0 {
		return "size unknown"
	}

	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	suffixes := []string{"B", "KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(size)/float64(div), suffixes[exp+1])
}

package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// openLinkPicker shows a picker for selecting and opening links from the current message
func (a *App) openLinkPicker() {
	messageID := a.GetCurrentMessageID()
	if messageID == "" {
		a.GetErrorHandler().ShowError(a.ctx, "No message selected")
		return
	}

	// Get link service
	_, _, _, _, _, _, _, linkService := a.GetServices()
	if linkService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Link service not available")
		return
	}

	// Create picker UI similar to prompts
	input := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldWidth(30)
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)

	type linkItem struct {
		index int
		url   string
		text  string
		type_ string
	}

	var all []linkItem
	var visible []linkItem

	// Reload function for filtering
	reload := func(filter string) {
		list.Clear()
		visible = visible[:0]
		for _, item := range all {
			if filter != "" {
				filterLower := strings.ToLower(filter)
				if !strings.Contains(strings.ToLower(item.url), filterLower) &&
					!strings.Contains(strings.ToLower(item.text), filterLower) {
					// Check for special filters
					if strings.HasPrefix(filterLower, "domain:") {
						domain := strings.TrimPrefix(filterLower, "domain:")
						if !strings.Contains(strings.ToLower(item.url), domain) {
							continue
						}
					} else if strings.HasPrefix(filterLower, "type:") {
						typeFilter := strings.TrimPrefix(filterLower, "type:")
						if !strings.Contains(strings.ToLower(item.type_), typeFilter) {
							continue
						}
					} else {
						continue
					}
				}
			}
			visible = append(visible, item)

			// Category icon based on link type
			icon := "🔗"
			switch item.type_ {
			case "email":
				icon = "📧"
			case "file":
				icon = "📁"
			case "external":
				icon = "🌐"
			case "html":
				icon = "🔗"
			default:
				icon = "🔗"
			}

			// Format: [n] Text - Domain
			display := fmt.Sprintf("%s [%d] %s", icon, item.index, item.text)
			if len(display) > 50 {
				display = display[:47] + "..."
			}

			// Show domain in secondary text if URL is different from text
			secondary := ""
			if item.url != item.text {
				if len(item.url) > 40 {
					secondary = item.url[:37] + "..."
				} else {
					secondary = item.url
				}
			}

			// Capture variables for closure
			linkURL := item.url
			linkText := item.text

			list.AddItem(display, secondary, 0, func() {
				// Close picker first (synchronous)
				a.closeLinkPicker()
				
				// Open link asynchronously
				go func() {
					// Show status message asynchronously
					go func() {
						a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Opening: %s", linkURL))
					}()
					
					// Open the link
					a.openSelectedLink(linkURL, linkText)
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

	// Load links in background
	go func() {
		links, err := linkService.GetMessageLinks(a.ctx, messageID)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load links: %v", err))
			return
		}

		if len(links) == 0 {
			a.GetErrorHandler().ShowInfo(a.ctx, "No links found in this message")
			return
		}

		// Convert to linkItem format
		all = make([]linkItem, 0, len(links))
		for _, link := range links {
			// Skip empty URLs
			if strings.TrimSpace(link.URL) == "" {
				continue
			}
			
			text := link.Text
			if text == "" || text == link.URL {
				// Extract domain for display if no meaningful text
				if strings.Contains(link.URL, "://") {
					parts := strings.Split(link.URL, "://")
					if len(parts) > 1 {
						domain := strings.Split(parts[1], "/")[0]
						text = domain
					}
				}
			}
			
			all = append(all, linkItem{
				index: link.Index,
				url:   link.URL,
				text:  text,
				type_: link.Type,
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
						a.closeLinkPicker()
						
						// Open link asynchronously
						go func() {
							// Show status message asynchronously
							go func() {
								a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Opening: %s", item.url))
							}()
							
							// Open the link
							a.openSelectedLink(item.url, item.text)
						}()
						return nil
					}
				}
				return e
			})

			// Handle enter in input field (select first match)
			input.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					a.closeLinkPicker()
					return
				}
				if key == tcell.KeyEnter {
					if len(visible) > 0 {
						item := visible[0]
						// Close picker first (synchronous)
						a.closeLinkPicker()
						
						// Open link asynchronously
						go func() {
							// Show status message asynchronously
							go func() {
								a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Opening: %s", item.url))
							}()
							
							// Open the link
							a.openSelectedLink(item.url, item.text)
						}()
					}
				}
			})

			// Create container
			container := tview.NewFlex().SetDirection(tview.FlexRow)
			container.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
			container.SetBorder(true)
			container.SetTitle(" 🔗 Links in Message ")
			container.SetTitleColor(tcell.ColorBlue)
			container.AddItem(input, 3, 0, true)
			container.AddItem(list, 0, 1, true)

			// Footer with instructions
			footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
			footer.SetText(" Enter/1-9 to open | Ctrl+Y to copy | Esc to cancel ")
			footer.SetTextColor(tcell.ColorGray)
			container.AddItem(footer, 1, 0, false)

			// Handle navigation between input and list
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyUp && list.GetCurrentItem() == 0 {
					a.SetFocus(input)
					return nil
				}
				if e.Key() == tcell.KeyEscape {
					a.closeLinkPicker()
					return nil
				}
				// Show URL in status when navigating
				if e.Key() == tcell.KeyDown || e.Key() == tcell.KeyUp {
					// Small delay to let list update selection first
					go func() {
						// Get current selection after navigation
						currentItem := list.GetCurrentItem()
						if currentItem >= 0 && currentItem < len(visible) {
							item := visible[currentItem]
							// Show URL in status bar asynchronously
							go func() {
								a.GetErrorHandler().ShowInfo(a.ctx, item.url)
							}()
						}
					}()
				}
				// Support copying URL with Ctrl+Y
				if e.Key() == tcell.KeyCtrlY {
					currentItem := list.GetCurrentItem()
					if currentItem >= 0 && currentItem < len(visible) {
						item := visible[currentItem]
						a.copyToClipboard(item.url)
						// Show success message asynchronously
						go func() {
							a.GetErrorHandler().ShowSuccess(a.ctx, "Link copied to clipboard")
						}()
					}
					return nil
				}
				// Quick number access
				if e.Rune() >= '1' && e.Rune() <= '9' {
					num := int(e.Rune() - '0')
					if num <= len(visible) && num > 0 {
						item := visible[num-1]
						// Close picker first (synchronous)
						a.closeLinkPicker()
						
						// Open link asynchronously
						go func() {
							// Show status message asynchronously
							go func() {
								a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Opening: %s", item.url))
							}()
							
							// Open the link
							a.openSelectedLink(item.url, item.text)
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
			a.currentFocus = "labels"  // Reuse labels focus state for consistency
			a.updateFocusIndicators("labels")
			a.labelsVisible = true // Reuse labels visibility state

			// Initial load
			reload("")
			
			// Set first item as selected for better UX
			if list.GetItemCount() > 0 {
				list.SetCurrentItem(0)
				// Show first URL in status bar
				if len(visible) > 0 {
					go func() {
						a.GetErrorHandler().ShowInfo(a.ctx, visible[0].url)
					}()
				}
			}
		})
	}()
}

// closeLinkPicker closes the link picker and restores focus
func (a *App) closeLinkPicker() {
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

// openSelectedLink opens a link using the link service
func (a *App) openSelectedLink(url, text string) {
	// Get link service
	_, _, _, _, _, _, _, linkService := a.GetServices()
	if linkService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Link service not available")
		return
	}

	// Open the link
	if err := linkService.OpenLink(a.ctx, url); err != nil {
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to open link: %v", err))
		return
	}

	// Show success message
	displayText := text
	if displayText == url || displayText == "" {
		if len(url) > 50 {
			displayText = url[:47] + "..."
		} else {
			displayText = url
		}
	}
	a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Opened: %s", displayText))
}

// copyToClipboard copies text to clipboard using system commands
func (a *App) copyToClipboard(text string) {
	// Cross-platform clipboard copy implementation
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel as fallback
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			if a.logger != nil {
				a.logger.Printf("No clipboard utility found (xclip/xsel required on Linux)")
			}
			return
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		if a.logger != nil {
			a.logger.Printf("Clipboard not supported on platform: %s", runtime.GOOS)
		}
		return
	}
	
	if cmd != nil {
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err != nil {
			if a.logger != nil {
				a.logger.Printf("Failed to copy to clipboard: %v", err)
			}
		} else {
			if a.logger != nil {
				a.logger.Printf("Copied to clipboard: %s", text)
			}
		}
	}
}
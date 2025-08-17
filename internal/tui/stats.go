package tui

import (
	"fmt"
	"strings"

	"github.com/ajramos/gmail-tui/internal/services"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// showUsageStats displays prompt usage statistics
func (a *App) showUsageStats() {
	// Get prompt service
	_, _, _, _, _, promptService, _, _ := a.GetServices()
	if promptService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		return
	}

	// Get usage statistics
	stats, err := promptService.GetUsageStats(a.ctx)
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to get usage stats: %v", err))
		return
	}

	// Create stats view
	a.QueueUpdateDraw(func() {
		a.createUsageStatsModal(stats)
	})
}

// createUsageStatsModal creates and displays the usage statistics modal
func (a *App) createUsageStatsModal(stats *services.UsageStats) {
	// Create the stats view
	statsView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetScrollable(true)

	statsView.SetBorder(true)
	statsView.SetTitle(" 📊 Prompt Usage Statistics ")
	statsView.SetTitleColor(tcell.ColorYellow)

	// Format the statistics content
	var content strings.Builder
	
	// Summary section
	content.WriteString("[yellow]📊 USAGE SUMMARY[white]\n\n")
	content.WriteString(fmt.Sprintf("[blue]Total Prompt Uses:[white] %d\n", stats.TotalUsage))
	content.WriteString(fmt.Sprintf("[blue]Active Prompts:[white] %d\n", stats.UniquePrompts))
	content.WriteString(fmt.Sprintf("[blue]Favorite Prompts:[white] %d\n", len(stats.FavoritePrompts)))
	if !stats.LastUsed.IsZero() {
		content.WriteString(fmt.Sprintf("[blue]Last Used:[white] %s\n", stats.LastUsed.Format("2006-01-02 15:04")))
	}
	content.WriteString("\n")

	// Top prompts section
	if len(stats.TopPrompts) > 0 {
		content.WriteString("[yellow]🏆 TOP PROMPTS[white]\n\n")
		for i, prompt := range stats.TopPrompts {
			icon := "📝"
			switch prompt.Category {
			case "bulk_analysis":
				icon = "🚀"
			case "summary":
				icon = "📄"
			case "analysis":
				icon = "📊"
			case "reply":
				icon = "💬"
			}

			favoriteIcon := ""
			if prompt.IsFavorite {
				favoriteIcon = " ⭐"
			}

			content.WriteString(fmt.Sprintf("[green]%d.[white] %s %s%s\n", i+1, icon, prompt.Name, favoriteIcon))
			content.WriteString(fmt.Sprintf("    [dim]Uses: %d | Category: %s | Last: %s[white]\n", 
				prompt.UsageCount, prompt.Category, prompt.LastUsed))
			content.WriteString("\n")
		}
	} else {
		content.WriteString("[yellow]🏆 TOP PROMPTS[white]\n\n")
		content.WriteString("[dim]No prompt usage recorded yet.[white]\n")
		content.WriteString("[dim]Start using prompts to see statistics here![white]\n\n")
	}

	// Favorites section (if different from top)
	if len(stats.FavoritePrompts) > 0 && len(stats.FavoritePrompts) != len(stats.TopPrompts) {
		content.WriteString("[yellow]⭐ FAVORITE PROMPTS[white]\n\n")
		for _, prompt := range stats.FavoritePrompts {
			icon := "📝"
			switch prompt.Category {
			case "bulk_analysis":
				icon = "🚀"
			case "summary":
				icon = "📄"
			case "analysis":
				icon = "📊"
			case "reply":
				icon = "💬"
			}

			content.WriteString(fmt.Sprintf("• %s %s\n", icon, prompt.Name))
			content.WriteString(fmt.Sprintf("  [dim]Uses: %d | Category: %s[white]\n", 
				prompt.UsageCount, prompt.Category))
		}
		content.WriteString("\n")
	}

	// Help text
	content.WriteString("[dim]Press Esc to close | Use :stats to refresh[white]")

	statsView.SetText(content.String())

	// Handle input
	statsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			a.closeUsageStatsModal()
			return nil
		}
		return event
	})

	// Create modal using Pages
	modal := tview.NewFlex().SetDirection(tview.FlexRow)
	modal.AddItem(nil, 0, 1, false)
	modal.AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(statsView, 80, 0, true).
		AddItem(nil, 0, 1, false), 0, 3, true)
	modal.AddItem(nil, 0, 1, false)

	// Add to pages and show
	a.Pages.AddPage("usageStats", modal, true, true)
	a.Pages.SwitchToPage("usageStats")
	a.SetFocus(statsView)
}

// closeUsageStatsModal closes the usage statistics modal
func (a *App) closeUsageStatsModal() {
	a.Pages.RemovePage("usageStats")
	a.Pages.SwitchToPage("main")
	a.restoreFocusAfterModal()
}
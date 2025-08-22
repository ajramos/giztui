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
	_, _, _, _, _, promptService, _, _, _, _ := a.GetServices()
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
	statsView.SetTitleColor(a.GetComponentColors("stats").Title.Color())

	// Format the statistics content
	var content strings.Builder

	// Summary section
	content.WriteString(a.FormatTitle("📊 USAGE SUMMARY") + "\n\n")
	content.WriteString(fmt.Sprintf("%sTotal Prompt Uses:%s %d\n", a.GetColorTag("link"), a.GetEndTag(), stats.TotalUsage))
	content.WriteString(fmt.Sprintf("%sActive Prompts:%s %d\n", a.GetColorTag("link"), a.GetEndTag(), stats.UniquePrompts))
	content.WriteString(fmt.Sprintf("%sFavorite Prompts:%s %d\n", a.GetColorTag("link"), a.GetEndTag(), len(stats.FavoritePrompts)))
	if !stats.LastUsed.IsZero() {
		content.WriteString(fmt.Sprintf("%sLast Used:%s %s\n", a.GetColorTag("link"), a.GetEndTag(), stats.LastUsed.Format("2006-01-02 15:04")))
	}
	content.WriteString("\n")

	// Top prompts section
	if len(stats.TopPrompts) > 0 {
		content.WriteString(a.FormatTitle("🏆 TOP PROMPTS") + "\n\n")
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

			content.WriteString(fmt.Sprintf("%s%d.%s %s %s%s\n", a.GetColorTag("header"), i+1, a.GetEndTag(), icon, prompt.Name, favoriteIcon))
			content.WriteString(fmt.Sprintf("    %sUses: %d | Category: %s | Last: %s%s\n",
				a.GetColorTag("secondary"), prompt.UsageCount, prompt.Category, prompt.LastUsed, a.GetEndTag()))
			content.WriteString("\n")
		}
	} else {
		content.WriteString(a.FormatTitle("🏆 TOP PROMPTS") + "\n\n")
		content.WriteString(a.FormatSecondary("No prompt usage recorded yet.") + "\n")
		content.WriteString(a.FormatSecondary("Start using prompts to see statistics here!") + "\n\n")
	}

	// Favorites section (if different from top)
	if len(stats.FavoritePrompts) > 0 && len(stats.FavoritePrompts) != len(stats.TopPrompts) {
		content.WriteString(a.FormatTitle("⭐ FAVORITE PROMPTS") + "\n\n")
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
			content.WriteString(fmt.Sprintf("  %sUses: %d | Category: %s%s\n",
				a.GetColorTag("secondary"), prompt.UsageCount, prompt.Category, a.GetEndTag()))
		}
		content.WriteString("\n")
	}

	// Help text
	content.WriteString(a.FormatSecondary("Press Esc to close | Use :stats to refresh"))

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

package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/render"
	"github.com/derailed/tview"
)

func starEmailData() render.EmailColumnData {
	cols := make([]render.ColumnCell, 8)
	cols[0] = render.ColumnCell{Content: "●", Alignment: tview.AlignCenter}     // flags
	cols[1] = render.ColumnCell{Content: "sender", Alignment: tview.AlignLeft}  // from
	cols[2] = render.ColumnCell{Content: "subject", Alignment: tview.AlignLeft} // subject
	cols[3] = render.ColumnCell{Content: "", Alignment: tview.AlignLeft}        // labels
	cols[4] = render.ColumnCell{Content: "📎", Alignment: tview.AlignCenter}     // attachment
	cols[5] = render.ColumnCell{Content: "  ", Alignment: tview.AlignCenter}    // calendar (none)
	cols[6] = render.ColumnCell{Content: "2h", Alignment: tview.AlignRight}     // date
	cols[7] = render.ColumnCell{Content: "⭐", Alignment: tview.AlignCenter}     // star (SRC 7)
	return render.EmailColumnData{RowType: render.RowTypeFlatMessage, Columns: cols}
}

// Wide-like config: flags, From, Subject, Labels, attachment, calendar, Date, star(last).
func TestMapStarColumn_Wide(t *testing.T) {
	a := &App{}
	config := []render.ColumnConfig{
		{Header: "", Alignment: tview.AlignCenter, MaxWidth: 3, MinWidth: 3},
		{Header: "From", Alignment: tview.AlignLeft},
		{Header: "Subject", Alignment: tview.AlignLeft},
		{Header: "Labels", Alignment: tview.AlignLeft},
		{Header: "", Alignment: tview.AlignCenter, MaxWidth: 2, MinWidth: 2},
		{Header: "", Alignment: tview.AlignCenter, MaxWidth: 2, MinWidth: 2},
		{Header: "Date", Alignment: tview.AlignRight},
		{Header: "", Alignment: tview.AlignCenter, MaxWidth: 2, MinWidth: 2}, // star (last)
	}
	m := a.mapEmailDataToResponsiveColumns(starEmailData(), config, 0)
	if m[0].Content != "●" {
		t.Errorf("flags = %q", m[0].Content)
	}
	if m[4].Content != "📎" {
		t.Errorf("attachment = %q", m[4].Content)
	}
	if m[7].Content != "⭐" {
		t.Errorf("last (star) = %q, want ⭐", m[7].Content)
	}
}

// Narrow layout that DROPS the attachment/calendar columns: flags, From, Subject, star(last).
// This is the case that a naive positional mapping got wrong.
func TestMapStarColumn_NarrowNoIcons(t *testing.T) {
	a := &App{}
	config := []render.ColumnConfig{
		{Header: "", Alignment: tview.AlignCenter, MaxWidth: 3, MinWidth: 3},
		{Header: "From", Alignment: tview.AlignLeft},
		{Header: "Subject", Alignment: tview.AlignLeft},
		{Header: "", Alignment: tview.AlignCenter, MaxWidth: 2, MinWidth: 2}, // star (last)
	}
	m := a.mapEmailDataToResponsiveColumns(starEmailData(), config, 0)
	if m[0].Content != "●" {
		t.Errorf("flags = %q", m[0].Content)
	}
	if m[3].Content != "⭐" {
		t.Errorf("last (star) = %q, want ⭐ even without attachment/calendar columns", m[3].Content)
	}
}

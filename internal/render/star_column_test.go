package render

import "testing"

func TestEmailRenderer_ExtractStarIcon(t *testing.T) {
	er := NewEmailRenderer(nil)
	if got := er.ExtractStarIcon(rmsg([]string{"STARRED", "INBOX"}, nil, 0)); got != "⭐" {
		t.Errorf("starred ExtractStarIcon = %q, want ⭐", got)
	}
	if got := er.ExtractStarIcon(rmsg([]string{"INBOX"}, nil, 0)); got != "  " {
		t.Errorf("unstarred ExtractStarIcon = %q, want two spaces", got)
	}
	if got := er.ExtractStarIcon(nil); got != "  " {
		t.Errorf("nil ExtractStarIcon = %q, want two spaces", got)
	}
}

func TestEmailRenderer_StarInDedicatedColumn(t *testing.T) {
	er := NewEmailRenderer(nil)
	// Unread + starred, not important: flags is just "●", the star is a separate
	// column (SRC index 7), NOT baked into the flags cell.
	data := er.FormatFlatMessageColumns(
		rmsg([]string{"STARRED", "UNREAD", "INBOX"}, map[string]string{"From": "a@b", "Subject": "hi"}, 0),
	)
	if len(data.Columns) < 8 {
		t.Fatalf("expected >=8 columns, got %d", len(data.Columns))
	}
	if data.Columns[7].Content != "⭐" {
		t.Errorf("star column (idx 7) = %q, want ⭐", data.Columns[7].Content)
	}
	if data.Columns[0].Content != "●" {
		t.Errorf("flags column (idx 0) = %q, want plain ● (no star)", data.Columns[0].Content)
	}

	// Unstarred → blank star column.
	data2 := er.FormatFlatMessageColumns(
		rmsg([]string{"INBOX"}, map[string]string{"From": "a@b", "Subject": "hi"}, 0),
	)
	if data2.Columns[7].Content != "  " {
		t.Errorf("unstarred star column = %q, want two spaces", data2.Columns[7].Content)
	}
}

package tui

import (
	"sync"
	"testing"

	"github.com/ajramos/giztui/internal/gmail"
	"github.com/derailed/tview"
)

// Two overlapping message loads (rapid selection changes, e.g. a vim range like s5s) each
// call renderMessageContent on a background goroutine, and the header SetText ran unguarded:
// concurrent writes corrupted the tview buffer and panicked ("index out of range" in
// TextView.Write). The header write must take readerMu like every other reader write
// (writeReaderContent/writeReaderPlaceholder). Run with -race: the unguarded version fails.
func TestUpdateReaderHeaderConcurrentWrites(t *testing.T) {
	a := &App{views: map[string]tview.Primitive{"header": tview.NewTextView()}}
	msg := &gmail.Message{}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.updateReaderHeader(msg)
		}()
	}
	wg.Wait()
}

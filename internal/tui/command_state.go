package tui

import "sync/atomic"

// commandState holds the `:`-command bar state, extracted from the App god object so the command
// history logic is cohesive and unit-testable. Only `mode` crosses a goroutine boundary (the
// focus-backup loop in executeContentSearch reads it), so it is an atomic.Bool; the rest are
// event-loop-only plain fields. atomic.Bool is non-copyable: never copy a commandState; use it as
// a field accessed via a.cmd.* with pointer-receiver methods.
type commandState struct {
	mode          atomic.Bool // command bar open?
	buffer        string      // current command text
	suggestion    string      // current Tab/auto suggestion
	focusOverride string      // overrides focus restoration after a special command
	history       []string    // executed-command history (capped)
	historyIndex  int         // cursor into history; == len(history) means "new line"
}

// addToHistory records a command, skipping empties and a consecutive duplicate, capping the history
// at 100 (oldest dropped), and resetting the cursor to the end.
func (c *commandState) addToHistory(cmd string) {
	if cmd == "" || (len(c.history) > 0 && c.history[len(c.history)-1] == cmd) {
		return
	}
	c.history = append(c.history, cmd)
	if len(c.history) > 100 {
		c.history = c.history[1:]
	}
	c.historyIndex = len(c.history)
}

// resetHistoryCursor parks the cursor at the end (the empty "new line"); called when the bar opens.
func (c *commandState) resetHistoryCursor() {
	c.historyIndex = len(c.history)
}

// historyUp moves to an older entry and returns its text. ok=false when already at the top (the
// caller then leaves the input unchanged).
func (c *commandState) historyUp() (string, bool) {
	if c.historyIndex > 0 {
		c.historyIndex--
		if c.historyIndex >= 0 && c.historyIndex < len(c.history) {
			return c.history[c.historyIndex], true
		}
	}
	return "", false
}

// historyDown moves toward newer entries. Past the end it parks the cursor at len(history) and
// returns ("", true) so the caller clears the input. ok is true whenever the caller should set the
// input to the returned text.
func (c *commandState) historyDown() (string, bool) {
	if c.historyIndex < len(c.history)-1 {
		c.historyIndex++
		if c.historyIndex >= 0 && c.historyIndex < len(c.history) {
			return c.history[c.historyIndex], true
		}
		return "", false
	}
	c.historyIndex = len(c.history)
	return "", true
}

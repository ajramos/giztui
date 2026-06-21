package tui

// overlayBackup snapshots the reader pane's content (text/header/title) before a full-pane overlay
// (help, preload status, prompt stats) replaces it, so the prior content can be restored on close.
// Event-loop only — no synchronization needed. Fields are written directly (the save sites guard each
// widget independently); active()/clear() centralize the restore guard and the reset.
type overlayBackup struct {
	text   string
	header string
	title  string
}

// active reports whether a backup is currently held — keyed off text, matching the original
// `<x>BackupText != ""` restore guard.
func (b *overlayBackup) active() bool { return b.text != "" }

// clear resets the backup to empty (called after restoring).
func (b *overlayBackup) clear() {
	b.text = ""
	b.header = ""
	b.title = ""
}

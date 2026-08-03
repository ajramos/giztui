package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// TestCommandsDocumentedInReference enforces command↔doc parity: every command in
// commandRegistry must appear in docs/KEYBOARD_SHORTCUTS.md — by its name or one of
// its aliases, written as ":token" or `token`. This is the guardrail for the
// recurring "shipped a command but forgot the reference doc" miss (e.g. star in
// v1.24.0). Matching is lenient (any alias counts) to avoid false failures; when it
// does fail, document the new command in the reference before shipping.
func TestCommandsDocumentedInReference(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	docPath := filepath.Join(root, "docs", "KEYBOARD_SHORTCUTS.md")
	raw, err := os.ReadFile(docPath) // #nosec G304 -- fixed repo doc resolved from runtime.Caller
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}
	doc := string(raw)

	documented := func(tok string) bool {
		if tok == "" {
			return false
		}
		if strings.Contains(doc, "`"+tok+"`") {
			return true
		}
		// ":tok" not immediately followed by another identifier char (so ":s" does not
		// match ":stats"). The reference also documents subcommands like ":prompt list".
		re := regexp.MustCompile(`:` + regexp.QuoteMeta(tok) + `([^a-zA-Z0-9_-]|$)`)
		return re.MatchString(doc)
	}

	var missing []string
	for _, c := range commandRegistry {
		tokens := append([]string{c.name}, c.aliases...)
		found := false
		for _, tok := range tokens {
			if documented(tok) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, ":"+c.name)
		}
	}

	if len(missing) > 0 {
		t.Fatalf("commands missing from docs/KEYBOARD_SHORTCUTS.md: %s\n"+
			"Document each new command (name or an alias) in the reference — see docs/RELEASE_PROCEDURE.md → Documentation Requirements.",
			strings.Join(missing, ", "))
	}
}

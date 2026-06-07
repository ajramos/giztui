// Command fetch is a THROWAWAY helper that reuses giztui's authenticated Gmail
// client to pull real HTML-heavy emails into poc/htmlmd/samples/ as .html files,
// so the converter bake-off runs against real messy newsletters.
//
// Usage:
//
//	go run ./poc/htmlmd/fetch                                  # default: category:promotions
//	go run ./poc/htmlmd/fetch -n 12 -query "category:updates"
//	go run ./poc/htmlmd/fetch -query "from:substack.com"
//
// Deleted with the rest of poc/ before the real feature.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ajramos/giztui/internal/gmail"
	"github.com/ajramos/giztui/pkg/auth"
)

const sampleDir = "poc/htmlmd/samples"

func main() {
	query := flag.String("query", "category:promotions newer_than:90d", "Gmail search query")
	n := flag.Int("n", 8, "max messages to fetch")
	flag.Parse()

	home, err := os.UserHomeDir()
	if err != nil {
		fatal("home dir: %v", err)
	}
	creds := filepath.Join(home, ".config/giztui/credentials.json")
	token := filepath.Join(home, ".config/giztui/token.json")

	ctx := context.Background()
	svc, err := auth.NewGmailService(ctx, creds, token,
		"https://www.googleapis.com/auth/gmail.readonly")
	if err != nil {
		fatal("gmail auth (creds=%s token=%s): %v", creds, token, err)
	}
	client := gmail.NewClient(svc)

	msgs, _, err := client.SearchMessagesPage(*query, int64(*n), "")
	if err != nil {
		fatal("search %q: %v", *query, err)
	}
	fmt.Printf("query %q → %d message(s)\n", *query, len(msgs))

	if err := os.MkdirAll(sampleDir, 0o755); err != nil {
		fatal("mkdir: %v", err)
	}

	saved := 0
	for _, m := range msgs {
		full, err := client.GetMessageWithContent(m.Id)
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", m.Id, err)
			continue
		}
		if strings.TrimSpace(full.HTML) == "" {
			fmt.Printf("  - %s (%s): no HTML part, skipped\n", shortID(m.Id), trunc(full.Subject, 40))
			continue
		}
		name := sanitize(full.Subject)
		if name == "" {
			name = shortID(m.Id)
		}
		path := filepath.Join(sampleDir, name+".html")
		if err := os.WriteFile(path, []byte(full.HTML), 0o644); err != nil {
			fmt.Printf("  ✗ write %s: %v\n", path, err)
			continue
		}
		fmt.Printf("  ✓ %s  (%d KB HTML)\n", filepath.Base(path), len(full.HTML)/1024)
		saved++
	}
	fmt.Printf("\nSaved %d HTML sample(s) to %s/\nNow run:  go run ./poc/htmlmd\n", saved, sampleDir)
}

var nonName = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func sanitize(s string) string {
	s = nonName.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	return trunc(s, 50)
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func shortID(id string) string { return trunc(id, 12) }

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

// Command htmlmd is a THROWAWAY proof-of-concept that compares HTML→Markdown
// converters on real email samples, to decide how giztui should render messy
// HTML newsletters. See docs/superpowers/specs/2026-06-07-html-to-markdown-rendering-design.md
//
// Delete this whole poc/ directory (and `go mod tidy`) before the real feature.
//
// Usage:
//
//	go run ./poc/htmlmd            # process every poc/htmlmd/samples/*.eml
//
// For each sample it writes poc/htmlmd/out/<sample>/ with:
//   - input.html                : the extracted text/html part (sanity check)
//   - baseline.md               : current giztui render.FormatEmailForTerminal
//   - markitdown.md             : Microsoft markitdown CLI (via local venv)
//   - htmltomd.md               : JohannesKaufmann/html-to-markdown v2 (pure Go)
//   - *.ansi.txt                : glamour-rendered terminal preview of each .md
package main

import (
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"encoding/base64"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"github.com/ajramos/giztui/internal/gmail"
	"github.com/ajramos/giztui/internal/render"
	"github.com/charmbracelet/glamour"
)

// goConv is html-to-markdown v2 WITH the table plugin enabled — the default
// top-level ConvertString omits it, which mangles the layout/data tables that
// dominate newsletters. Give the pure-Go contender a fair fight.
var goConv = converter.NewConverter(
	converter.WithPlugins(
		base.NewBasePlugin(),
		commonmark.NewCommonmarkPlugin(),
		table.NewTablePlugin(),
	),
)

const (
	sampleDir = "poc/htmlmd/samples"
	outDir    = "poc/htmlmd/out"
	venvMD    = "poc/htmlmd/.venv/bin/markitdown"
	wrapWidth = 100
)

func main() {
	eml, _ := filepath.Glob(filepath.Join(sampleDir, "*.eml"))
	htmls, _ := filepath.Glob(filepath.Join(sampleDir, "*.html"))
	samples := append(eml, htmls...)
	if len(samples) == 0 {
		fail("no samples found in %s — add .eml or .html files there\n"+
			"(run: go run ./poc/htmlmd/fetch   to pull real emails via giztui's Gmail client)", sampleDir)
	}
	fmt.Printf("Found %d sample(s)\n\n", len(samples))

	for _, path := range samples {
		name := strings.TrimSuffix(filepath.Base(path), ".eml")
		fmt.Printf("── %s ───────────────────────────────\n", name)
		if err := processSample(path, name); err != nil {
			fmt.Printf("  ✗ %v\n", err)
			continue
		}
	}
	fmt.Printf("\nDone. Compare outputs under %s/<sample>/\n", outDir)
}

func processSample(path, name string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	// .html files are already the raw HTML body; .eml needs MIME extraction.
	var htmlStr string
	if strings.EqualFold(filepath.Ext(path), ".html") {
		htmlStr = string(raw)
	} else {
		htmlStr, err = extractHTML(raw)
		if err != nil {
			return fmt.Errorf("extract html: %w", err)
		}
	}
	if strings.TrimSpace(htmlStr) == "" {
		return fmt.Errorf("no text/html part found")
	}

	dir := filepath.Join(outDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	write(dir, "input.html", htmlStr)

	// 1. Baseline — the current giztui renderer (reused as-is).
	baseline, _ := render.FormatEmailForTerminal(
		context.Background(),
		&gmail.Message{HTML: htmlStr},
		render.FormatOptions{WrapWidth: wrapWidth, UseLLM: false},
		nil,
	)
	write(dir, "baseline.md", baseline)
	renderGlamour(dir, "baseline", baseline)

	// 2. markitdown — Python CLI via local venv.
	if mdOut, err := runMarkitdown(htmlStr); err != nil {
		write(dir, "markitdown.md", "ERROR: "+err.Error())
		fmt.Printf("  ✗ markitdown: %v\n", err)
	} else {
		write(dir, "markitdown.md", mdOut)
		renderGlamour(dir, "markitdown", mdOut)
	}

	// 3. html-to-markdown v2 (+ table plugin) — pure Go.
	if goOut, err := goConv.ConvertString(htmlStr); err != nil {
		write(dir, "htmltomd.md", "ERROR: "+err.Error())
		fmt.Printf("  ✗ html-to-markdown: %v\n", err)
	} else {
		write(dir, "htmltomd.md", goOut)
		renderGlamour(dir, "htmltomd", goOut)
	}

	fmt.Printf("  ✓ wrote %s/\n", dir)
	return nil
}

// runMarkitdown writes html to a temp file and shells out to the venv markitdown.
func runMarkitdown(htmlStr string) (string, error) {
	tmp, err := os.CreateTemp("", "poc-*.html")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(htmlStr); err != nil {
		return "", err
	}
	tmp.Close()

	cmd := exec.Command(venvMD, tmp.Name())
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("markitdown failed: %w", err)
	}
	return string(out), nil
}

// renderGlamour writes a glamour terminal preview next to the markdown.
func renderGlamour(dir, kind, markdown string) {
	out, err := glamour.Render(markdown, "dark")
	if err != nil {
		return
	}
	write(dir, kind+".ansi.txt", out)
}

// extractHTML walks a raw RFC822 message and returns the best text/html part,
// decoding Content-Transfer-Encoding (quoted-printable / base64).
func extractHTML(raw []byte) (string, error) {
	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		return "", err
	}
	var best string
	if err := walkPart(msg.Header.Get("Content-Type"),
		msg.Header.Get("Content-Transfer-Encoding"), msg.Body, &best); err != nil {
		return "", err
	}
	return best, nil
}

func walkPart(contentType, cte string, body io.Reader, best *string) error {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		// No/!parseable Content-Type — treat as plain; ignore for HTML extraction.
		mediaType = "text/plain"
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return nil
		}
		mr := multipart.NewReader(body, boundary)
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			_ = walkPart(p.Header.Get("Content-Type"),
				p.Header.Get("Content-Transfer-Encoding"), p, best)
			p.Close()
		}
		return nil
	}

	if mediaType == "text/html" {
		decoded, err := decodeBody(cte, body)
		if err != nil {
			return err
		}
		// Prefer the largest html part (newsletters sometimes have stub parts).
		if len(decoded) > len(*best) {
			*best = decoded
		}
	}
	return nil
}

func decodeBody(cte string, body io.Reader) (string, error) {
	switch strings.ToLower(strings.TrimSpace(cte)) {
	case "base64":
		dec := base64.NewDecoder(base64.StdEncoding, body)
		b, err := io.ReadAll(dec)
		return string(b), err
	case "quoted-printable":
		b, err := io.ReadAll(quotedprintable.NewReader(body))
		return string(b), err
	default:
		b, err := io.ReadAll(body)
		return string(b), err
	}
}

func write(dir, file, content string) {
	if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
		fmt.Printf("  ✗ write %s: %v\n", file, err)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

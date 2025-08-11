package render

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	gmailwrap "github.com/ajramos/gmail-tui/internal/gmail"
	"golang.org/x/net/html"
	gmailapi "google.golang.org/api/gmail/v1"
)

// LinkRef represents a collected hyperlink reference
type LinkRef struct {
	Index int
	URL   string
	Text  string
}

// AttachmentMeta contains attachment or inline image metadata
type AttachmentMeta struct {
	Filename  string
	MimeType  string
	Size      int64
	Inline    bool
	ContentID string
}

// FormatOptions controls terminal formatting behavior
type FormatOptions struct {
	WrapWidth int
	UseLLM    bool
}

// TouchUpFunc is an optional post-processing function (e.g., LLM) to adjust whitespace only
type TouchUpFunc func(ctx context.Context, input string, wrapWidth int) (string, error)

// FormatEmailForTerminal builds terminal-friendly text with [BODY], [ATTACHMENTS], [IMAGES], [LINKS]
// It preserves quotes (>), code/pre/PGP blocks, lists and rudimentary ASCII tables.
func FormatEmailForTerminal(ctx context.Context, msg *gmailwrap.Message, opts FormatOptions, touchUp TouchUpFunc) (string, error) {
	// Choose source body
	var body string
	var links []LinkRef
	var imagesFromHTML []AttachmentMeta
	if strings.TrimSpace(msg.HTML) != "" {
		b, l, imgs, err := renderHTMLToText(msg.HTML)
		if err == nil {
			body, links, imagesFromHTML = b, l, imgs
		}
	}
	if strings.TrimSpace(body) == "" {
		body = msg.PlainText
	}

	// Collect attachments and inline images from MIME
	atts, inlineImgs := CollectAttachments(msg.Message)
	// Merge images detected from HTML refs (unique by ContentID/URL)
	images := mergeImages(imagesFromHTML, inlineImgs)

	// Normalize newlines
	body = normalizeNewlines(body)

	// If no links were collected from HTML, detect plain-text URLs to reference them
	if len(links) == 0 {
		detected, replaced := detectPlainTextLinks(body)
		if len(detected) > 0 {
			links = detected
			body = replaced
		}
	}

	// Wrap body respecting quotes, code and PGP blocks
	if opts.WrapWidth > 0 {
		body = WrapTextPreserving(body, opts.WrapWidth)
	}

	// Light sanitization for terminal glyphs (after wrapping) and de-duplication
	body = sanitizeBodyPreservingCode(body)
	body = dedupeConsecutiveLines(body)
	body = dedupeNearDuplicateParagraphs(body, 8)
	body = collapsePipeNavRuns(body)

	// Compose sections
	out := &strings.Builder{}
	out.WriteString("[BODY]\n")
	out.WriteString(body)
	out.WriteString("\n\n")

	// ATTACHMENTS
	out.WriteString("[ATTACHMENTS]\n")
	if len(atts) == 0 {
		out.WriteString("None\n\n")
	} else {
		for _, a := range atts {
			line := a.Filename
			if line == "" {
				line = "(attachment)"
			}
			if a.MimeType != "" {
				line += fmt.Sprintf(" (%s)", a.MimeType)
			}
			out.WriteString(line + "\n")
		}
		out.WriteString("\n")
	}

	// IMAGES
	out.WriteString("[IMAGES]\n")
	if len(images) == 0 {
		out.WriteString("None\n\n")
	} else {
		for _, im := range images {
			name := im.Filename
			if name == "" {
				if im.ContentID != "" {
					name = "cid:" + im.ContentID
				} else {
					name = "(image)"
				}
			}
			if im.MimeType != "" {
				name += fmt.Sprintf(" (%s)", im.MimeType)
			}
			out.WriteString(name + "\n")
		}
		out.WriteString("\n")
	}

	// LINKS
	out.WriteString("[LINKS]\n")
	if len(links) == 0 {
		out.WriteString("None\n")
	} else {
		// Ensure links sorted by Index
		sort.Slice(links, func(i, j int) bool { return links[i].Index < links[j].Index })
		for _, lr := range links {
			out.WriteString(fmt.Sprintf("(%d) %s\n", lr.Index, lr.URL))
		}
	}

	result := out.String()

	// Optional LLM touch-up (whitespace/line breaks only)
	if opts.UseLLM && touchUp != nil {
		if txt, err := touchUp(ctx, result, opts.WrapWidth); err == nil && strings.TrimSpace(txt) != "" {
			return txt, nil
		}
	}

	return result, nil
}

// detectPlainTextLinks finds URLs in plain text and replaces them with [n] references
func detectPlainTextLinks(input string) ([]LinkRef, string) {
	// Rough URL regex (http/https schemas)
	re := regexp.MustCompile(`(?i)\bhttps?://[\w\-\._~:/%\?#\[\]@!$&'()*+,;=]+`)
	idx := 0
	links := make([]LinkRef, 0, 4)
	replaced := re.ReplaceAllStringFunc(input, func(m string) string {
		idx++
		links = append(links, LinkRef{Index: idx, URL: m, Text: m})
		// Replace full URL in body with compact reference only
		return fmt.Sprintf("[%d]", idx)
	})
	return links, replaced
}

// sanitizeBodyPreservingCode applies glyph/whitespace sanitization to non-code lines
func sanitizeBodyPreservingCode(s string) string {
	lines := strings.Split(s, "\n")
	inCode := false
	for i, ln := range lines {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		lines[i] = sanitizeForTerminal(ln)
	}
	return strings.Join(lines, "\n")
}

// sanitizeForTerminal replaces common rich-text glyphs with ASCII-safe equivalents
func sanitizeForTerminal(s string) string {
	if s == "" {
		return s
	}
	// Normalize common unicode glyphs that often render as tofu
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\u00A0': // NBSP
			b.WriteRune(' ')
		case '\u200B', '\u200C', '\u200D', '\uFEFF':
			// zero-width and BOM → drop
		case '\u034F': // combining grapheme joiner
			// drop
		case '\u2060': // word joiner
			// drop
		case '\u00AD': // soft hyphen
			// drop
		case '\u2000', '\u2001', '\u2002', '\u2003', '\u2004', '\u2005', '\u2006', '\u2007', '\u2008', '\u2009', '\u200A':
			b.WriteRune(' ')
		case '\u202F': // narrow no-break space
			b.WriteRune(' ')
		case '\u2013', '\u2014':
			b.WriteRune('-')
		case '\u2022', '\u2043', '\u25AA', '\u25CF', '\u25E6':
			b.WriteString("- ")
		case '\u2018', '\u2019':
			b.WriteRune('\'')
		case '\u201C', '\u201D':
			b.WriteRune('"')
		case '\u2026':
			b.WriteString("...")
		default:
			// Skip control chars except newline/tab
			if unicode.IsControl(r) && r != '\n' && r != '\t' {
				continue
			}
			// Drop many symbol/emoji classified as So to avoid tofu blocks
			if unicode.Is(unicode.So, r) {
				continue
			}
			b.WriteRune(r)
		}
	}
	out := b.String()
	// collapse triple blank lines
	for strings.Contains(out, "\n\n\n") {
		out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	}
	return out
}

// dedupeConsecutiveLines drops consecutive duplicate lines and noisy separators
func dedupeConsecutiveLines(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return s
	}
	out := make([]string, 0, len(lines))
	var prev string
	for _, ln := range lines {
		cur := strings.TrimRight(ln, " ")
		trimmed := strings.TrimSpace(cur)
		// Skip duplicate consecutive lines (avoid repeated footer/header)
		if trimmed != "" && trimmed == prev {
			continue
		}
		// Drop noisy tiny tables converted to pipes
		if trimmed == "| |" || trimmed == "|" {
			continue
		}
		out = append(out, cur)
		prev = trimmed
	}
	res := strings.Join(out, "\n")
	for strings.Contains(res, "\n\n\n") {
		res = strings.ReplaceAll(res, "\n\n\n", "\n\n")
	}
	return res
}

// dedupeConsecutiveParagraphs removes consecutive duplicate paragraphs (blocks separated by blank lines)
// dedupeNearDuplicateParagraphs removes duplicates within a sliding window of size k
func dedupeNearDuplicateParagraphs(s string, window int) string {
	blocks := strings.Split(s, "\n\n")
	if len(blocks) <= 1 {
		return s
	}
	out := make([]string, 0, len(blocks))
	recent := make([]string, 0, window)
	normText := func(b string) string {
		cur := strings.TrimSpace(b)
		cur = sanitizeForTerminal(cur)
		return strings.Join(strings.Fields(cur), " ")
	}
	for _, b := range blocks {
		n := normText(b)
		dup := false
		if n != "" {
			for i := len(recent) - 1; i >= 0 && i >= len(recent)-window; i-- {
				if recent[i] == n {
					dup = true
					break
				}
			}
		}
		if dup {
			continue
		}
		out = append(out, strings.TrimRight(b, "\n"))
		// push to recent
		recent = append(recent, n)
		if len(recent) > window {
			recent = recent[1:]
		}
	}
	res := strings.Join(out, "\n\n")
	for strings.Contains(res, "\n\n\n\n") {
		res = strings.ReplaceAll(res, "\n\n\n\n", "\n\n")
	}
	return res
}

// collapsePipeNavRuns compacts lines that are mostly label pipes like "A | B | C" repeated
func collapsePipeNavRuns(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	lastNav := ""
	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if strings.Count(trimmed, "|") >= 2 {
			// keep only one occurrence if repeated
			norm := strings.Join(strings.Fields(strings.ReplaceAll(trimmed, "|", " ")), " ")
			if norm == lastNav {
				continue
			}
			lastNav = norm
		} else {
			lastNav = ""
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

// renderHTMLToText parses HTML and emits text, collecting links and inline image references.
func renderHTMLToText(htmlStr string) (string, []LinkRef, []AttachmentMeta, error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", nil, nil, err
	}
	var b strings.Builder
	links := make([]LinkRef, 0, 8)
	images := make([]AttachmentMeta, 0, 4)
	linkCounter := 0

	var quoteDepth int
	var inPre bool

	// visit walks the DOM. If skip is true, it will not emit text for this node or its children.
	var visit func(n *html.Node, skip bool)
	visit = func(n *html.Node, skip bool) {
		switch n.Type {
		case html.TextNode:
			if !skip {
				text := sanitizeText(n.Data)
				if strings.TrimSpace(text) != "" {
					if quoteDepth > 0 && !strings.HasPrefix(text, "> ") {
						// apply quote prefix once at start of each text run in blockquote
						prefix := strings.Repeat("> ", min(quoteDepth, 3))
						lines := strings.Split(text, "\n")
						for i, ln := range lines {
							if i > 0 {
								b.WriteByte('\n')
							}
							b.WriteString(prefix)
							b.WriteString(strings.TrimRightFunc(ln, unicode.IsSpace))
						}
					} else {
						b.WriteString(text)
					}
				}
			}
		case html.CommentNode:
			// ignore
		case html.ElementNode:
			tag := strings.ToLower(n.Data)
			switch tag {
			case "head", "style", "script", "title", "meta", "link":
				// Skip entire subtree
				return
			case "div", "section":
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					visit(c, skip)
				}
				b.WriteString("\n")
				return
			case "h1", "h2", "h3", "h4", "h5", "h6":
				var inner strings.Builder
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					collectText(&inner, c)
				}
				t := strings.TrimSpace(inner.String())
				if t != "" {
					b.WriteString(t)
					b.WriteString("\n\n")
				}
				return
			case "hr":
				b.WriteString("\n-----\n")
				return
			case "br":
				b.WriteByte('\n')
			case "p":
				// separate paragraphs with blank line
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					visit(c, skip)
				}
				b.WriteString("\n\n")
				return
			case "ul", "ol":
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.ElementNode && strings.ToLower(c.Data) == "li" {
						b.WriteString("- ")
						for li := c.FirstChild; li != nil; li = li.NextSibling {
							visit(li, skip)
						}
						b.WriteByte('\n')
					}
				}
				return
			case "blockquote":
				quoteDepth++
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					visit(c, skip)
				}
				quoteDepth--
				b.WriteByte('\n')
				return
			case "pre", "code":
				if !inPre {
					b.WriteString("```\n")
				}
				was := inPre
				inPre = true
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					visit(c, skip)
				}
				inPre = was
				if !inPre {
					b.WriteString("\n```\n")
				}
				return
			case "a":
				// Extract href
				href := ""
				for _, a := range n.Attr {
					if strings.EqualFold(a.Key, "href") {
						href = strings.TrimSpace(a.Val)
						break
					}
				}
				// Collect inner text
				var inner strings.Builder
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					collectText(&inner, c)
				}
				label := strings.TrimSpace(inner.String())
				if label == "" {
					// Fallback to aria-label/title/alt
					for _, a := range n.Attr {
						if strings.EqualFold(a.Key, "aria-label") || strings.EqualFold(a.Key, "title") || strings.EqualFold(a.Key, "alt") {
							if t := strings.TrimSpace(a.Val); t != "" {
								label = t
								break
							}
						}
					}
				}
				if label == "" {
					label = href
				}
				if href != "" {
					linkCounter++
					links = append(links, LinkRef{Index: linkCounter, URL: href, Text: label})
					b.WriteString(label + fmt.Sprintf(" [%d]", linkCounter))
				} else {
					b.WriteString(label)
				}
				return
			case "img":
				var src, cid string
				for _, a := range n.Attr {
					if strings.EqualFold(a.Key, "src") {
						src = strings.TrimSpace(a.Val)
					}
					if strings.EqualFold(a.Key, "cid") || strings.EqualFold(a.Key, "data-cid") {
						cid = strings.Trim(strings.TrimSpace(a.Val), "<>")
					}
				}
				images = append(images, AttachmentMeta{Filename: src, MimeType: "", Inline: true, ContentID: cid})
				return
			case "table":
				// Render rows anywhere inside the table (handles thead/tbody)
				var walkRows func(n *html.Node)
				walkRows = func(n *html.Node) {
					if n == nil {
						return
					}
					if n.Type == html.ElementNode && strings.ToLower(n.Data) == "tr" {
						row := make([]string, 0, 8)
						for td := n.FirstChild; td != nil; td = td.NextSibling {
							if td.Type == html.ElementNode {
								name := strings.ToLower(td.Data)
								if name == "td" || name == "th" {
									var cell strings.Builder
									for c := td.FirstChild; c != nil; c = c.NextSibling {
										collectText(&cell, c)
									}
									row = append(row, strings.TrimSpace(cell.String()))
								}
							}
						}
						if len(row) > 0 {
							// Heuristic: collapse “grid of single chars” (common in newsletter trackers)
							allSingle := true
							for _, c := range row {
								if len([]rune(strings.TrimSpace(c))) > 1 {
									allSingle = false
									break
								}
							}
							line := ""
							if allSingle && len(row) >= 5 {
								line = strings.Join(row, "")
							} else {
								line = strings.Join(row, " | ")
							}
							if strings.TrimSpace(line) != "" {
								b.WriteString(line + "\n")
							}
						}
					}
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						walkRows(c)
					}
				}
				walkRows(n)
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			// If current node is a style/script/head, we already returned above
			visit(c, skip)
		}
	}
	visit(doc, false)
	text := strings.TrimSpace(b.String())
	return text, links, images, nil
}

// CollectAttachments traverses MIME parts to collect attachments and inline images
func CollectAttachments(msg *gmailapi.Message) (atts []AttachmentMeta, images []AttachmentMeta) {
	if msg == nil || msg.Payload == nil {
		return nil, nil
	}
	atts = make([]AttachmentMeta, 0, 4)
	images = make([]AttachmentMeta, 0, 2)
	var walk func(p *gmailapi.MessagePart)
	walk = func(p *gmailapi.MessagePart) {
		if p == nil {
			return
		}
		// Inline images: image/* or Content-Id present
		isImage := strings.HasPrefix(strings.ToLower(p.MimeType), "image/")
		var cid string
		for _, h := range p.Headers {
			if strings.EqualFold(h.Name, "Content-Id") {
				cid = strings.Trim(h.Value, "<>")
				break
			}
		}
		if p.Body != nil && p.Body.AttachmentId != "" && p.Filename != "" {
			meta := AttachmentMeta{Filename: p.Filename, MimeType: p.MimeType, Size: p.Body.Size}
			if isImage || cid != "" {
				meta.Inline = true
				meta.ContentID = cid
				images = append(images, meta)
			} else {
				atts = append(atts, meta)
			}
		} else if isImage || cid != "" {
			images = append(images, AttachmentMeta{Filename: p.Filename, MimeType: p.MimeType, Inline: true, ContentID: cid})
		}
		for _, c := range p.Parts {
			walk(c)
		}
	}
	walk(msg.Payload)
	return atts, images
}

// WrapTextPreserving wraps text to width preserving quotes (> ), code/PGP blocks and URLs
func WrapTextPreserving(input string, width int) string {
	if width <= 0 {
		return input
	}
	lines := strings.Split(normalizeNewlines(input), "\n")
	var out strings.Builder
	inCode := false
	inPGP := false
	urlRe := regexp.MustCompile(`(?i)^[a-z][a-z0-9+\-.]*://\S+$`)
	for i, line := range lines {
		// Detect code fences
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCode = !inCode
			out.WriteString(line)
			if i < len(lines)-1 {
				out.WriteByte('\n')
			}
			continue
		}
		// Detect PGP/SMIME blocks
		if strings.HasPrefix(line, "-----BEGIN ") {
			inPGP = true
		}
		if inCode || inPGP {
			out.WriteString(line)
			if strings.HasPrefix(line, "-----END ") {
				inPGP = false
			}
			if i < len(lines)-1 {
				out.WriteByte('\n')
			}
			continue
		}
		// Determine quote prefix (e.g., "> ")
		prefix := ""
		trimmed := line
		for strings.HasPrefix(trimmed, "> ") {
			prefix += "> "
			trimmed = strings.TrimPrefix(trimmed, "> ")
		}
		// Tokenize without breaking URLs
		tokens := strings.Fields(trimmed)
		if len(tokens) == 0 {
			out.WriteString(prefix)
			if i < len(lines)-1 {
				out.WriteByte('\n')
			}
			continue
		}
		cur := prefix
		curLen := displayLen(cur)
		for ti, tok := range tokens {
			add := tok
			if urlRe.MatchString(tok) {
				// do not break
			} else if displayLen(tok)+curLen+1 > width {
				// soft-wrap token if extremely long, otherwise move to next line
				if displayLen(tok) > width-1 {
					// hard cut very long single token (rare)
					for len(add) > 0 {
						space := width - curLen
						if space <= 0 { // move line
							out.WriteString(strings.TrimRight(cur, " "))
							out.WriteByte('\n')
							cur = prefix
							curLen = displayLen(cur)
							space = width - curLen
						}
						if space > len(add) {
							cur += add
							curLen = displayLen(cur)
							add = ""
							break
						}
						cur += add[:space]
						add = add[space:]
						out.WriteString(strings.TrimRight(cur, " "))
						out.WriteByte('\n')
						cur = prefix
						curLen = displayLen(cur)
					}
					continue
				}
			}
			if curLen == displayLen(prefix) {
				// first token in line
				if curLen+displayLen(add) <= width {
					cur += add
					curLen = displayLen(cur)
				} else {
					// emit empty line with prefix then token on next
					out.WriteString(strings.TrimRight(cur, " "))
					out.WriteByte('\n')
					cur = prefix + add
					curLen = displayLen(cur)
				}
			} else if curLen+1+displayLen(add) <= width {
				cur += " " + add
				curLen = displayLen(cur)
			} else {
				out.WriteString(strings.TrimRight(cur, " "))
				out.WriteByte('\n')
				cur = prefix + add
				curLen = displayLen(cur)
			}
			if ti == len(tokens)-1 {
				out.WriteString(strings.TrimRight(cur, " "))
			}
		}
		if i < len(lines)-1 {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

// Helpers
func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	// collapse 3+ blank lines into 2
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}

func sanitizeText(s string) string {
	// Kept for compatibility with older calls; delegate to sanitizeForTerminal
	return sanitizeForTerminal(s)
}

func collectText(b *strings.Builder, n *html.Node) {
	switch n.Type {
	case html.TextNode:
		b.WriteString(sanitizeForTerminal(n.Data))
	case html.ElementNode:
		tag := strings.ToLower(n.Data)
		switch tag {
		case "br":
			b.WriteByte('\n')
		case "p":
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				collectText(b, c)
			}
			b.WriteString("\n\n")
			return
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectText(b, c)
	}
}

func mergeImages(a, b []AttachmentMeta) []AttachmentMeta {
	if len(a) == 0 {
		return append([]AttachmentMeta(nil), b...)
	}
	if len(b) == 0 {
		return append([]AttachmentMeta(nil), a...)
	}
	seen := make(map[string]struct{}, len(a)+len(b))
	res := make([]AttachmentMeta, 0, len(a)+len(b))
	key := func(im AttachmentMeta) string {
		if im.ContentID != "" {
			return "cid:" + im.ContentID
		}
		return im.Filename + "|" + im.MimeType
	}
	for _, im := range a {
		k := key(im)
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			res = append(res, im)
		}
	}
	for _, im := range b {
		k := key(im)
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			res = append(res, im)
		}
	}
	return res
}

func displayLen(s string) int { return len([]rune(s)) }
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

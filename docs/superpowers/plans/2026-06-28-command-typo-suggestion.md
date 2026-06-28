# Command Typo Suggestion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** On an unknown `:` command, enrich the error with the nearest registry match — `Unknown command: 'archvie'. Did you mean ':archive'?`.

**Architecture:** A free function `closestCommand(typed)` does case-insensitive Levenshtein over the existing `commandRegistry` names+aliases (guarded against noisy short matches); `executeCommand`'s `default:` case uses it to build the message. Pure in-memory, event-loop only.

**Tech Stack:** Go, standard `testing`.

---

## Task 1: `levenshtein` + `closestCommand` + tests

**Files:**
- Modify: `internal/tui/command_completion.go` (append the two functions)
- Test: `internal/tui/command_completion_test.go` (append)

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/command_completion_test.go`:

```go
func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"a", "a", 0},
		{"kitten", "sitting", 3},
		{"archive", "archvie", 2}, // i<->v transposition = 2 plain edits
		{"labels", "lables", 2},
	}
	for _, c := range cases {
		if got := levenshtein(c.a, c.b); got != c.want {
			t.Errorf("levenshtein(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestClosestCommand(t *testing.T) {
	cases := []struct {
		typed     string
		want      string
		wantFound bool
	}{
		{"archvie", "archive", true},
		{"lables", "labels", true},
		{"serach", "search", true},
		{"zzzzzzz", "", false},  // far from everything
		{"xy", "", false},       // too short (< 3)
		{"qqqq", "", false},     // no command within distance 2
	}
	for _, c := range cases {
		got, found := closestCommand(c.typed)
		if found != c.wantFound || got != c.want {
			t.Errorf("closestCommand(%q) = (%q,%v), want (%q,%v)", c.typed, got, found, c.want, c.wantFound)
		}
	}
}
```

- [ ] **Step 2: Run, verify FAIL**

Run: `go test ./internal/tui/ -run 'TestLevenshtein|TestClosestCommand' -v`
Expected: FAIL — `levenshtein`/`closestCommand` undefined.

- [ ] **Step 3: Implement both functions**

Append to `internal/tui/command_completion.go`:

```go
// levenshtein returns the edit distance between a and b (two-row DP, O(len(a)*len(b))).
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			m := del
			if ins < m {
				m = ins
			}
			if sub < m {
				m = sub
			}
			curr[j] = m
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

// closestCommand returns the canonical command name nearest to typed (case-insensitive Levenshtein
// over every registry name and alias of length >= 3), or ("", false) when there is no confident
// suggestion. Guards: typed must be >= 3 chars, and the best distance must be in (0, 2].
func closestCommand(typed string) (string, bool) {
	typed = strings.ToLower(strings.TrimSpace(typed))
	if len(typed) < 3 {
		return "", false
	}
	best := ""
	bestDist := 1 << 30
	consider := func(candidate, canonical string) {
		if len(candidate) < 3 {
			return
		}
		d := levenshtein(typed, strings.ToLower(candidate))
		if d < bestDist {
			bestDist = d
			best = canonical
		}
	}
	for i := range commandRegistry {
		s := &commandRegistry[i]
		consider(s.name, s.name)
		for _, al := range s.aliases {
			consider(al, s.name)
		}
	}
	if best != "" && bestDist > 0 && bestDist <= 2 {
		return best, true
	}
	return "", false
}
```

(`strings` is already imported in this file.)

- [ ] **Step 4: Run, verify PASS**

Run: `go test ./internal/tui/ -run 'TestLevenshtein|TestClosestCommand' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/tui/command_completion.go internal/tui/command_completion_test.go
git add internal/tui/command_completion.go internal/tui/command_completion_test.go
git commit -m "feat(tui): closestCommand + levenshtein for command typo suggestions"
```

(No Co-Authored-By line.)

---

## Task 2: Wire the suggestion into `executeCommand` + verify

**Files:**
- Modify: `internal/tui/commands.go` (the `default:` case of `executeCommand`, ~431-436)

- [ ] **Step 1: Update the default case**

In `internal/tui/commands.go`, replace:

```go
	default:
		// Check for numeric shortcuts like :1, :$
		if matched := a.executeNumericShortcut(command); !matched {
			a.showError(fmt.Sprintf("Unknown command: %s", command))
		}
	}
```

with:

```go
	default:
		// Check for numeric shortcuts like :1, :$
		if matched := a.executeNumericShortcut(command); !matched {
			if suggestion, ok := closestCommand(command); ok {
				a.showError(fmt.Sprintf("Unknown command: '%s'. Did you mean ':%s'?", command, suggestion))
			} else {
				a.showError(fmt.Sprintf("Unknown command: %s", command))
			}
		}
	}
```

- [ ] **Step 2: Build + format**

Run: `go build ./...` → success.
Run: `gofmt -w internal/tui/commands.go`.

- [ ] **Step 3: Tests**

Run: `go test ./internal/tui/ 2>&1 | tail -3` → `ok`.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/commands.go
git commit -m "feat(tui): suggest the nearest command on an unknown :command (#30)"
```

(No Co-Authored-By line.)

- [ ] **Step 5: Canonical gate + harness smoke**

Run: `make pre-commit-check` → "All pre-commit checks passed!".
Run: `make test 2>&1 | grep -E '^FAIL' || echo NO_FAILURES` → `NO_FAILURES`.
Run: `make build` → built.

Harness smoke (the pty driver `/tmp/giztui_smoke.py`; truncate the log first, type `:archvie` + Enter, then quit; the status message is logged):
```bash
log=~/.config/giztui/giztui.log; : > "$log"
timeout 25 python3 /tmp/giztui_smoke.py ./build/giztui "11:" "1::" "1:archvie" "1:\r" "2:" "1:q" >/dev/null 2>&1
grep -ai "Did you mean ':archive'" "$log" && echo "SUGGESTION OK"
grep -aic panic "$log"  # expect 0
```
Expected: the log contains `Did you mean ':archive'?`, 0 panics.

- [ ] **Step 6: Finish**

Use superpowers:finishing-a-development-branch. Update the in-app `:help`? No new keys/commands — the
behavior is a better error message, no help entry needed. Do NOT push/merge without explicit user
confirmation. No Co-Authored-By line.

---

## Self-Review

**Spec coverage:** `levenshtein` + `closestCommand` with the `<3` / `len>=3` / `0<d<=2` guards (Task 1);
wired into the `default:` case producing `Unknown command: '<typed>'. Did you mean ':<canonical>'?`
(Task 2 Step 1); unchanged message when no match (the `else`); tests + harness (Task 1 + Task 2 Step 5).
All spec requirements covered. ✓

**Placeholder scan:** none — full code in every step, exact commands. ✓

**Type consistency:** `levenshtein(a, b string) int` and `closestCommand(typed string) (string, bool)`
named/signed identically in Task 1 and called identically in Task 2. ✓

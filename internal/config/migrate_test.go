package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDeepMergeMissing(t *testing.T) {
	user := map[string]any{
		"_comment": "keep me",
		"llm":      map[string]any{"provider": "ollama"},
		"keep":     "user-value",
	}
	defaults := map[string]any{
		"llm":          map[string]any{"provider": "openai", "model": "x"},
		"keep":         "default-value",
		"auto_refresh": map[string]any{"enabled": false, "interval": "5m"},
	}

	added := deepMergeMissing(user, defaults, "")

	if user["keep"] != "user-value" {
		t.Fatalf("must not overwrite existing value, got %v", user["keep"])
	}
	if user["_comment"] != "keep me" {
		t.Fatal("must preserve _comment key")
	}
	llm := user["llm"].(map[string]any)
	if llm["provider"] != "ollama" {
		t.Fatalf("must keep user's nested value, got %v", llm["provider"])
	}
	if llm["model"] != "x" {
		t.Fatal("must add missing nested key")
	}
	if _, ok := user["auto_refresh"]; !ok {
		t.Fatal("must add wholly-missing top-level key")
	}
	got := map[string]bool{}
	for _, p := range added {
		got[p] = true
	}
	if !got["llm.model"] || !got["auto_refresh"] {
		t.Fatalf("added should include llm.model and auto_refresh, got %v", added)
	}
	if got["keep"] {
		t.Fatalf("added should NOT include existing key 'keep', got %v", added)
	}
}

func TestMigrateConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	original := `{
  "_comment": "my notes",
  "llm": { "provider": "ollama" }
}`
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	added, _, backup, err := MigrateConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(added) == 0 {
		t.Fatal("expected some added keys")
	}
	if backup == "" {
		t.Fatal("expected a backup path")
	}
	if b, _ := os.ReadFile(filepath.Clean(backup)); string(b) != original {
		t.Fatal("backup should contain the original file bytes")
	}
	merged := map[string]any{}
	b, _ := os.ReadFile(filepath.Clean(path))
	if err := json.Unmarshal(b, &merged); err != nil {
		t.Fatalf("merged file invalid: %v", err)
	}
	if merged["_comment"] != "my notes" {
		t.Fatal("merged lost the _comment")
	}
	if _, ok := merged["auto_refresh"]; !ok {
		t.Fatal("merged should contain auto_refresh defaults")
	}
}

func TestMigrateConfigFile_NoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	full, _ := json.MarshalIndent(DefaultConfig(), "", "  ")
	if err := os.WriteFile(path, full, 0600); err != nil {
		t.Fatal(err)
	}
	added, removed, backup, err := MigrateConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 0 || len(removed) != 0 || backup != "" {
		t.Fatalf("expected no-op, got added=%v removed=%v backup=%q", added, removed, backup)
	}
	if _, err := os.Stat(path + ".bak"); !os.IsNotExist(err) {
		t.Fatal("no .bak should be written on a no-op")
	}
}

func TestMigrateConfigFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{ not json "), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := MigrateConfigFile(path); err == nil {
		t.Fatal("expected an error on invalid JSON")
	}
	if _, err := os.Stat(path + ".bak"); !os.IsNotExist(err) {
		t.Fatal("must not write a backup when the source is invalid")
	}
}

func TestSetAutoRefreshEnabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// A file with an unrelated section + a _comment key, plus auto_refresh.enabled=true.
	orig := `{
  "_comment": "keep me",
  "slack": {"enabled": true},
  "auto_refresh": {"enabled": true, "interval": "1m", "notify_slack": true}
}`
	if err := os.WriteFile(path, []byte(orig), 0600); err != nil {
		t.Fatal(err)
	}
	if err := SetAutoRefreshEnabled(path, false); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	data, _ := os.ReadFile(path) // #nosec G304 -- path is a t.TempDir() join
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	ar, ok := got["auto_refresh"].(map[string]any)
	if !ok {
		t.Fatalf("auto_refresh section missing: %v", got)
	}
	if ar["enabled"] != false {
		t.Fatalf("enabled not persisted as false: %v", ar["enabled"])
	}
	// Sibling keys within the section and unrelated keys must be preserved.
	if ar["interval"] != "1m" || ar["notify_slack"] != true {
		t.Fatalf("auto_refresh siblings clobbered: %v", ar)
	}
	if got["_comment"] != "keep me" {
		t.Fatalf("_comment key dropped: %v", got["_comment"])
	}
	if s, _ := got["slack"].(map[string]any); s == nil || s["enabled"] != true {
		t.Fatalf("unrelated slack section clobbered: %v", got["slack"])
	}
	// Round-trip back to true.
	if err := SetAutoRefreshEnabled(path, true); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path) // #nosec G304 -- path is a t.TempDir() join
	_ = json.Unmarshal(data, &got)
	if ar, _ := got["auto_refresh"].(map[string]any); ar["enabled"] != true {
		t.Fatalf("enabled not re-enabled: %v", got["auto_refresh"])
	}
}

func TestPruneObsolete(t *testing.T) {
	user := map[string]any{
		"keys": map[string]any{
			"prompt_test": "ctrl+t",
			"speak":       "ctrl+e",
		},
		"llm": map[string]any{"provider": "ollama"},
	}
	removed := pruneObsolete(user, []string{"keys.prompt_test", "keys.does_not_exist", "top.missing"})

	if len(removed) != 1 || removed[0] != "keys.prompt_test" {
		t.Fatalf("expected only keys.prompt_test removed, got %v", removed)
	}
	keys := user["keys"].(map[string]any)
	if _, ok := keys["prompt_test"]; ok {
		t.Fatal("keys.prompt_test should have been deleted")
	}
	if keys["speak"] != "ctrl+e" {
		t.Fatal("must not touch sibling keys")
	}
}

func TestMigrateConfigFile_PrunesObsolete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// A config that is otherwise complete (no additions) but carries the obsolete key.
	full := map[string]any{}
	data, _ := json.Marshal(DefaultConfig())
	_ = json.Unmarshal(data, &full)
	full["keys"].(map[string]any)["prompt_test"] = "ctrl+t"
	out, _ := json.MarshalIndent(full, "", "  ")
	if err := os.WriteFile(path, out, 0600); err != nil {
		t.Fatal(err)
	}

	added, removed, backup, err := MigrateConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 0 {
		t.Fatalf("expected no additions, got %v", added)
	}
	if len(removed) != 1 || removed[0] != "keys.prompt_test" {
		t.Fatalf("expected keys.prompt_test pruned, got %v", removed)
	}
	if backup == "" {
		t.Fatal("expected a backup when a key is pruned")
	}
	merged := map[string]any{}
	b, _ := os.ReadFile(filepath.Clean(path))
	if err := json.Unmarshal(b, &merged); err != nil {
		t.Fatalf("merged file invalid: %v", err)
	}
	if _, ok := merged["keys"].(map[string]any)["prompt_test"]; ok {
		t.Fatal("prompt_test should be gone from the migrated file")
	}
}

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

	added, backup, err := MigrateConfigFile(path)
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
	added, backup, err := MigrateConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 0 || backup != "" {
		t.Fatalf("expected no-op, got added=%v backup=%q", added, backup)
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
	if _, _, err := MigrateConfigFile(path); err == nil {
		t.Fatal("expected an error on invalid JSON")
	}
	if _, err := os.Stat(path + ".bak"); !os.IsNotExist(err) {
		t.Fatal("must not write a backup when the source is invalid")
	}
}

package obsidian

import "testing"

func TestDefaultObsidianConfig(t *testing.T) {
	c := DefaultObsidianConfig()
	if c == nil {
		t.Fatal("DefaultObsidianConfig returned nil")
	}
	if c.IngestFolder == "" || c.TemplateFile == "" || c.FilenameFormat == "" {
		t.Errorf("defaults should be populated, got %+v", c)
	}
}

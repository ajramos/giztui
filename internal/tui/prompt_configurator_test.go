package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPromptConfiguratorTitle_SingleMode(t *testing.T) {
	got := promptConfiguratorTitle(promptConfiguratorContext{mode: "single"})
	assert.Contains(t, got, "Prompt Configurator")
	assert.Contains(t, got, "1 msg")
}

func TestPromptConfiguratorTitle_BulkMode(t *testing.T) {
	got := promptConfiguratorTitle(promptConfiguratorContext{
		mode:       "bulk",
		messageIDs: []string{"a", "b", "c"},
	})
	assert.Contains(t, got, "3 msgs scoped")
}

func TestPromptConfiguratorTitle_BulkWithCategory(t *testing.T) {
	got := promptConfiguratorTitle(promptConfiguratorContext{
		mode:         "bulk",
		messageIDs:   []string{"a", "b"},
		categoryName: "Newsletters",
	})
	assert.Contains(t, got, "2 msgs from")
	assert.Contains(t, got, "Newsletters")
}

func TestPromptConfiguratorTitle_DraftMode(t *testing.T) {
	got := promptConfiguratorTitle(promptConfiguratorContext{mode: "draft"})
	assert.Contains(t, got, "draft only")
}

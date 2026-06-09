package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/services"
	"github.com/derailed/tview"
	"github.com/stretchr/testify/assert"
)

func TestActionPlan_RebuildTreeAndRemoveCategory(t *testing.T) {
	plan := &services.ActionPlan{
		TotalAnalyzed: 3, BatchesTotal: 1, BatchesDone: 1,
		Categories: []services.ActionPlanCategory{
			{Name: "Newsletters", Priority: "low", Action: "archive", MessageIDs: []string{"m1", "m2"}},
			{Name: "Follow up", Priority: "high", Action: "label", Label: "needs-reply", MessageIDs: []string{"m3"}},
		},
	}
	a := &App{}
	a.Keys.Archive, a.Keys.ToggleRead, a.Keys.Trash, a.Keys.ManageLabels = "a", "t", "d", "l"
	root := tview.NewTreeNode("")
	tree := tview.NewTreeView().SetRoot(root).SetCurrentNode(root)
	state := &actionPlanState{plan: plan, selectedCategory: 0, root: root, tree: tree}
	state.header = tview.NewTextView()

	// rebuildActionPlanTree populates nodes and clamps selectedCategory.
	state.selectedCategory = 5 // out-of-range; should be clamped to 1
	a.rebuildActionPlanTree(state)
	assert.Equal(t, 1, state.selectedCategory)
	assert.Len(t, root.GetChildren(), 2)

	state.selectedCategory = 0
	a.rebuildActionPlanTree(state)
	assert.Equal(t, 0, state.selectedCategory)

	// Removing a completed category drops it and re-renders without panic.
	a.removeActionPlanCategory(state, "Newsletters")
	assert.Len(t, state.plan.Categories, 1)
	assert.Equal(t, "Follow up", state.plan.Categories[0].Name)
}

func TestActionPlan_VirtualBulkSelection(t *testing.T) {
	a := &App{selected: map[string]bool{"old": true}}
	a.setVirtualBulkSelection([]string{"m1", "m2"})
	assert.True(t, a.bulkMode)
	assert.True(t, a.selected["m1"])
	assert.True(t, a.selected["m2"])
	assert.False(t, a.selected["old"]) // previous selection cleared
}

func TestActionPlan_CurrentCategoryNilSafe(t *testing.T) {
	a := &App{}
	// Empty plan → currentActionPlanCategory returns nil, no panic.
	state := &actionPlanState{plan: &services.ActionPlan{}, selectedCategory: 0}
	assert.Nil(t, a.currentActionPlanCategory(state))
	// Nil plan → also nil.
	state2 := &actionPlanState{plan: nil, selectedCategory: 0}
	assert.Nil(t, a.currentActionPlanCategory(state2))
}

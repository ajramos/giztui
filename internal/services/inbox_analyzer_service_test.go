package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractJSONObject(t *testing.T) {
	// Plain JSON passes through.
	assert.Equal(t, `{"a":1}`, extractJSONObject(`{"a":1}`))
	// Markdown fences are stripped.
	assert.Equal(t, `{"a":1}`, extractJSONObject("```json\n{\"a\":1}\n```"))
	// Leading/trailing prose is removed.
	assert.Equal(t, `{"a":1}`, extractJSONObject("Here you go:\n{\"a\":1}\nDone."))
	// Nested braces are balanced correctly.
	assert.Equal(t, `{"a":{"b":2}}`, extractJSONObject(`prefix {"a":{"b":2}} suffix`))
	// No object → empty string.
	assert.Equal(t, "", extractJSONObject("no json here"))
}

func TestParseAnalyzerResponse(t *testing.T) {
	// batchIDs maps per-batch local number (1-based) → concrete message ID.
	batchIDs := []string{"m1", "m2", "m3", "m4"}
	raw := `{
	  "categories": [
	    {"name":"Newsletters","priority":"low","description":"marketing","action":"archive","label":"","messages":[1,3]},
	    {"name":"Follow up","priority":"high","description":"needs reply","action":"label","label":"needs-reply","messages":[4]}
	  ],
	  "read_manually": [2]
	}`

	cats, readManually, err := parseAnalyzerResponse(raw, batchIDs)
	assert.NoError(t, err)
	assert.Len(t, cats, 2)
	assert.Equal(t, []string{"m1", "m3"}, cats[0].MessageIDs)
	assert.Equal(t, "archive", cats[0].Action)
	assert.Equal(t, "needs-reply", cats[1].Label)
	assert.Equal(t, []string{"m4"}, cats[1].MessageIDs)
	assert.Equal(t, []int{2}, readManually) // local indices, resolved by caller
}

func TestParseAnalyzerResponse_OutOfRangeIndexIgnored(t *testing.T) {
	batchIDs := []string{"m1", "m2"}
	raw := `{"categories":[{"name":"X","priority":"low","description":"d","action":"archive","label":"","messages":[1,9]}],"read_manually":[]}`
	cats, _, err := parseAnalyzerResponse(raw, batchIDs)
	assert.NoError(t, err)
	// index 9 is out of range → dropped, index 1 → m1.
	assert.Equal(t, []string{"m1"}, cats[0].MessageIDs)
}

func TestParseAnalyzerResponse_Malformed(t *testing.T) {
	_, _, err := parseAnalyzerResponse("not json", []string{"m1"})
	assert.Error(t, err)
}

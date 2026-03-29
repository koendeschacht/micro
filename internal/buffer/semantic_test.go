package buffer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetSemanticHighlightsJSONStoresTransitions(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("alpha beta", "", BTDefault)
	ok := b.SetSemanticHighlightsJSON(`[
		{"line":0,"start":0,"length":5,"group":"function.declaration"},
		{"line":0,"start":6,"length":4,"group":"parameter.readonly"}
	]`, 3)

	assert.True(ok)
	assert.Equal(3, b.SemanticVersion)
	assert.Equal(SemanticLineMatch{0: "function.declaration", 5: "", 6: "parameter.readonly", 10: ""}, b.SemanticLine(0))
}

func TestSetSemanticHighlightsJSONRejectsOlderVersion(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("alpha", "", BTDefault)
	assert.True(b.SetSemanticHighlightsJSON(`[{"line":0,"start":0,"length":5,"group":"function"}]`, 2))
	assert.False(b.SetSemanticHighlightsJSON(`[{"line":0,"start":0,"length":5,"group":"variable"}]`, 1))
	assert.Equal(2, b.SemanticVersion)
	assert.Equal("function", b.SemanticLine(0)[0])
}

func TestMarkModifiedClearsSemanticHighlights(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("alpha", "", BTDefault)
	assert.True(b.SetSemanticHighlightsJSON(`[{"line":0,"start":0,"length":5,"group":"function"}]`, 1))

	b.MarkModified(0, 0)

	assert.Nil(b.SemanticMatch)
	assert.Equal(0, b.SemanticVersion)
}

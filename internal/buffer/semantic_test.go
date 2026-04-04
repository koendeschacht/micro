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

func TestMarkModifiedClearsSemanticHighlightsFromEditedLine(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("alpha\nbeta\ngamma", "", BTDefault)
	assert.True(b.SetSemanticHighlightsJSON(`[
		{"line":0,"start":0,"length":5,"group":"function"},
		{"line":1,"start":0,"length":4,"group":"variable"},
		{"line":2,"start":0,"length":5,"group":"string"}
	]`, 1))

	b.MarkModified(1, 1)

	assert.Equal(SemanticLineMatch{0: "function", 5: ""}, b.SemanticLine(0))
	assert.Nil(b.SemanticLine(1))
	assert.Equal(SemanticLineMatch{0: "string", 5: ""}, b.SemanticLine(2))
	assert.Equal(1, b.SemanticVersion)
}

func TestInsertPreservesSemanticHighlightsBelowEditedLine(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("alpha\nbeta\ngamma", "", BTDefault)
	assert.True(b.SetSemanticHighlightsJSON(`[
		{"line":0,"start":0,"length":5,"group":"function"},
		{"line":1,"start":0,"length":4,"group":"variable"},
		{"line":2,"start":0,"length":5,"group":"string"}
	]`, 1))

	b.insert(Loc{2, 1}, []byte("\nzz"))

	assert.Equal(SemanticLineMatch{0: "function", 5: ""}, b.SemanticLine(0))
	assert.Nil(b.SemanticLine(1))
	assert.Nil(b.SemanticLine(2))
	assert.Equal(SemanticLineMatch{0: "string", 5: ""}, b.SemanticLine(3))
}

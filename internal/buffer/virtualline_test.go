package buffer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetVirtualLinesJSONStoresEntries(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("alpha\nbeta\n", "", BTDefault)
	ok := b.SetVirtualLinesJSON("diff", `[
		{"line":1,"above":true,"text":"- removed before beta","group":"diff-deleted"},
		{"line":1,"above":false,"text":"+ inserted after beta","group":"diff-added"}
	]`, 3)

	assert.True(ok)
	assert.Equal(3, b.virtualLineOwnerState("diff").Version)
	assert.Equal([]VirtualLine{{ID: "diff:0", Text: "- removed before beta", Group: "diff-deleted"}}, b.VirtualLines(1, true))
	assert.Equal([]VirtualLine{{ID: "diff:1", Text: "+ inserted after beta", Group: "diff-added"}}, b.VirtualLines(1, false))
	assert.True(b.HasVirtualLines())
	assert.Equal(1, b.VirtualLineCount(1, true))
	assert.Equal(1, b.VirtualLineCount(1, false))
}

func TestSetVirtualLinesJSONRejectsOlderVersion(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("alpha", "", BTDefault)
	assert.True(b.SetVirtualLinesJSON("diff", `[
		{"line":0,"above":true,"text":"before","group":"diff-deleted"}
	]`, 2))
	assert.False(b.SetVirtualLinesJSON("diff", `[
		{"line":0,"above":true,"text":"newer? no","group":"diff-added"}
	]`, 1))

	assert.Equal([]VirtualLine{{ID: "diff:0", Text: "before", Group: "diff-deleted"}}, b.VirtualLines(0, true))
}

func TestClearVirtualLinesRemovesOnlyOwner(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("alpha", "", BTDefault)
	assert.True(b.SetVirtualLinesJSON("one", `[
		{"line":0,"above":true,"text":"one-a","group":"diff-deleted"}
	]`, 1))
	assert.True(b.SetVirtualLinesJSON("two", `[
		{"line":0,"above":true,"text":"two-a","group":"diff-added"},
		{"line":0,"above":false,"text":"two-b","group":"diff-added"}
	]`, 1))

	assert.Equal([]VirtualLine{
		{ID: "one:0", Text: "one-a", Group: "diff-deleted"},
		{ID: "two:0", Text: "two-a", Group: "diff-added"},
	}, b.VirtualLines(0, true))

	b.ClearVirtualLines("one")
	assert.Equal([]VirtualLine{{ID: "two:0", Text: "two-a", Group: "diff-added"}}, b.VirtualLines(0, true))
	assert.Equal([]VirtualLine{{ID: "two:1", Text: "two-b", Group: "diff-added"}}, b.VirtualLines(0, false))
}

func TestSetVirtualLineDecorationsJSONStoresResolvedDecorations(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("alpha\nbeta\n", "", BTDefault)
	assert.True(b.SetVirtualLinesJSON("diff", `[
		{"id":"old-1","line":1,"above":true,"text":"removed text","group":"diff-deleted"}
	]`, 1))
	assert.True(b.SetVirtualLineDecorationsJSON("diff", `[
		{"id":"old-1","start":2,"end":8,"group":"diff-removed-word","priority":3}
	]`, 2))

	assert.Equal(2, b.virtualLineDecorationOwnerState("diff").Version)
	assert.Equal(DecorationLineMatch{2: {Group: "diff-removed-word", Priority: 3}, 8: {Group: "", Priority: 0}}, b.VirtualLineDecoration("old-1"))
}

func TestVirtualLineDecorationsRejectOlderVersionAndUnknownIDs(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("alpha", "", BTDefault)
	assert.True(b.SetVirtualLinesJSON("diff", `[
		{"id":"old-1","line":0,"above":true,"text":"removed text","group":"diff-deleted"}
	]`, 1))
	assert.True(b.SetVirtualLineDecorationsJSON("diff", `[
		{"id":"old-1","start":0,"end":7,"group":"diff-removed-word","priority":1},
		{"id":"missing","start":0,"end":3,"group":"ignored","priority":2}
	]`, 2))
	assert.False(b.SetVirtualLineDecorationsJSON("diff", `[
		{"id":"old-1","start":0,"end":4,"group":"newer? no","priority":1}
	]`, 1))

	assert.Equal(DecorationLineMatch{0: {Group: "diff-removed-word", Priority: 1}, 7: {Group: "", Priority: 0}}, b.VirtualLineDecoration("old-1"))
	assert.Nil(b.VirtualLineDecoration("missing"))
}

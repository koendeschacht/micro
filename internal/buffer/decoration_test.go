package buffer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetDecorationsJSONStoresResolvedDecorations(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("alpha\nbeta\n", "", BTDefault)
	ok := b.SetDecorationsJSON("diff", `[
		{"kind":"span","startLine":0,"start":0,"endLine":0,"end":5,"group":"diff-added","priority":1},
		{"kind":"line","line":1,"group":"diff-modified","priority":2},
		{"kind":"gutter","line":1,"group":"diff-modified","symbol":"!","priority":3}
	]`, 3)

	assert.True(ok)
	assert.Equal(3, b.decorationOwnerState("diff").Version)
	assert.Equal(DecorationLineMatch{0: {Group: "diff-added", Priority: 1}, 5: {Group: "", Priority: 0}}, b.DecorationLine(0))
	assert.Equal(Decoration{Group: "diff-modified", Priority: 2}, func() Decoration {
		decoration, _ := b.LineDecoration(1)
		return decoration
	}())
	assert.Equal(GutterDecoration{Group: "diff-modified", Symbol: '!', Priority: 3}, func() GutterDecoration {
		decoration, _ := b.GutterDecoration(1)
		return decoration
	}())
	assert.True(b.HasGutterDecorations())
}

func TestSetDecorationsJSONRejectsOlderVersion(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("alpha", "", BTDefault)
	assert.True(b.SetDecorationsJSON("diff", `[
		{"kind":"line","line":0,"group":"diff-added","priority":1}
	]`, 2))
	assert.False(b.SetDecorationsJSON("diff", `[
		{"kind":"line","line":0,"group":"diff-modified","priority":1}
	]`, 1))

	decoration, ok := b.LineDecoration(0)
	assert.True(ok)
	assert.Equal(Decoration{Group: "diff-added", Priority: 1}, decoration)
}

func TestDecorationsMergeOwnersByPriority(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("alpha", "", BTDefault)
	assert.True(b.SetDecorationsJSON("one", `[
		{"kind":"span","startLine":0,"start":0,"endLine":0,"end":5,"group":"diff-added","priority":1},
		{"kind":"line","line":0,"group":"diff-added","priority":1},
		{"kind":"gutter","line":0,"group":"diff-added","symbol":"+","priority":1}
	]`, 1))
	assert.True(b.SetDecorationsJSON("two", `[
		{"kind":"span","startLine":0,"start":2,"endLine":0,"end":4,"group":"diff-modified","priority":2},
		{"kind":"line","line":0,"group":"diff-modified","priority":2},
		{"kind":"gutter","line":0,"group":"diff-modified","symbol":"~","priority":2}
	]`, 1))

	assert.Equal(DecorationLineMatch{
		0: {Group: "diff-added", Priority: 1},
		2: {Group: "diff-modified", Priority: 2},
		4: {Group: "diff-added", Priority: 1},
		5: {Group: "", Priority: 0},
	}, b.DecorationLine(0))

	lineDecoration, ok := b.LineDecoration(0)
	assert.True(ok)
	assert.Equal(Decoration{Group: "diff-modified", Priority: 2}, lineDecoration)

	gutterDecoration, ok := b.GutterDecoration(0)
	assert.True(ok)
	assert.Equal(GutterDecoration{Group: "diff-modified", Symbol: '~', Priority: 2}, gutterDecoration)

	b.ClearDecorations("two")
	assert.Equal(DecorationLineMatch{0: {Group: "diff-added", Priority: 1}, 5: {Group: "", Priority: 0}}, b.DecorationLine(0))
}

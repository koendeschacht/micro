package action

import (
	"testing"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/display"
	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

func TestVirtualLineIDAtVisualAvailableToLua(t *testing.T) {
	b := buffer.NewBufferFromString("zero\none", "", buffer.BTDefault)
	b.SetOptionNative("softwrap", false)
	assert.True(t, b.SetVirtualLinesJSON("test", `[
		{"id":"comment-1","line":1,"above":true,"text":"comment"}
	]`, 1))

	w := display.NewBufWindow(0, 0, 80, 24, b)
	h := NewBufPane(b, w, nil)
	l := lua.NewState()
	defer l.Close()
	l.SetGlobal("pane", luar.New(l, h))

	assert.NoError(t, l.DoString(`result = pane:VirtualLineIDAtVisual(0, 1)`))
	assert.Equal(t, lua.LString("comment-1"), l.GetGlobal("result"))
}

func TestMoveCursorUpSkipsVirtualLinesAbove(t *testing.T) {
	b := buffer.NewBufferFromString("zero\none\ntwo", "", buffer.BTDefault)
	b.SetOptionNative("softwrap", true)
	assert.True(t, b.SetVirtualLinesJSON("test", `[
		{"line":1,"above":true,"text":"- removed 1","group":"diff-deleted"},
		{"line":1,"above":true,"text":"- removed 2","group":"diff-deleted"}
	]`, 1))

	w := display.NewBufWindow(0, 0, 80, 24, b)
	w.Resize(80, 24)
	h := NewBufPane(b, w, nil)

	h.Cursor.GotoLoc(buffer.Loc{X: 0, Y: 1})
	h.MoveCursorUp(1)

	assert.Equal(t, buffer.Loc{X: 0, Y: 0}, h.Cursor.Loc)
}

func TestMoveCursorDownSkipsVirtualLinesBelow(t *testing.T) {
	b := buffer.NewBufferFromString("zero\none\ntwo", "", buffer.BTDefault)
	b.SetOptionNative("softwrap", true)
	assert.True(t, b.SetVirtualLinesJSON("test", `[
		{"line":1,"above":false,"text":"+ added 1","group":"diff-added"},
		{"line":1,"above":false,"text":"+ added 2","group":"diff-added"}
	]`, 1))

	w := display.NewBufWindow(0, 0, 80, 24, b)
	w.Resize(80, 24)
	h := NewBufPane(b, w, nil)

	h.Cursor.GotoLoc(buffer.Loc{X: 3, Y: 1})
	h.MoveCursorDown(1)

	assert.Equal(t, buffer.Loc{X: 3, Y: 2}, h.Cursor.Loc)
}

func TestMoveCursorDownWithSoftwrapSkipsVirtualRowsMappedToLineEnd(t *testing.T) {
	b := buffer.NewBufferFromString("zero\nlonger line\nthird", "", buffer.BTDefault)
	b.SetOptionNative("softwrap", true)
	assert.True(t, b.SetVirtualLinesJSON("test", `[
		{"line":1,"above":false,"text":"+ added 1","group":"diff-added"},
		{"line":1,"above":false,"text":"+ added 2","group":"diff-added"}
	]`, 1))

	w := display.NewBufWindow(0, 0, 80, 24, b)
	w.Resize(80, 24)
	h := NewBufPane(b, w, nil)

	h.Cursor.GotoLoc(buffer.Loc{X: 0, Y: 1})
	h.MoveCursorDown(1)

	assert.Equal(t, buffer.Loc{X: 0, Y: 2}, h.Cursor.Loc)
}

func TestMoveCursorDownWithSoftwrapFromLineHavingVirtualLinesAbove(t *testing.T) {
	b := buffer.NewBufferFromString("zero\nline\nthird", "", buffer.BTDefault)
	b.SetOptionNative("softwrap", true)
	assert.True(t, b.SetVirtualLinesJSON("test", `[
		{"line":1,"above":true,"text":"- removed 1","group":"diff-deleted"},
		{"line":1,"above":true,"text":"- removed 2","group":"diff-deleted"}
	]`, 1))

	w := display.NewBufWindow(0, 0, 80, 24, b)
	w.Resize(80, 24)
	h := NewBufPane(b, w, nil)

	h.Cursor.GotoLoc(buffer.Loc{X: 0, Y: 1})
	h.MoveCursorDown(1)

	assert.Equal(t, buffer.Loc{X: 0, Y: 2}, h.Cursor.Loc)
}

func TestMoveCursorDownWithoutSoftwrapSkipsVirtualRowsMappedToSameLine(t *testing.T) {
	b := buffer.NewBufferFromString("zero\none\ntwo", "", buffer.BTDefault)
	b.SetOptionNative("softwrap", false)
	assert.True(t, b.SetVirtualLinesJSON("test", `[
		{"line":1,"above":false,"text":"+ added 1","group":"diff-added"},
		{"line":1,"above":false,"text":"+ added 2","group":"diff-added"}
	]`, 1))

	w := display.NewBufWindow(0, 0, 80, 24, b)
	w.Resize(80, 24)
	h := NewBufPane(b, w, nil)

	h.Cursor.GotoLoc(buffer.Loc{X: 0, Y: 1})
	h.MoveCursorDown(1)

	assert.Equal(t, buffer.Loc{X: 0, Y: 2}, h.Cursor.Loc)
}

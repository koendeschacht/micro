package display

import (
	"testing"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
)

func init() {
	ulua.L = lua.NewState()
	config.InitRuntimeFiles(false)
	config.InitGlobalSettings()
	config.GlobalSettings["backup"] = false
	config.GlobalSettings["fastdirty"] = true
}

func TestVirtualLinesAffectSLocScrollAndLocMapping(t *testing.T) {
	assert := assert.New(t)

	b := buffer.NewBufferFromString("alpha\nbeta", "", buffer.BTDefault)
	b.SetOptionNative("softwrap", false)
	assert.True(b.SetVirtualLinesJSON("diff", `[
		{"line":1,"above":true,"text":"- deleted 1","group":"diff-deleted"},
		{"line":1,"above":true,"text":"- deleted 2","group":"diff-deleted"},
		{"line":1,"above":false,"text":"+ inserted","group":"diff-added"}
	]`, 1))

	w := NewBufWindow(0, 0, 80, 20, b)

	assert.Equal(SLoc{Line: 1, Row: 2}, w.SLocFromLoc(buffer.Loc{X: 0, Y: 1}))
	assert.Equal(SLoc{Line: 1, Row: 0}, w.Scroll(SLoc{Line: 0, Row: 0}, 1))
	assert.Equal(SLoc{Line: 1, Row: 2}, w.Scroll(SLoc{Line: 0, Row: 0}, 3))
	assert.Equal(3, w.Diff(SLoc{Line: 0, Row: 0}, SLoc{Line: 1, Row: 2}))

	assert.Equal(buffer.Loc{X: 0, Y: 1}, w.LocFromVLoc(VLoc{SLoc: SLoc{Line: 1, Row: 0}, VisualX: 0}))
	assert.Equal(buffer.Loc{X: 0, Y: 1}, w.LocFromVLoc(VLoc{SLoc: SLoc{Line: 1, Row: 1}, VisualX: 0}))
	assert.Equal(buffer.Loc{X: 4, Y: 1}, w.LocFromVLoc(VLoc{SLoc: SLoc{Line: 1, Row: 3}, VisualX: 0}))
}

func TestVirtualLineIDAtVisual(t *testing.T) {
	assert := assert.New(t)

	b := buffer.NewBufferFromString("alpha\nbeta", "", buffer.BTDefault)
	b.SetOptionNative("softwrap", false)
	assert.True(b.SetVirtualLinesJSON("comments", `[
		{"id":"above","line":1,"above":true,"text":"review header"},
		{"id":"above-2","line":1,"above":true,"text":"another header"},
		{"id":"below","line":1,"above":false,"text":"review body"}
	]`, 1))

	w := NewBufWindow(3, 2, 80, 20, b)
	assert.Equal("", w.VirtualLineIDAtVisual(3, 2))
	assert.Equal("above", w.VirtualLineIDAtVisual(3, 3))
	assert.Equal("above-2", w.VirtualLineIDAtVisual(3, 4))
	assert.Equal("", w.VirtualLineIDAtVisual(3, 5))
	assert.Equal("below", w.VirtualLineIDAtVisual(3, 6))
	assert.Equal("", w.VirtualLineIDAtVisual(3, 7))
	assert.Equal("", w.VirtualLineIDAtVisual(2, 3))
	assert.Equal("", w.VirtualLineIDAtVisual(3, 22))

	w.StartLine = SLoc{Line: 1, Row: 1}
	assert.Equal("above-2", w.VirtualLineIDAtVisual(3, 2))
	assert.Equal("", w.VirtualLineIDAtVisual(3, 3))
	assert.Equal("below", w.VirtualLineIDAtVisual(3, 4))
}

package display

import (
	"testing"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/stretchr/testify/assert"
)

func TestSetSidePaneJSONStoresEntries(t *testing.T) {
	assert := assert.New(t)

	b := buffer.NewBufferFromString("alpha\nbeta\n", "", buffer.BTDefault)
	b.SetOptionNative("ruler", false)
	w := NewBufWindow(0, 0, 80, 20, b)

	ok := w.SetSidePaneJSON(`{"width":12,"entries":[{"line":0,"text":"first","group":"comment"},{"line":1,"text":"second"}]}`, 3)

	assert.True(ok)
	if assert.NotNil(w.sidePane) {
		assert.Equal(12, w.sidePane.Width)
		assert.Equal(SidePaneLine{Text: "first", Group: "comment"}, w.sidePane.Entries[0])
		assert.Equal(SidePaneLine{Text: "second", Group: ""}, w.sidePane.Entries[1])
	}
	assert.Equal(12, w.gutterOffset)
	assert.Equal(68, w.bufWidth)
}

func TestSetSidePaneJSONRejectsOlderVersion(t *testing.T) {
	assert := assert.New(t)

	b := buffer.NewBufferFromString("alpha", "", buffer.BTDefault)
	b.SetOptionNative("ruler", false)
	w := NewBufWindow(0, 0, 40, 10, b)

	assert.True(w.SetSidePaneJSON(`{"width":10,"entries":[{"line":0,"text":"new"}]}`, 2))
	assert.False(w.SetSidePaneJSON(`{"width":8,"entries":[{"line":0,"text":"old"}]}`, 1))

	if assert.NotNil(w.sidePane) {
		assert.Equal(10, w.sidePane.Width)
		assert.Equal(SidePaneLine{Text: "new", Group: ""}, w.sidePane.Entries[0])
	}

	assert.True(w.SetSidePaneJSON("", 3))
	assert.Nil(w.sidePane)
	assert.Equal(3, w.sidePaneVersion)
	assert.False(w.SetSidePaneJSON(`{"width":8,"entries":[{"line":0,"text":"stale"}]}`, 2))
	assert.Nil(w.sidePane)
}

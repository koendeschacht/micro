package action

import (
	"testing"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/display"
	"github.com/micro-editor/micro/v2/internal/info"
	"github.com/stretchr/testify/assert"
)

func newCommandEditTestPane(text string, x int) *BufPane {
	config.InitRuntimeFiles(false)
	config.InitGlobalSettings()

	b := buffer.NewBufferFromString(text, "", buffer.BTDefault)
	b.GetActiveCursor().GotoLoc(buffer.Loc{X: x, Y: 0})
	w := display.NewBufWindow(0, 0, 80, 24, b)
	return NewBufPane(b, w, nil)
}

func TestExpandCommandEditPromptUsesCursorWord(t *testing.T) {
	h := newCommandEditTestPane("old_name = 1", 2)

	assert.Equal(t, "rename old_name", expandCommandEditPrompt(h, "rename {cursor-word}"))
}

func TestCursorWordUsesPreviousWordAtEnd(t *testing.T) {
	h := newCommandEditTestPane("old_name = 1", len("old_name"))

	assert.Equal(t, "old_name", cursorWord(h))
}

func TestCursorWordIsEmptyOutsideWord(t *testing.T) {
	h := newCommandEditTestPane("old_name = 1", len("old_name ="))

	assert.Equal(t, "", cursorWord(h))
}

func TestCommandEditActionSelectsPrefilledPrompt(t *testing.T) {
	ib := info.NewBuffer()
	InfoBar = NewInfoPane(ib, display.NewBufWindow(0, 0, 80, 1, ib.Buffer), nil)
	t.Cleanup(func() {
		InfoBar = nil
	})
	h := newCommandEditTestPane("old_name = 1", 2)

	CommandEditAction("rename {cursor-word}")(h)

	active := InfoBar.Buf.GetActiveCursor()
	assert.Equal(t, "> rename: ", InfoBar.Msg)
	assert.Equal(t, "old_name", InfoBar.PromptText())
	assert.True(t, active.HasSelection())
	assert.Equal(t, "old_name", string(active.GetSelection()))
}

package action

import (
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/display"
	"github.com/micro-editor/micro/v2/internal/info"
	"github.com/stretchr/testify/assert"
)

func TestPromptEditorEnterAddsLineAndCtrlEnterSubmits(t *testing.T) {
	config.InitRuntimeFiles(false)
	config.InitGlobalSettings()
	ib := info.NewBuffer()
	h := NewInfoPane(ib, display.NewBufWindow(0, 0, 80, 6, ib.Buffer), nil)
	var response string
	var canceled bool
	h.PromptEditor("Comment", "first", "test", 5, nil, func(resp string, wasCanceled bool) {
		response = resp
		canceled = wasCanceled
	})

	enter := tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone)
	h.HandleEvent(enter)
	assert.True(t, h.HasPrompt)
	assert.Equal(t, "first\n", h.PromptText())

	ctrlEnter := tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModCtrl)
	h.HandleEvent(ctrlEnter)
	assert.False(t, h.HasPrompt)
	assert.False(t, canceled)
	assert.Equal(t, "first\n", response)
}

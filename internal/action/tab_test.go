package action

import (
	"testing"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/display"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTabListSyncLayoutReservesInfoPopupHeightForSplitPanes(t *testing.T) {
	config.InitRuntimeFiles(false)
	config.InitGlobalSettings()

	_, err := screen.InitSimScreen()
	require.NoError(t, err)
	t.Cleanup(func() {
		screen.Screen.Fini()
		InfoBar = nil
		config.InfoBarOffset = nil
	})

	InfoBar = NewInfoBar()

	b := buffer.NewBufferFromString("top\nmiddle\nbottom", "", buffer.BTDefault)
	other := buffer.NewBufferFromString("alpha\nbeta\ngamma", "", buffer.BTDefault)
	tab := NewTabFromBuffer(0, 0, 80, 24, b)
	tab.CurPane().HSplitBuf(other)

	tabs := &TabList{TabWindow: display.NewTabWindow(80, 0), List: []*Tab{tab}}
	tabs.syncLayout()
	assert.Equal(t, 24, tab.Node.H)

	InfoBar.Message("hello")
	tabs.syncLayout()

	assert.Equal(t, 24, tab.Node.H)
	for _, pane := range tab.Panes {
		view := pane.GetView()
		assert.LessOrEqual(t, view.Y+view.Height, tab.Node.H)
	}
}

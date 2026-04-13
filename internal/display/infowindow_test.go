package display

import (
	"testing"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/info"
	"github.com/stretchr/testify/assert"
)

func init() {
	config.InitRuntimeFiles(false)
	config.InitGlobalSettings()
}

func TestInfoWindowPopupHeightIncludesSuggestionRow(t *testing.T) {
	ib := info.NewBuffer()
	ib.Prompt("> ", "", "Command", nil, nil)
	ib.HasSuggestions = true
	ib.Suggestions = []string{"open", "option"}

	iw := &InfoWindow{
		InfoBuf: ib,
		View:    &View{Width: 80, Y: 23},
	}

	assert.Equal(t, 5, iw.PopupHeight())
}

func TestBufWindowReservesInfoPopupHeightAboveStatusline(t *testing.T) {
	b := buffer.NewBufferFromString("one\ntwo\nthree", "", buffer.BTDefault)
	w := NewBufWindow(0, 0, 80, 24, b)

	config.InfoBarOffset = func() int { return 3 }
	t.Cleanup(func() {
		config.InfoBarOffset = nil
	})

	w.Resize(80, 24)

	assert.Equal(t, 20, w.bufHeight)
	assert.Equal(t, 24, w.Height)
}

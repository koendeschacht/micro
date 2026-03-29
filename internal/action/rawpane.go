package action

import (
	"fmt"
	"reflect"

	"github.com/gdamore/tcell/v3"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/display"
)

type RawPane struct {
	*BufPane
}

func NewRawPaneFromWin(b *buffer.Buffer, win display.BWindow, tab *Tab) *RawPane {
	rh := new(RawPane)
	rh.BufPane = NewBufPane(b, win, tab)

	return rh
}

func NewRawPane(tab *Tab) *RawPane {
	b := buffer.NewBufferFromString("", "", buffer.BTRaw)
	b.SetName("Event viewer")

	w := display.NewBufWindow(0, 0, 0, 0, b)

	return NewRawPaneFromWin(b, w, tab)
}

func (h *RawPane) HandleEvent(event tcell.Event) {
	switch e := event.(type) {
	case *tcell.EventKey:
		ke := keyEvent(e)
		if ke.code == tcell.KeyRune && ke.mod&tcell.ModCtrl != 0 && (ke.r == 'q' || ke.r == 'Q') {
			h.Quit()
		}
	}

	h.Buf.Insert(h.Cursor.Loc, reflect.TypeOf(event).String()[7:])

	e, err := ConstructEvent(event)
	if err == nil {
		h.Buf.Insert(h.Cursor.Loc, fmt.Sprintf(": %s", e.Name()))
	}

	h.Buf.Insert(h.Cursor.Loc, "\n")

	h.Relocate()
}

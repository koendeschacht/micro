package action

import (
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/display"
	"github.com/micro-editor/micro/v2/internal/info"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func init() {
	ulua.L = lua.NewState()
}

func TestFindEventLeavesSingleKeysAsSingleEvents(t *testing.T) {
	event, err := findEvent("Ctrl-e")
	require.NoError(t, err)
	_, ok := event.(KeyEvent)
	assert.True(t, ok)
}

func TestFindEventExpandsLeaderSequences(t *testing.T) {
	oldSettings := config.GlobalSettings
	config.GlobalSettings = config.DefaultAllSettings()
	t.Cleanup(func() {
		config.GlobalSettings = oldSettings
	})

	event, err := findEvent("<leader>cf")
	require.NoError(t, err)

	seq, ok := event.(KeySequenceEvent)
	require.True(t, ok)
	require.Len(t, seq.keys, 3)

	assert.Equal(t, KeyEvent{code: tcell.KeyRune, mod: tcell.ModCtrl, r: 'k'}, seq.keys[0])
	assert.Equal(t, KeyEvent{code: tcell.KeyRune, r: 'c'}, seq.keys[1])
	assert.Equal(t, KeyEvent{code: tcell.KeyRune, r: 'f'}, seq.keys[2])
	assert.Equal(t, "<Ctrl-k><c><f>", seq.Name())

	bracketed, err := findEvent("<leader><c><f>")
	require.NoError(t, err)
	assert.Equal(t, seq, bracketed)
}

func TestKeyTreeCursorsAreIndependent(t *testing.T) {
	tree := NewKeyTree()
	sequence := KeySequenceEvent{keys: []Event{
		KeyEvent{code: tcell.KeyRune, r: 'c'},
		KeyEvent{code: tcell.KeyRune, r: 'f'},
	}}

	tree.RegisterKeyBinding(sequence, func(Pane) bool { return true })

	first := tree.NewCursor()
	second := tree.NewCursor()

	action, more := tree.NextEvent(first, KeyEvent{code: tcell.KeyRune, r: 'c'}, nil)
	assert.Nil(t, action)
	assert.True(t, more)

	action, more = tree.NextEvent(second, KeyEvent{code: tcell.KeyRune, r: 'x'}, nil)
	assert.Nil(t, action)
	assert.False(t, more)

	action, more = tree.NextEvent(first, KeyEvent{code: tcell.KeyRune, r: 'f'}, nil)
	assert.NotNil(t, action)
	assert.False(t, more)
}

func TestFindEventSupportsCtrlShiftRuneBindings(t *testing.T) {
	event, err := findEvent("Ctrl-Shift-k")
	require.NoError(t, err)

	assert.Equal(t, KeyEvent{code: tcell.KeyRune, mod: tcell.ModCtrl | tcell.ModShift, r: 'K'}, event)
	assert.Equal(t, KeyEvent{code: tcell.KeyRune, mod: tcell.ModCtrl | tcell.ModShift, r: 'K'}, keyEvent(tcell.NewEventKey(tcell.KeyRune, "K", tcell.ModCtrl|tcell.ModShift)))
}

func TestFindEventSupportsCtrlPunctuationBindings(t *testing.T) {
	event, err := findEvent("Ctrl-[")
	require.NoError(t, err)

	assert.Equal(t, KeyEvent{code: tcell.KeyRune, mod: tcell.ModCtrl, r: '['}, event)
}

func TestFindEventSupportsCtrlSpaceBindings(t *testing.T) {
	event, err := findEvent("Ctrl-Space")
	require.NoError(t, err)

	assert.Equal(t, KeyEvent{code: tcell.KeyRune, mod: tcell.ModCtrl, r: ' '}, event)
}

func TestBufPaneBracketedPasteBuffersText(t *testing.T) {
	config.InitRuntimeFiles(false)
	config.InitGlobalSettings()
	InfoBar = &InfoPane{InfoBuf: info.NewBuffer()}

	b := buffer.NewBufferFromString("", "", buffer.BTDefault)
	w := display.NewBufWindow(0, 0, 80, 24, b)
	h := NewBufPane(b, w, nil)

	h.HandleEvent(tcell.NewEventPaste(true))
	h.HandleEvent(tcell.NewEventKey(tcell.KeyRune, "a", tcell.ModNone))
	h.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone))
	h.HandleEvent(tcell.NewEventKey(tcell.KeyRune, "b", tcell.ModNone))
	h.HandleEvent(tcell.NewEventPaste(false))

	assert.Equal(t, "a\nb", string(b.Bytes()))
	assert.True(t, h.Undo())
	assert.Equal(t, "", string(b.Bytes()))
}

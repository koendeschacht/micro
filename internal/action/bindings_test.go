package action

import (
	"testing"

	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/tcell/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	assert.Equal(t, KeyEvent{code: tcell.KeyCtrlK, mod: tcell.ModCtrl}, seq.keys[0])
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

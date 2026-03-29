package action

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell/v3"
)

type Event interface {
	Name() string
}

// KeyEvent is a key event containing a key code,
// some possible modifiers (alt, ctrl, etc...) and
// a rune if it was simply a character press
// Note: to be compatible with tcell events,
// for ctrl keys r=code
type KeyEvent struct {
	code tcell.Key
	mod  tcell.ModMask
	r    rune
	any  bool
}

func metaToAlt(mod tcell.ModMask) tcell.ModMask {
	if mod&tcell.ModMeta != 0 {
		mod &= ^tcell.ModMeta
		mod |= tcell.ModAlt
	}
	return mod
}

func keyEvent(e *tcell.EventKey) KeyEvent {
	key, mod, r := normalizeKey(e.Key(), metaToAlt(e.Modifiers()), e.Str())
	ke := KeyEvent{
		code: key,
		mod:  mod,
		r:    r,
	}
	return ke
}

func normalizeKey(key tcell.Key, mod tcell.ModMask, str string) (tcell.Key, tcell.ModMask, rune) {
	if r, ok := legacyCtrlRune(key); ok {
		return tcell.KeyRune, mod | tcell.ModCtrl, r
	}

	if key == tcell.KeyRune && str != "" {
		r, _ := utf8.DecodeRuneInString(str)
		return key, mod, r
	}

	return key, mod, 0
}

func firstRune(str string) (rune, bool) {
	if str == "" {
		return 0, false
	}
	r, _ := utf8.DecodeRuneInString(str)
	if r == utf8.RuneError {
		return 0, false
	}
	return r, true
}

func pasteEventText(e *tcell.EventKey) (string, bool) {
	switch e.Key() {
	case tcell.KeyRune:
		return e.Str(), e.Str() != ""
	case tcell.KeyEnter:
		return "\n", true
	case tcell.KeyTab:
		return "\t", true
	}

	return "", false
}

func legacyCtrlRune(key tcell.Key) (rune, bool) {
	switch key {
	case tcell.KeyCtrlA:
		return 'a', true
	case tcell.KeyCtrlB:
		return 'b', true
	case tcell.KeyCtrlC:
		return 'c', true
	case tcell.KeyCtrlD:
		return 'd', true
	case tcell.KeyCtrlE:
		return 'e', true
	case tcell.KeyCtrlF:
		return 'f', true
	case tcell.KeyCtrlG:
		return 'g', true
	case tcell.KeyCtrlI:
		return 'i', true
	case tcell.KeyCtrlJ:
		return 'j', true
	case tcell.KeyCtrlK:
		return 'k', true
	case tcell.KeyCtrlL:
		return 'l', true
	case tcell.KeyCtrlN:
		return 'n', true
	case tcell.KeyCtrlO:
		return 'o', true
	case tcell.KeyCtrlP:
		return 'p', true
	case tcell.KeyCtrlQ:
		return 'q', true
	case tcell.KeyCtrlR:
		return 'r', true
	case tcell.KeyCtrlS:
		return 's', true
	case tcell.KeyCtrlT:
		return 't', true
	case tcell.KeyCtrlU:
		return 'u', true
	case tcell.KeyCtrlV:
		return 'v', true
	case tcell.KeyCtrlW:
		return 'w', true
	case tcell.KeyCtrlX:
		return 'x', true
	case tcell.KeyCtrlY:
		return 'y', true
	case tcell.KeyCtrlZ:
		return 'z', true
	}

	return 0, false
}

func (k KeyEvent) Name() string {
	if k.any {
		return "<any>"
	}
	s := ""
	m := []string{}
	if k.mod&tcell.ModShift != 0 {
		m = append(m, "Shift")
	}
	if k.mod&tcell.ModAlt != 0 {
		m = append(m, "Alt")
	}
	if k.mod&tcell.ModMeta != 0 {
		m = append(m, "Meta")
	}
	if k.mod&tcell.ModCtrl != 0 {
		m = append(m, "Ctrl")
	}

	ok := false
	if s, ok = tcell.KeyNames[k.code]; !ok {
		if k.code == tcell.KeyRune {
			s = string(k.r)
		} else {
			s = fmt.Sprintf("Key[%d]", k.code)
		}
	}
	if len(m) != 0 {
		if k.mod&tcell.ModCtrl != 0 && strings.HasPrefix(s, "Ctrl-") {
			s = s[5:]
			if len(s) == 1 {
				s = strings.ToLower(s)
			}
		}
		return fmt.Sprintf("%s-%s", strings.Join(m, "-"), s)
	}
	return s
}

// A KeySequence defines a list of consecutive
// events. All events in the sequence must be KeyEvents
// or MouseEvents.
type KeySequenceEvent struct {
	keys []Event
}

func (k KeySequenceEvent) Name() string {
	buf := bytes.Buffer{}
	for _, e := range k.keys {
		buf.WriteByte('<')
		buf.WriteString(e.Name())
		buf.WriteByte('>')
	}
	return buf.String()
}

type MouseState int

const (
	MousePress = iota
	MouseDrag
	MouseRelease
)

// MouseEvent is a mouse event with a mouse button and
// any possible key modifiers
type MouseEvent struct {
	btn   tcell.ButtonMask
	mod   tcell.ModMask
	state MouseState
}

func (m MouseEvent) Name() string {
	mod := ""
	if m.mod&tcell.ModShift != 0 {
		mod = "Shift-"
	}
	if m.mod&tcell.ModAlt != 0 {
		mod = "Alt-"
	}
	if m.mod&tcell.ModMeta != 0 {
		mod = "Meta-"
	}
	if m.mod&tcell.ModCtrl != 0 {
		mod = "Ctrl-"
	}

	state := ""
	switch m.state {
	case MouseDrag:
		state = "Drag"
	case MouseRelease:
		state = "Release"
	}

	for k, v := range mouseEvents {
		if v == m.btn {
			return fmt.Sprintf("%s%s%s", mod, k, state)
		}
	}
	return ""
}

// ConstructEvent takes a tcell event and returns a micro
// event. Note that tcell events can't express certain
// micro events such as key sequences. This function is
// mostly used for debugging/raw panes or constructing
// intermediate micro events while parsing a sequence.
func ConstructEvent(event tcell.Event) (Event, error) {
	switch e := event.(type) {
	case *tcell.EventKey:
		return keyEvent(e), nil
	case *tcell.EventMouse:
		return MouseEvent{
			btn: e.Buttons(),
			mod: metaToAlt(e.Modifiers()),
		}, nil
	}
	return nil, errors.New("No micro event equivalent")
}

// A Handler will take a tcell event and execute it
// appropriately
type Handler interface {
	HandleEvent(tcell.Event)
	HandleCommand(string)
}

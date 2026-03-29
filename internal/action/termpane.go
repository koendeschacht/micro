package action

import (
	"errors"
	"runtime"
	"strings"

	"github.com/gdamore/tcell/v3"
	"github.com/micro-editor/micro/v2/internal/clipboard"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/display"
	"github.com/micro-editor/micro/v2/internal/keymenu"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/shell"
	"github.com/micro-editor/terminal"
)

type TermKeyAction func(*TermPane)

var TermBindings *KeyTree

func init() {
	TermBindings = NewKeyTree()
}

func TermKeyActionGeneral(a TermKeyAction) PaneKeyAction {
	return func(p Pane) bool {
		a(p.(*TermPane))
		return true
	}
}

func TermMapEvent(k Event, action string) {
	config.Bindings["terminal"][k.Name()] = action
	keymenu.SetBinding("terminal", k.Name(), action, keyMenuSequenceFromEvent(k))

	switch e := k.(type) {
	case KeyEvent, KeySequenceEvent:
		termMapKey(e, action)
	case MouseEvent:
		termMapMouse(e, action)
	}
}

func termMapKey(k Event, action string) {
	if f, ok := TermKeyActions[action]; ok {
		TermBindings.RegisterKeyBinding(k, TermKeyActionGeneral(f))
	}
}

func termMapMouse(k MouseEvent, action string) {
	// TODO: map mouse
	termMapKey(k, action)
}

type TermPane struct {
	*shell.Terminal
	display.Window

	mouseReleased bool
	id            uint64
	tab           *Tab
	keyCursor     *KeyTreeCursor
}

func NewTermPane(x, y, w, h int, t *shell.Terminal, id uint64, tab *Tab) (*TermPane, error) {
	if !TermEmuSupported {
		return nil, errors.New("Terminal emulator is not supported on this system")
	}

	th := new(TermPane)
	th.Terminal = t
	th.id = id
	th.mouseReleased = true
	th.Window = display.NewTermWindow(x, y, w, h, t)
	th.tab = tab
	return th, nil
}

func (t *TermPane) ID() uint64 {
	return t.id
}

func (t *TermPane) SetID(i uint64) {
	t.id = i
}

func (t *TermPane) Name() string {
	return t.Terminal.Name()
}

func (t *TermPane) SetTab(tab *Tab) {
	t.tab = tab
}

func (t *TermPane) Tab() *Tab {
	return t.tab
}

func (t *TermPane) bindingCursor() *KeyTreeCursor {
	if t.keyCursor == nil || t.keyCursor.tree != TermBindings {
		t.keyCursor = TermBindings.NewCursor()
	}
	return t.keyCursor
}

func (t *TermPane) Close() {}

// Quit closes this termpane
func (t *TermPane) Quit() {
	t.Close()
	if len(MainTab().Panes) > 1 {
		t.Unsplit()
	} else if len(Tabs.List) > 1 {
		Tabs.RemoveTab(t.id)
	} else {
		screen.Screen.Fini()
		InfoBar.Close()
		runtime.Goexit()
	}
}

// Unsplit removes this split
func (t *TermPane) Unsplit() {
	n := MainTab().GetNode(t.id)
	n.Unsplit()

	MainTab().RemovePane(MainTab().GetPane(t.id))
	MainTab().Resize()
	MainTab().SetActive(len(MainTab().Panes) - 1)
}

// HandleEvent handles a tcell event by forwarding it to the terminal emulator
// If the event is a mouse event and the program running in the emulator
// does not have mouse support, the emulator will support selections and
// copy-paste
func (t *TermPane) HandleEvent(event tcell.Event) {
	if e, ok := event.(*tcell.EventKey); ok {
		ke := keyEvent(e)
		action, more := TermBindings.NextEvent(t.bindingCursor(), ke, nil)

		if !more {
			if action != nil {
				action(t)
				TermBindings.ResetEvents(t.bindingCursor())
				return
			}
			TermBindings.ResetEvents(t.bindingCursor())
		}

		if more {
			return
		}

		if t.Status == shell.TTDone {
			switch {
			case e.Key() == tcell.KeyEscape:
				t.Close()
				t.Quit()
			case ke.code == tcell.KeyRune && ke.mod&tcell.ModCtrl != 0 && strings.EqualFold(string(ke.r), "q"):
				t.Close()
				t.Quit()
			case e.Key() == tcell.KeyEnter:
				t.Close()
				t.Quit()
			}
		}
		if ke.code == tcell.KeyRune && ke.mod&tcell.ModCtrl != 0 && strings.EqualFold(string(ke.r), "c") && t.HasSelection() {
			clipboard.Write(t.GetSelection(t.GetView().Width), clipboard.ClipboardReg)
			InfoBar.Message("Copied selection to clipboard")
		} else if t.Status != shell.TTDone {
			t.WriteString(encodeTerminalKey(e, ke))
		}
	} else if e, ok := event.(*tcell.EventPaste); ok {
		if t.Status != shell.TTDone {
			if e.Start() {
				t.WriteString("\x1b[200~")
			} else if e.End() {
				t.WriteString("\x1b[201~")
			}
		}
	} else if e, ok := event.(*tcell.EventMouse); !ok || t.State.Mode(terminal.ModeMouseMask) {
		// t.WriteString(event.EscSeq())
	} else {
		x, y := e.Position()
		v := t.GetView()
		x -= v.X
		y -= v.Y

		if e.Buttons() == tcell.Button1 {
			if !t.mouseReleased {
				// drag
				t.Selection[1].X = x
				t.Selection[1].Y = y
			} else {
				t.Selection[0].X = x
				t.Selection[0].Y = y
				t.Selection[1].X = x
				t.Selection[1].Y = y
			}

			t.mouseReleased = false
		} else if e.Buttons() == tcell.ButtonNone {
			if !t.mouseReleased {
				t.Selection[1].X = x
				t.Selection[1].Y = y
			}
			t.mouseReleased = true
		}
	}
}

func encodeTerminalKey(e *tcell.EventKey, ke KeyEvent) string {
	if ke.code == tcell.KeyRune {
		if ctrl, ok := ctrlRune(ke.r); ok && ke.mod&tcell.ModCtrl != 0 {
			if ke.mod&tcell.ModAlt != 0 {
				return "\x1b" + string(ctrl)
			}
			return string(ctrl)
		}

		if e.Str() == "" {
			return ""
		}
		if ke.mod&tcell.ModAlt != 0 {
			return "\x1b" + e.Str()
		}
		return e.Str()
	}

	switch e.Key() {
	case tcell.KeyEnter:
		return "\r"
	case tcell.KeyTab:
		return "\t"
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		return "\x7f"
	case tcell.KeyEscape:
		return "\x1b"
	case tcell.KeyUp:
		return "\x1b[A"
	case tcell.KeyDown:
		return "\x1b[B"
	case tcell.KeyRight:
		return "\x1b[C"
	case tcell.KeyLeft:
		return "\x1b[D"
	case tcell.KeyHome:
		return "\x1b[H"
	case tcell.KeyEnd:
		return "\x1b[F"
	case tcell.KeyPgUp:
		return "\x1b[5~"
	case tcell.KeyPgDn:
		return "\x1b[6~"
	case tcell.KeyInsert:
		return "\x1b[2~"
	case tcell.KeyDelete:
		return "\x1b[3~"
	}

	return ""
}

func ctrlRune(r rune) (rune, bool) {
	switch {
	case r >= 'a' && r <= 'z':
		return r - 'a' + 1, true
	case r >= 'A' && r <= 'Z':
		return r - 'A' + 1, true
	case r == ' ':
		return 0, true
	case r == '[':
		return 27, true
	case r == '\\':
		return 28, true
	case r == ']':
		return 29, true
	case r == '^':
		return 30, true
	case r == '_':
		return 31, true
	case r == '?':
		return 127, true
	default:
		return 0, false
	}
}

// HandleTermClose is called when a terminal has finished its job
// and should be closed. If that terminal is this termpane's terminal,
// HandleTermClose will close the terminal and the termpane itself.
func (t *TermPane) HandleTermClose() {
	if t.Status == shell.TTClose {
		t.Quit()
	}
}

// Exit closes the termpane
func (t *TermPane) Exit() {
	t.Terminal.Close()
	t.Quit()
}

// CommandMode opens the termpane's command mode
func (t *TermPane) CommandMode() {
	InfoBar.Prompt("> ", "", "TerminalCommand", nil, func(resp string, canceled bool) {
		if !canceled {
			t.HandleCommand(resp)
		}
	})
}

// NextSplit moves to the next split
func (t *TermPane) NextSplit() {
	a := t.tab.active
	if a < len(t.tab.Panes)-1 {
		a++
	} else {
		a = 0
	}

	t.tab.SetActive(a)
}

// HandleCommand handles a command for the term pane
func (t *TermPane) HandleCommand(input string) {
	InfoBar.Error("Commands are unsupported in term for now")
}

// TermKeyActions contains the list of all possible key actions the termpane could execute
var TermKeyActions = map[string]TermKeyAction{
	"Exit":        (*TermPane).Exit,
	"CommandMode": (*TermPane).CommandMode,
	"NextSplit":   (*TermPane).NextSplit,
}

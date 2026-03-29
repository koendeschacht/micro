package screen

import (
	"log"
	"os"
	"sync"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/micro-editor/micro/v2/internal/config"
)

// Screen is the tcell screen we use to draw to the terminal
// Synchronization is used because we poll the screen on a separate
// thread and sometimes the screen is shut down by the main thread
// (for example on TermMessage) so we don't want to poll a nil/shutdown
// screen. TODO: maybe we should worry about polling and drawing at the
// same time too.
var Screen tcell.Screen

// Events is the channel of tcell events
var Events chan (tcell.Event)

// RestartCallback is called when the screen is restarted after it was
// temporarily shut down
var RestartCallback func()

// The lock is necessary since the screen is polled on a separate thread
var lock sync.Mutex

// drawChan is a channel that will cause the screen to redraw when
// written to even if no event user event has occurred
var drawChan chan bool

// Lock locks the screen lock
func Lock() {
	lock.Lock()
}

// Unlock unlocks the screen lock
func Unlock() {
	lock.Unlock()
}

// Redraw schedules a redraw with the draw channel
func Redraw() {
	select {
	case drawChan <- true:
	default:
		// channel is full
	}
}

// DrawChan returns the draw channel
func DrawChan() chan bool {
	return drawChan
}

type screenCell struct {
	x, y  int
	r     rune
	combc []rune
	style tcell.Style
}

var lastCursor screenCell

// ShowFakeCursor displays a cursor at the given position by modifying the
// style of the given column instead of actually using the terminal cursor
// This can be useful in certain terminals such as the windows console where
// modifying the cursor location is slow and frequent modifications cause flashing
// This keeps track of the most recent fake cursor location and resets it when
// a new fake cursor location is specified
func ShowFakeCursor(x, y int) {
	r, combc, style := getContent(x, y)
	Screen.SetContent(lastCursor.x, lastCursor.y, lastCursor.r, lastCursor.combc, lastCursor.style)
	Screen.SetContent(x, y, r, combc, config.DefStyle.Reverse(true))

	lastCursor.x, lastCursor.y = x, y
	lastCursor.r = r
	lastCursor.combc = combc
	lastCursor.style = style
}

func UseFake() bool {
	return config.GetGlobalOption("fakecursor").(bool)
}

// ShowFakeCursorMulti is the same as ShowFakeCursor except it does not
// reset previous locations of the cursor
// Fake cursors are also necessary to display multiple cursors
func ShowFakeCursorMulti(x, y int) {
	r, _, _ := getContent(x, y)
	Screen.SetContent(x, y, r, nil, config.DefStyle.Reverse(true))
}

func getContent(x, y int) (rune, []rune, tcell.Style) {
	str, style, _ := Screen.Get(x, y)
	runes := []rune(str)
	if len(runes) == 0 {
		return ' ', nil, style
	}
	return runes[0], runes[1:], style
}

// ShowCursor puts the cursor at the given location using a fake cursor
// if enabled or using the terminal cursor otherwise
// By default only the windows console will use a fake cursor
func ShowCursor(x, y int) {
	if UseFake() {
		ShowFakeCursor(x, y)
	} else {
		Screen.ShowCursor(x, y)
	}
}

// SetContent sets a cell at a point on the screen and makes sure that it is
// synced with the last cursor location
func SetContent(x, y int, mainc rune, combc []rune, style tcell.Style) {
	Screen.SetContent(x, y, mainc, combc, style)
	if UseFake() && lastCursor.x == x && lastCursor.y == y {
		lastCursor.r = mainc
		lastCursor.style = style
		lastCursor.combc = combc
	}
}

// RegisterRawSeq is a no-op. Raw escape bindings are not supported on tcell v3.
func RegisterRawSeq(r string) {
}

// UnregisterRawSeq is a no-op. Raw escape bindings are not supported on tcell v3.
func UnregisterRawSeq(r string) {
}

// TempFini shuts the screen down temporarily
func TempFini() bool {
	screenWasNil := Screen == nil

	if !screenWasNil {
		Screen.Fini()
		Lock()
		Screen = nil
	}
	return screenWasNil
}

// TempStart restarts the screen after it was temporarily disabled
func TempStart(screenWasNil bool) {
	if !screenWasNil {
		Init()
		Unlock()

		if RestartCallback != nil {
			RestartCallback()
		}
	}
}

// Init creates and initializes the tcell screen
func Init() error {
	drawChan = make(chan bool, 8)

	// Should we enable true color?
	truecolor := config.GetGlobalOption("truecolor").(string)
	if truecolor == "on" || (truecolor == "auto" && os.Getenv("MICRO_TRUECOLOR") == "1") {
		os.Setenv("TCELL_TRUECOLOR", "enable")
	} else if truecolor == "off" {
		os.Setenv("TCELL_TRUECOLOR", "disable")
	} else {
		// For "auto", tcell already autodetects truecolor by default
	}

	var oldTerm string
	modifiedTerm := false
	setXterm := func() {
		oldTerm = os.Getenv("TERM")
		os.Setenv("TERM", "xterm-256color")
		modifiedTerm = true
	}

	if config.GetGlobalOption("xterm").(bool) {
		setXterm()
	}

	// Initilize tcell
	var err error
	Screen, err = tcell.NewScreen()
	if err != nil {
		log.Println("Warning: during screen initialization:", err)
		log.Println("Falling back to TERM=xterm-256color")
		setXterm()
		Screen, err = tcell.NewScreen()
		if err != nil {
			return err
		}
	}
	if err = Screen.Init(); err != nil {
		return err
	}

	if config.GetGlobalOption("paste").(bool) {
		Screen.EnablePaste()
	} else {
		Screen.DisablePaste()
	}

	// restore TERM
	if modifiedTerm {
		os.Setenv("TERM", oldTerm)
	}

	if config.GetGlobalOption("mouse").(bool) {
		Screen.EnableMouse()
	}

	return nil
}

type SimulationScreen interface {
	tcell.Screen
	InjectResize()
	InjectKey(tcell.Key, string, tcell.ModMask)
	InjectKeyBytes([]byte)
	InjectMouse(int, int, tcell.ButtonMask, tcell.ModMask)
}

type simScreen struct {
	tcell.Screen
}

func (s *simScreen) InjectResize() {
	width, height := s.Size()
	s.EventQ() <- tcell.NewEventResize(width, height)
}

func (s *simScreen) InjectKey(key tcell.Key, str string, mod tcell.ModMask) {
	if r, ok := ctrlKeyRune(key); ok {
		key = tcell.KeyRune
		str = string(r)
		mod |= tcell.ModCtrl
	}
	s.EventQ() <- tcell.NewEventKey(key, str, mod)
}

func (s *simScreen) InjectKeyBytes(data []byte) {
	for _, r := range string(data) {
		s.EventQ() <- tcell.NewEventKey(tcell.KeyRune, string(r), tcell.ModNone)
	}
}

func (s *simScreen) InjectMouse(x, y int, buttons tcell.ButtonMask, mod tcell.ModMask) {
	s.EventQ() <- tcell.NewEventMouse(x, y, buttons, mod)
}

func ctrlKeyRune(key tcell.Key) (rune, bool) {
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

// InitSimScreen initializes a simulation screen for testing purposes
func InitSimScreen() (SimulationScreen, error) {
	drawChan = make(chan bool, 8)

	term := vt.NewMockTerm(vt.MockOptSize{X: 80, Y: 24})
	s, err := tcell.NewTerminfoScreenFromTty(term)
	if err != nil {
		return nil, err
	}
	if err = s.Init(); err != nil {
		return nil, err
	}

	Screen = s
	for len(s.EventQ()) > 0 {
		<-s.EventQ()
	}

	if config.GetGlobalOption("mouse").(bool) {
		Screen.EnableMouse()
	}

	return &simScreen{Screen: s}, nil
}

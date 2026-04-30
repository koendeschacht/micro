package info

import (
	"fmt"
	"time"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/screen"
)

const (
	defaultInfoMessageDuration  = 2 * time.Second
	defaultErrorMessageDuration = 10 * time.Second
)

// The InfoBuf displays messages and other info at the bottom of the screen.
// It is represented as a buffer and a message with a style.
type InfoBuf struct {
	*buffer.Buffer

	HasPrompt  bool
	HasMessage bool
	HasError   bool
	HasYN      bool

	PromptType       string
	PromptInputRows  int
	PromptBufferMode bool
	PromptUseHistory bool
	MsgKind          MsgKind
	ExpiresAt        time.Time

	Msg    string
	YNResp bool

	Popup PopupState

	// This map stores the history for all the different kinds of uses Prompt has
	// It's a map of history type -> history array
	History    map[string][]string
	HistoryNum int
	// HistorySearch indicates whether we are searching for history items
	// beginning with HistorySearchPrefix
	HistorySearch       bool
	HistorySearchPrefix string

	// Is the current message a message from the gutter
	HasGutter bool
	KeyMenu   []KeyMenuEntry

	PromptCallback func(resp string, canceled bool)
	EventCallback  func(resp string)
	YNCallback     func(yes bool, canceled bool)
}

func (i *InfoBuf) PromptText() string {
	return string(i.Buffer.Substr(i.Start(), i.End()))
}

func (i *InfoBuf) beginPrompt(prompt string, msg string, ptype string, inputRows int, useHistory bool, bufferMode bool, eventcb func(string), donecb func(string, bool)) {
	if i.HasPrompt {
		i.DonePrompt(true)
	}

	if inputRows < 1 {
		inputRows = 1
	}

	if useHistory {
		if _, ok := i.History[ptype]; !ok {
			i.History[ptype] = []string{""}
		} else {
			i.History[ptype] = append(i.History[ptype], "")
		}
		i.HistoryNum = len(i.History[ptype]) - 1
		i.HistorySearch = false
	} else {
		i.HistoryNum = 0
		i.HistorySearch = false
		i.HistorySearchPrefix = ""
	}

	i.PromptType = ptype
	i.PromptInputRows = inputRows
	i.PromptBufferMode = bufferMode
	i.PromptUseHistory = useHistory
	i.setPopup(PopupState{
		Kind:    PopupPrompt,
		Source:  PopupSourceUser,
		MsgKind: MsgInfo,
		Text:    prompt,
	})
	i.HasYN = false
	i.YNCallback = nil
	i.PromptCallback = donecb
	i.EventCallback = eventcb
	i.Replace(i.Start(), i.End(), msg)
	i.Buffer.GetActiveCursor().GotoLoc(i.End())
}

type KeyMenuEntry struct {
	Key   string
	Label string
}

type MsgKind int

const (
	MsgInfo MsgKind = iota
	MsgSuccess
	MsgError
)

type PopupKind int

const (
	PopupNone PopupKind = iota
	PopupMessage
	PopupPrompt
)

type PopupSource int

const (
	PopupSourceUser PopupSource = iota
	PopupSourceCursor
	PopupSourceDiagnostic
)

type PopupState struct {
	Kind      PopupKind
	Source    PopupSource
	MsgKind   MsgKind
	Text      string
	ExpiresAt time.Time
}

var RootKeyMenuEntries func() []KeyMenuEntry

func (i *InfoBuf) SetKeyMenu(entries []KeyMenuEntry) {
	i.KeyMenu = append(i.KeyMenu[:0], entries...)
}

func (i *InfoBuf) ClearKeyMenu() {
	i.KeyMenu = i.KeyMenu[:0]
}

func (i *InfoBuf) setPopup(p PopupState) {
	i.Popup = p
	i.Msg = p.Text
	i.MsgKind = p.MsgKind
	i.ExpiresAt = p.ExpiresAt
	i.HasPrompt = p.Kind == PopupPrompt
	i.HasMessage = p.Kind == PopupMessage && p.MsgKind != MsgError
	i.HasError = p.Kind == PopupMessage && p.MsgKind == MsgError
	i.HasGutter = p.Source == PopupSourceCursor || p.Source == PopupSourceDiagnostic
}

func (i *InfoBuf) clearPopup() {
	i.setPopup(PopupState{MsgKind: MsgInfo})
}

func (i *InfoBuf) showMessage(kind MsgKind, source PopupSource, msg string, expiresAt time.Time) {
	i.setPopup(PopupState{
		Kind:      PopupMessage,
		Source:    source,
		MsgKind:   kind,
		Text:      msg,
		ExpiresAt: expiresAt,
	})
}

func (i *InfoBuf) showTransientMessage(kind MsgKind, source PopupSource, msg string) {
	duration := transientMessageDuration(kind)
	expiresAt := time.Time{}
	if duration > 0 {
		expiresAt = time.Now().Add(duration)
		time.AfterFunc(duration, screen.Redraw)
	}
	i.showMessage(kind, source, msg, expiresAt)
}

func transientMessageDuration(kind MsgKind) time.Duration {
	option := "infomessagetimeout"
	fallback := defaultInfoMessageDuration
	if kind == MsgError {
		option = "errormessagetimeout"
		fallback = defaultErrorMessageDuration
	}

	seconds, ok := config.GetGlobalOption(option).(float64)
	if !ok {
		return fallback
	}
	if seconds <= 0 {
		return 0
	}

	return time.Duration(seconds * float64(time.Second))
}

func (i *InfoBuf) MessageExpired() bool {
	return !i.ExpiresAt.IsZero() && time.Now().After(i.ExpiresAt)
}

func (i *InfoBuf) ExpireMessage() {
	if i.HasPrompt || !i.MessageExpired() {
		return
	}
	i.ClearMessage()
}

func (i *InfoBuf) ClearMessage() {
	if i.HasPrompt {
		return
	}
	i.clearPopup()
}

// NewBuffer returns a new infobuffer
func NewBuffer() *InfoBuf {
	ib := new(InfoBuf)
	ib.History = make(map[string][]string)

	ib.Buffer = buffer.NewBufferFromString("", "", buffer.BTInfo)
	ib.LoadHistory()

	return ib
}

// Close performs any cleanup necessary when shutting down the infobuffer
func (i *InfoBuf) Close() {
	i.SaveHistory()
}

// Message sends a message to the user
func (i *InfoBuf) Message(msg ...any) {
	// only display a new message if there isn't an active prompt
	// this is to prevent overwriting an existing prompt to the user
	if !i.HasPrompt {
		i.showTransientMessage(MsgInfo, PopupSourceUser, fmt.Sprint(msg...))
	}
}

// Success sends a success message to the user.
func (i *InfoBuf) Success(msg ...any) {
	if !i.HasPrompt {
		i.showTransientMessage(MsgSuccess, PopupSourceUser, fmt.Sprint(msg...))
	}
}

// GutterMessage displays a message and marks it as a gutter message
func (i *InfoBuf) GutterMessage(msg ...any) {
	if !i.HasPrompt {
		i.showTransientMessage(MsgInfo, PopupSourceCursor, fmt.Sprint(msg...))
	}
}

func (i *InfoBuf) DiagnosticMessage(kind buffer.MsgType, msg ...any) {
	if i.HasPrompt {
		return
	}

	msgKind := MsgInfo
	if kind == buffer.MTError {
		msgKind = MsgError
	}

	i.showMessage(msgKind, PopupSourceDiagnostic, fmt.Sprint(msg...), time.Time{})
}

// ClearGutter clears the info bar and unmarks the message
func (i *InfoBuf) ClearGutter() {
	i.ClearMessage()
}

// Error sends an error message to the user
func (i *InfoBuf) Error(msg ...any) {
	// only display a new message if there isn't an active prompt
	// this is to prevent overwriting an existing prompt to the user
	if !i.HasPrompt {
		i.showTransientMessage(MsgError, PopupSourceUser, fmt.Sprint(msg...))
	}
	// TODO: add to log?
}

// Prompt starts a prompt for the user, it takes a prompt, a possibly partially filled in msg
// and callbacks executed when the user executes an event and when the user finishes the prompt
// The eventcb passes the current user response as the argument and donecb passes the user's message
// and a boolean indicating if the prompt was canceled
func (i *InfoBuf) Prompt(prompt string, msg string, ptype string, eventcb func(string), donecb func(string, bool)) {
	i.beginPrompt(prompt, msg, ptype, 1, true, false, eventcb, donecb)
}

// PromptBuffer starts a popup prompt backed by a multiline buffer.
func (i *InfoBuf) PromptBuffer(prompt string, msg string, ptype string, inputRows int, eventcb func(string), donecb func(string, bool)) {
	i.beginPrompt(prompt, msg, ptype, inputRows, false, true, eventcb, donecb)
}

// YNPrompt creates a yes or no prompt, and the callback returns the yes/no result and whether
// the prompt was canceled
func (i *InfoBuf) YNPrompt(prompt string, donecb func(bool, bool)) {
	if i.HasPrompt {
		i.DonePrompt(true)
	}

	i.setPopup(PopupState{
		Kind:    PopupPrompt,
		Source:  PopupSourceUser,
		MsgKind: MsgInfo,
		Text:    prompt,
	})
	i.HasYN = true
	i.PromptInputRows = 1
	i.PromptBufferMode = false
	i.PromptUseHistory = false
	i.PromptCallback = nil
	i.EventCallback = nil
	i.YNCallback = donecb
}

// DonePrompt finishes the current prompt and indicates whether or not it was canceled
func (i *InfoBuf) DonePrompt(canceled bool) {
	hadYN := i.HasYN
	promptUseHistory := i.PromptUseHistory
	resp := i.PromptText()
	i.HasYN = false
	i.PromptInputRows = 1
	i.PromptBufferMode = false
	i.PromptUseHistory = false
	i.clearPopup()
	if !hadYN {
		if i.PromptCallback != nil {
			if canceled {
				i.Replace(i.Start(), i.End(), "")
				if promptUseHistory {
					h := i.History[i.PromptType]
					i.History[i.PromptType] = h[:len(h)-1]
				}
				i.PromptCallback("", true)
			} else {
				i.Replace(i.Start(), i.End(), "")
				if promptUseHistory {
					h := i.History[i.PromptType]
					h[len(h)-1] = resp

					// avoid duplicates
					for j := len(h) - 2; j >= 0; j-- {
						if h[j] == h[len(h)-1] {
							i.History[i.PromptType] = append(h[:j], h[j+1:]...)
							break
						}
					}
				}

				i.PromptCallback(resp, false)
			}
			// i.PromptCallback = nil
		}
	}
	if i.YNCallback != nil && hadYN {
		i.YNCallback(i.YNResp, canceled)
	}
}

// Reset resets the infobuffer's msg and info
func (i *InfoBuf) Reset() {
	i.clearPopup()
	i.PromptInputRows = 1
	i.PromptBufferMode = false
	i.PromptUseHistory = false
}

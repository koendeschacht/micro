package buffer

import (
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/util"
	"github.com/micro-editor/tcell/v2"
)

type MsgType int

const (
	MTInfo = iota
	MTWarning
	MTError
)

// Message represents the information for a gutter message
type Message struct {
	// The Msg iteslf
	Msg string
	// Start and End locations for the message
	Start, End Loc
	// The Kind stores the message type
	Kind MsgType
	// The Owner of the message
	Owner string
}

// NewMessage creates a new gutter message
func NewMessage(owner string, msg string, start, end Loc, kind MsgType) *Message {
	return &Message{
		Msg:   msg,
		Start: start,
		End:   end,
		Kind:  kind,
		Owner: owner,
	}
}

// NewMessageAtLine creates a new gutter message at a given line
func NewMessageAtLine(owner string, msg string, line int, kind MsgType) *Message {
	start := Loc{-1, line - 1}
	end := start
	return NewMessage(owner, msg, start, end, kind)
}

func (m *Message) Style() tcell.Style {
	switch m.Kind {
	case MTInfo:
		if style, ok := config.Colorscheme["gutter-info"]; ok {
			return style
		}
	case MTWarning:
		if style, ok := config.Colorscheme["gutter-warning"]; ok {
			return style
		}
	case MTError:
		if style, ok := config.Colorscheme["gutter-error"]; ok {
			return style
		}
	}
	return config.DefStyle
}

func (b *Buffer) AddMessage(m *Message) {
	b.Messages = append(b.Messages, m)
}

func (b *Buffer) removeMsg(i int) {
	copy(b.Messages[i:], b.Messages[i+1:])
	b.Messages[len(b.Messages)-1] = nil
	b.Messages = b.Messages[:len(b.Messages)-1]
}

func (b *Buffer) ClearMessages(owner string) {
	for i := len(b.Messages) - 1; i >= 0; i-- {
		if b.Messages[i].Owner == owner {
			b.removeMsg(i)
		}
	}
}

func (b *Buffer) ClearAllMessages() {
	b.Messages = make([]*Message, 0)
}

type Messager interface {
	Message(msg ...any)
}

var prompt Messager

func SetMessager(m Messager) {
	prompt = m
}

func IsDiagnosticMessage(m *Message) bool {
	return m != nil && m.Msg != ""
}

func DiagnosticSortKey(m *Message) Loc {
	if m == nil {
		return Loc{}
	}
	loc := m.Start
	if loc.X < 0 {
		loc.X = 0
	}
	if loc.Y < 0 {
		loc.Y = 0
	}
	return loc
}

func messageContainsCursor(m *Message, loc Loc) bool {
	start, end := m.Start, m.End
	if end.LessThan(start) {
		start, end = end, start
	}
	if start == end {
		return loc.Y == start.Y
	}
	return !loc.LessThan(start) && loc.LessThan(end)
}

func messageContainsLine(m *Message, line int) bool {
	startY := util.Min(m.Start.Y, m.End.Y)
	endY := util.Max(m.Start.Y, m.End.Y)
	return line >= startY && line <= endY
}

func messagePriority(m *Message) int {
	switch m.Kind {
	case MTError:
		return 0
	case MTWarning:
		return 1
	default:
		return 2
	}
}

func messageSpan(m *Message, b *Buffer) int {
	start, end := m.Start, m.End
	if start.X < 0 {
		start.X = 0
	}
	if end.X < 0 {
		end.X = 0
	}
	if start.Y < 0 {
		start.Y = 0
	}
	if end.Y < 0 {
		end.Y = 0
	}
	return start.Diff(end, b)
}

func CurrentDiagnosticMessage(msgs []*Message, cursor Loc, b *Buffer) *Message {
	var best *Message
	bestMatchRank := 3
	bestPriority := 3
	bestSpan := 0

	for _, m := range msgs {
		if !IsDiagnosticMessage(m) {
			continue
		}

		matchRank := 3
		switch {
		case messageContainsCursor(m, cursor):
			matchRank = 0
		case m.Start.Y == cursor.Y:
			matchRank = 1
		case messageContainsLine(m, cursor.Y):
			matchRank = 2
		default:
			continue
		}

		priority := messagePriority(m)
		span := messageSpan(m, b)
		if best == nil || matchRank < bestMatchRank ||
			(matchRank == bestMatchRank && priority < bestPriority) ||
			(matchRank == bestMatchRank && priority == bestPriority && span < bestSpan) {
			best = m
			bestMatchRank = matchRank
			bestPriority = priority
			bestSpan = span
		}
	}

	return best
}

func NextDiagnosticMessage(msgs []*Message, cursor Loc, forward bool) *Message {
	var target *Message
	var wrapTarget *Message

	for _, msg := range msgs {
		if !IsDiagnosticMessage(msg) {
			continue
		}

		msgLoc := DiagnosticSortKey(msg)
		if forward {
			if msgLoc.GreaterThan(cursor) && (target == nil || msgLoc.LessThan(DiagnosticSortKey(target))) {
				target = msg
			}
			if wrapTarget == nil || msgLoc.LessThan(DiagnosticSortKey(wrapTarget)) {
				wrapTarget = msg
			}
		} else {
			if msgLoc.LessThan(cursor) && (target == nil || msgLoc.GreaterThan(DiagnosticSortKey(target))) {
				target = msg
			}
			if wrapTarget == nil || msgLoc.GreaterThan(DiagnosticSortKey(wrapTarget)) {
				wrapTarget = msg
			}
		}
	}

	if target != nil {
		return target
	}
	return wrapTarget
}

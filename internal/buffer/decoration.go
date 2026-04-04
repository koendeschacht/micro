package buffer

import (
	"encoding/json"
	"log"
	"sort"
	"unicode/utf8"

	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/util"
)

// Decoration stores a resolved decoration style for a line or span.
type Decoration struct {
	Group    string
	Priority int
}

// DecorationLineMatch stores decoration group transitions for one line.
type DecorationLineMatch map[int]Decoration

// GutterDecoration stores a resolved gutter marker for one line.
type GutterDecoration struct {
	Group    string
	Symbol   rune
	Priority int
}

type DecorationOwnerState struct {
	Version int
	Spans   []DecorationSpan
	Lines   []LineDecoration
	Gutters []GutterDecorationSpan
}

type DecorationSpan struct {
	Start    Loc
	End      Loc
	Group    string
	Priority int
	owner    string
	order    int
}

type LineDecoration struct {
	Line     int
	Group    string
	Priority int
	owner    string
	order    int
}

type GutterDecorationSpan struct {
	Line     int
	Group    string
	Symbol   rune
	Priority int
	owner    string
	order    int
}

type externalDecorationSpan struct {
	Kind      string `json:"kind"`
	StartLine int    `json:"startLine"`
	Start     int    `json:"start"`
	EndLine   int    `json:"endLine"`
	End       int    `json:"end"`
	Line      int    `json:"line"`
	Group     string `json:"group"`
	Priority  int    `json:"priority"`
	Symbol    string `json:"symbol"`
}

type decorationEvent struct {
	pos   int
	start bool
	span  DecorationSpan
}

type resolvedDecoration struct {
	Decoration
	owner string
	order int
}

type resolvedGutterDecoration struct {
	GutterDecoration
	owner string
	order int
}

func (b *SharedBuffer) decorationOwnerState(owner string) DecorationOwnerState {
	if b.decorationOwners == nil {
		return DecorationOwnerState{}
	}
	return b.decorationOwners[owner]
}

func decorationWins(newPriority int, newOwner string, newOrder int, curPriority int, curOwner string, curOrder int, hasCurrent bool) bool {
	if !hasCurrent {
		return true
	}
	if newPriority != curPriority {
		return newPriority > curPriority
	}
	if newOwner != curOwner {
		return newOwner < curOwner
	}
	return newOrder < curOrder
}

func bestActiveDecoration(active []DecorationSpan) (DecorationSpan, bool) {
	var best DecorationSpan
	hasBest := false
	for _, span := range active {
		if decorationWins(span.Priority, span.owner, span.order, best.Priority, best.owner, best.order, hasBest) {
			best = span
			hasBest = true
		}
	}
	return best, hasBest
}

func addDecorationTransition(line DecorationLineMatch, pos int, group string, priority int) {
	if line == nil {
		return
	}
	if cur, ok := line[pos]; ok && cur.Group == group && cur.Priority == priority {
		return
	}
	line[pos] = Decoration{Group: group, Priority: priority}
}

func (b *SharedBuffer) rebuildDecorations() {
	if len(b.decorationOwners) == 0 {
		b.DecorationMatch = nil
		b.LineDecorations = nil
		b.GutterDecorations = nil
		return
	}

	spanEvents := make(map[int][]decorationEvent)
	lineDecorations := make(map[int]resolvedDecoration)
	gutterDecorations := make(map[int]resolvedGutterDecoration)

	for owner, state := range b.decorationOwners {
		for _, span := range state.Spans {
			for line := span.Start.Y; line <= span.End.Y; line++ {
				if line < 0 || line >= b.LinesNum() {
					continue
				}

				startX := 0
				if line == span.Start.Y {
					startX = span.Start.X
				}

				lineLen := util.CharacterCount(b.LineBytes(line))
				endX := lineLen
				if line == span.End.Y {
					endX = span.End.X
				}
				if endX <= startX {
					continue
				}

				eventSpan := span
				eventSpan.owner = owner
				spanEvents[line] = append(spanEvents[line],
					decorationEvent{pos: startX, start: true, span: eventSpan},
					decorationEvent{pos: endX, start: false, span: eventSpan},
				)
			}
		}

		for _, line := range state.Lines {
			if line.Line < 0 || line.Line >= b.LinesNum() {
				continue
			}
			cur, ok := lineDecorations[line.Line]
			if decorationWins(line.Priority, line.owner, line.order, cur.Priority, cur.owner, cur.order, ok) {
				lineDecorations[line.Line] = resolvedDecoration{
					Decoration: Decoration{Group: line.Group, Priority: line.Priority},
					owner:      line.owner,
					order:      line.order,
				}
			}
		}

		for _, gutter := range state.Gutters {
			if gutter.Line < 0 || gutter.Line >= b.LinesNum() {
				continue
			}
			cur, ok := gutterDecorations[gutter.Line]
			if decorationWins(gutter.Priority, gutter.owner, gutter.order, cur.Priority, cur.owner, cur.order, ok) {
				gutterDecorations[gutter.Line] = resolvedGutterDecoration{
					GutterDecoration: GutterDecoration{Group: gutter.Group, Symbol: gutter.Symbol, Priority: gutter.Priority},
					owner:            gutter.owner,
					order:            gutter.order,
				}
			}
		}
	}

	decorationMatch := make(map[int]DecorationLineMatch)
	for line, events := range spanEvents {
		sort.Slice(events, func(i, j int) bool {
			return events[i].pos < events[j].pos
		})

		active := make([]DecorationSpan, 0, len(events))
		transitions := make(DecorationLineMatch)
		for i := 0; i < len(events); {
			pos := events[i].pos
			samePos := make([]decorationEvent, 0, 4)
			samePos = append(samePos, events[i])
			i++
			for i < len(events) && events[i].pos == pos {
				samePos = append(samePos, events[i])
				i++
			}

			for _, event := range samePos {
				if event.start {
					continue
				}
				for idx := len(active) - 1; idx >= 0; idx-- {
					candidate := active[idx]
					if candidate.Start == event.span.Start && candidate.End == event.span.End &&
						candidate.Group == event.span.Group && candidate.Priority == event.span.Priority &&
						candidate.owner == event.span.owner && candidate.order == event.span.order {
						active = append(active[:idx], active[idx+1:]...)
						break
					}
				}
			}

			for _, event := range samePos {
				if !event.start {
					continue
				}
				active = append(active, event.span)
			}

			if best, ok := bestActiveDecoration(active); ok {
				addDecorationTransition(transitions, pos, best.Group, best.Priority)
			} else {
				addDecorationTransition(transitions, pos, "", 0)
			}
		}

		if len(transitions) > 0 {
			decorationMatch[line] = transitions
		}
	}

	if len(decorationMatch) == 0 {
		b.DecorationMatch = nil
	} else {
		b.DecorationMatch = decorationMatch
	}
	if len(lineDecorations) == 0 {
		b.LineDecorations = nil
	} else {
		b.LineDecorations = make(map[int]Decoration, len(lineDecorations))
		for line, decoration := range lineDecorations {
			b.LineDecorations[line] = decoration.Decoration
		}
	}
	if len(gutterDecorations) == 0 {
		b.GutterDecorations = nil
	} else {
		b.GutterDecorations = make(map[int]GutterDecoration, len(gutterDecorations))
		for line, decoration := range gutterDecorations {
			b.GutterDecorations[line] = decoration.GutterDecoration
		}
	}
}

func (b *SharedBuffer) setDecorationOwnerState(owner string, state DecorationOwnerState) {
	if b.decorationOwners == nil {
		b.decorationOwners = make(map[string]DecorationOwnerState)
	}
	b.decorationOwners[owner] = state
	b.rebuildDecorations()
}

func (b *SharedBuffer) clearDecorations(owner string) {
	if b.decorationOwners == nil {
		return
	}
	delete(b.decorationOwners, owner)
	b.rebuildDecorations()
}

func normalizeDecorationSpan(raw externalDecorationSpan, owner string, order int, buf *Buffer) (DecorationSpan, bool) {
	if raw.Group == "" || raw.StartLine < 0 || raw.EndLine < raw.StartLine || raw.Start < 0 || raw.End < 0 {
		return DecorationSpan{}, false
	}
	if raw.StartLine >= buf.LinesNum() {
		return DecorationSpan{}, false
	}
	endLine := util.Clamp(raw.EndLine, 0, buf.LinesNum()-1)
	startLine := raw.StartLine
	startX := raw.Start
	endX := raw.End
	if startLine == endLine && endX <= startX {
		return DecorationSpan{}, false
	}

	startLen := util.CharacterCount(buf.LineBytes(startLine))
	if startX > startLen {
		return DecorationSpan{}, false
	}
	endLen := util.CharacterCount(buf.LineBytes(endLine))
	if endX > endLen {
		endX = endLen
	}

	return DecorationSpan{
		Start:    Loc{X: startX, Y: startLine},
		End:      Loc{X: endX, Y: endLine},
		Group:    raw.Group,
		Priority: raw.Priority,
		owner:    owner,
		order:    order,
	}, true
}

func normalizeLineDecoration(raw externalDecorationSpan, owner string, order int, buf *Buffer) (LineDecoration, bool) {
	if raw.Group == "" || raw.Line < 0 || raw.Line >= buf.LinesNum() {
		return LineDecoration{}, false
	}
	return LineDecoration{Line: raw.Line, Group: raw.Group, Priority: raw.Priority, owner: owner, order: order}, true
}

func normalizeGutterDecoration(raw externalDecorationSpan, owner string, order int, buf *Buffer) (GutterDecorationSpan, bool) {
	if raw.Group == "" || raw.Line < 0 || raw.Line >= buf.LinesNum() || raw.Symbol == "" {
		return GutterDecorationSpan{}, false
	}
	symbol, _ := utf8.DecodeRuneInString(raw.Symbol)
	if symbol == utf8.RuneError {
		return GutterDecorationSpan{}, false
	}
	return GutterDecorationSpan{Line: raw.Line, Group: raw.Group, Symbol: symbol, Priority: raw.Priority, owner: owner, order: order}, true
}

// ClearDecorations removes all decorations owned by the given owner.
func (b *Buffer) ClearDecorations(owner string) {
	b.clearDecorations(owner)
	screen.Redraw()
}

// DecorationLine returns resolved span decoration transitions for one line.
func (b *Buffer) DecorationLine(line int) DecorationLineMatch {
	if b.DecorationMatch == nil {
		return nil
	}
	return b.DecorationMatch[line]
}

// LineDecoration returns the highest-priority full-line decoration for one line.
func (b *Buffer) LineDecoration(line int) (Decoration, bool) {
	if b.LineDecorations == nil {
		return Decoration{}, false
	}
	decoration, ok := b.LineDecorations[line]
	return decoration, ok
}

// GutterDecoration returns the highest-priority gutter decoration for one line.
func (b *Buffer) GutterDecoration(line int) (GutterDecoration, bool) {
	if b.GutterDecorations == nil {
		return GutterDecoration{}, false
	}
	decoration, ok := b.GutterDecorations[line]
	return decoration, ok
}

// HasGutterDecorations reports whether the buffer has any gutter decorations.
func (b *Buffer) HasGutterDecorations() bool {
	return len(b.GutterDecorations) > 0
}

// SetDecorationsJSON parses decorations and applies them for one owner.
func (b *Buffer) SetDecorationsJSON(owner, serialized string, version int) bool {
	if owner == "" {
		return false
	}
	current := b.decorationOwnerState(owner)
	if version < current.Version {
		return false
	}

	state := DecorationOwnerState{Version: version}
	if serialized != "" && serialized != "[]" {
		var raw []externalDecorationSpan
		if err := json.Unmarshal([]byte(serialized), &raw); err != nil {
			log.Printf("decoration: unmarshal failed path=%q owner=%q version=%d bytes=%d err=%v", b.AbsPath, owner, version, len(serialized), err)
			return false
		}

		state.Spans = make([]DecorationSpan, 0, len(raw))
		state.Lines = make([]LineDecoration, 0, len(raw))
		state.Gutters = make([]GutterDecorationSpan, 0, len(raw))
		for i, item := range raw {
			switch item.Kind {
			case "span":
				if span, ok := normalizeDecorationSpan(item, owner, i, b); ok {
					state.Spans = append(state.Spans, span)
				}
			case "line":
				if line, ok := normalizeLineDecoration(item, owner, i, b); ok {
					state.Lines = append(state.Lines, line)
				}
			case "gutter":
				if gutter, ok := normalizeGutterDecoration(item, owner, i, b); ok {
					state.Gutters = append(state.Gutters, gutter)
				}
			}
		}
	}

	b.setDecorationOwnerState(owner, state)
	screen.Redraw()
	return true
}

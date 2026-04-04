package buffer

import (
	"encoding/json"
	"log"
	"sort"
	"strconv"

	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/util"
)

// VirtualLine is a non-editable display row anchored relative to a real buffer line.
type VirtualLine struct {
	ID    string
	Text  string
	Group string
}

type VirtualLineOwnerState struct {
	Version int
	Entries []VirtualLineEntry
}

type VirtualLineEntry struct {
	ID    string
	Line  int
	Above bool
	Text  string
	Group string
	owner string
	order int
}

type VirtualLineDecorationOwnerState struct {
	Version int
	Entries []VirtualLineDecorationEntry
}

type VirtualLineDecorationEntry struct {
	ID       string
	Start    int
	End      int
	Group    string
	Priority int
	owner    string
	order    int
}

type externalVirtualLine struct {
	ID    string `json:"id"`
	Line  int    `json:"line"`
	Above bool   `json:"above"`
	Text  string `json:"text"`
	Group string `json:"group"`
}

type externalVirtualLineDecoration struct {
	ID       string `json:"id"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Group    string `json:"group"`
	Priority int    `json:"priority"`
}

type virtualDecorationEvent struct {
	pos   int
	start bool
	entry VirtualLineDecorationEntry
}

func normalizeVirtualLine(raw externalVirtualLine, owner string, order int, buf *Buffer) (VirtualLineEntry, bool) {
	if raw.Text == "" || raw.Line < 0 || raw.Line >= buf.LinesNum() {
		return VirtualLineEntry{}, false
	}
	id := raw.ID
	if id == "" {
		id = owner + ":" + strconv.Itoa(order)
	}
	return VirtualLineEntry{
		ID:    id,
		Line:  raw.Line,
		Above: raw.Above,
		Text:  raw.Text,
		Group: raw.Group,
		owner: owner,
		order: order,
	}, true
}

func normalizeVirtualLineDecoration(raw externalVirtualLineDecoration, owner string, order int) (VirtualLineDecorationEntry, bool) {
	if raw.ID == "" || raw.Group == "" || raw.Start < 0 || raw.End <= raw.Start {
		return VirtualLineDecorationEntry{}, false
	}
	return VirtualLineDecorationEntry{
		ID:       raw.ID,
		Start:    raw.Start,
		End:      raw.End,
		Group:    raw.Group,
		Priority: raw.Priority,
		owner:    owner,
		order:    order,
	}, true
}

func (b *SharedBuffer) virtualLineOwnerState(owner string) VirtualLineOwnerState {
	if b.virtualLineOwners == nil {
		return VirtualLineOwnerState{}
	}
	return b.virtualLineOwners[owner]
}

func (b *SharedBuffer) virtualLineDecorationOwnerState(owner string) VirtualLineDecorationOwnerState {
	if b.virtualLineDecorationOwners == nil {
		return VirtualLineDecorationOwnerState{}
	}
	return b.virtualLineDecorationOwners[owner]
}

func (b *SharedBuffer) rebuildVirtualLines() {
	if len(b.virtualLineOwners) == 0 {
		b.VirtualLinesAbove = nil
		b.VirtualLinesBelow = nil
		b.rebuildVirtualLineDecorations()
		return
	}

	entries := make([]VirtualLineEntry, 0)
	for _, state := range b.virtualLineOwners {
		entries = append(entries, state.Entries...)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Line != entries[j].Line {
			return entries[i].Line < entries[j].Line
		}
		if entries[i].Above != entries[j].Above {
			return entries[i].Above && !entries[j].Above
		}
		if entries[i].owner != entries[j].owner {
			return entries[i].owner < entries[j].owner
		}
		return entries[i].order < entries[j].order
	})

	above := make(map[int][]VirtualLine)
	below := make(map[int][]VirtualLine)
	for _, entry := range entries {
		line := VirtualLine{ID: entry.ID, Text: entry.Text, Group: entry.Group}
		if entry.Above {
			above[entry.Line] = append(above[entry.Line], line)
		} else {
			below[entry.Line] = append(below[entry.Line], line)
		}
	}

	if len(above) == 0 {
		b.VirtualLinesAbove = nil
	} else {
		b.VirtualLinesAbove = above
	}
	if len(below) == 0 {
		b.VirtualLinesBelow = nil
	} else {
		b.VirtualLinesBelow = below
	}
	b.rebuildVirtualLineDecorations()
}

func (b *SharedBuffer) rebuildVirtualLineDecorations() {
	if len(b.virtualLineDecorationOwners) == 0 || (len(b.VirtualLinesAbove) == 0 && len(b.VirtualLinesBelow) == 0) {
		b.VirtualLineDecorations = nil
		return
	}

	idToLine := make(map[string]VirtualLine)
	for _, lines := range b.VirtualLinesAbove {
		for _, line := range lines {
			idToLine[line.ID] = line
		}
	}
	for _, lines := range b.VirtualLinesBelow {
		for _, line := range lines {
			idToLine[line.ID] = line
		}
	}

	byID := make(map[string][]VirtualLineDecorationEntry)
	for _, state := range b.virtualLineDecorationOwners {
		for _, entry := range state.Entries {
			line, ok := idToLine[entry.ID]
			if !ok {
				continue
			}
			lineLen := util.CharacterCountInString(line.Text)
			if entry.Start >= lineLen {
				continue
			}
			if entry.End > lineLen {
				entry.End = lineLen
			}
			if entry.End <= entry.Start {
				continue
			}
			byID[entry.ID] = append(byID[entry.ID], entry)
		}
	}

	if len(byID) == 0 {
		b.VirtualLineDecorations = nil
		return
	}

	result := make(map[string]DecorationLineMatch, len(byID))
	for id, entries := range byID {
		matches := buildVirtualLineDecorationMatch(entries)
		if len(matches) > 0 {
			result[id] = matches
		}
	}

	if len(result) == 0 {
		b.VirtualLineDecorations = nil
	} else {
		b.VirtualLineDecorations = result
	}
}

func buildVirtualLineDecorationMatch(entries []VirtualLineDecorationEntry) DecorationLineMatch {
	events := make([]virtualDecorationEvent, 0, len(entries)*2)
	for _, entry := range entries {
		events = append(events,
			virtualDecorationEvent{pos: entry.Start, start: true, entry: entry},
			virtualDecorationEvent{pos: entry.End, start: false, entry: entry},
		)
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].pos < events[j].pos
	})

	active := make([]VirtualLineDecorationEntry, 0, len(entries))
	transitions := make(DecorationLineMatch)
	for i := 0; i < len(events); {
		pos := events[i].pos
		samePos := make([]virtualDecorationEvent, 0, 4)
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
				if candidate == event.entry {
					active = append(active[:idx], active[idx+1:]...)
					break
				}
			}
		}

		for _, event := range samePos {
			if !event.start {
				continue
			}
			active = append(active, event.entry)
		}

		if best, ok := bestVirtualLineDecoration(active); ok {
			addDecorationTransition(transitions, pos, best.Group, best.Priority)
		} else {
			addDecorationTransition(transitions, pos, "", 0)
		}
	}

	return transitions
}

func bestVirtualLineDecoration(active []VirtualLineDecorationEntry) (VirtualLineDecorationEntry, bool) {
	var best VirtualLineDecorationEntry
	hasBest := false
	for _, entry := range active {
		if decorationWins(entry.Priority, entry.owner, entry.order, best.Priority, best.owner, best.order, hasBest) {
			best = entry
			hasBest = true
		}
	}
	return best, hasBest
}

func (b *SharedBuffer) setVirtualLineOwnerState(owner string, state VirtualLineOwnerState) {
	if b.virtualLineOwners == nil {
		b.virtualLineOwners = make(map[string]VirtualLineOwnerState)
	}
	b.virtualLineOwners[owner] = state
	b.rebuildVirtualLines()
}

func (b *SharedBuffer) setVirtualLineDecorationOwnerState(owner string, state VirtualLineDecorationOwnerState) {
	if b.virtualLineDecorationOwners == nil {
		b.virtualLineDecorationOwners = make(map[string]VirtualLineDecorationOwnerState)
	}
	b.virtualLineDecorationOwners[owner] = state
	b.rebuildVirtualLineDecorations()
}

func (b *SharedBuffer) clearVirtualLines(owner string) {
	if b.virtualLineOwners == nil {
		return
	}
	delete(b.virtualLineOwners, owner)
	b.rebuildVirtualLines()
}

func (b *SharedBuffer) clearVirtualLineDecorations(owner string) {
	if b.virtualLineDecorationOwners == nil {
		return
	}
	delete(b.virtualLineDecorationOwners, owner)
	b.rebuildVirtualLineDecorations()
}

// VirtualLines returns the virtual lines anchored above or below a real line.
func (b *Buffer) VirtualLines(line int, above bool) []VirtualLine {
	if above {
		if b.VirtualLinesAbove == nil {
			return nil
		}
		return b.VirtualLinesAbove[line]
	}
	if b.VirtualLinesBelow == nil {
		return nil
	}
	return b.VirtualLinesBelow[line]
}

// VirtualLineDecoration returns decoration transitions for a virtual line by id.
func (b *Buffer) VirtualLineDecoration(id string) DecorationLineMatch {
	if b.VirtualLineDecorations == nil {
		return nil
	}
	return b.VirtualLineDecorations[id]
}

// VirtualLineCount returns the number of virtual lines anchored above or below a real line.
func (b *Buffer) VirtualLineCount(line int, above bool) int {
	return len(b.VirtualLines(line, above))
}

// HasVirtualLines reports whether the buffer has any virtual lines.
func (b *Buffer) HasVirtualLines() bool {
	return len(b.VirtualLinesAbove) > 0 || len(b.VirtualLinesBelow) > 0
}

// ClearVirtualLines removes all virtual lines for one owner.
func (b *Buffer) ClearVirtualLines(owner string) {
	b.clearVirtualLines(owner)
	screen.Redraw()
}

// ClearVirtualLineDecorations removes all virtual line decorations for one owner.
func (b *Buffer) ClearVirtualLineDecorations(owner string) {
	b.clearVirtualLineDecorations(owner)
	screen.Redraw()
}

// SetVirtualLinesJSON parses and applies virtual lines for one owner.
func (b *Buffer) SetVirtualLinesJSON(owner, serialized string, version int) bool {
	if owner == "" {
		return false
	}
	current := b.virtualLineOwnerState(owner)
	if version < current.Version {
		return false
	}

	state := VirtualLineOwnerState{Version: version}
	if serialized != "" && serialized != "[]" {
		var raw []externalVirtualLine
		if err := json.Unmarshal([]byte(serialized), &raw); err != nil {
			log.Printf("virtualline: unmarshal failed path=%q owner=%q version=%d bytes=%d err=%v", b.AbsPath, owner, version, len(serialized), err)
			return false
		}

		state.Entries = make([]VirtualLineEntry, 0, len(raw))
		for i, item := range raw {
			if entry, ok := normalizeVirtualLine(item, owner, i, b); ok {
				state.Entries = append(state.Entries, entry)
			}
		}
	}

	b.setVirtualLineOwnerState(owner, state)
	screen.Redraw()
	return true
}

// SetVirtualLineDecorationsJSON parses and applies virtual line span decorations for one owner.
func (b *Buffer) SetVirtualLineDecorationsJSON(owner, serialized string, version int) bool {
	if owner == "" {
		return false
	}
	current := b.virtualLineDecorationOwnerState(owner)
	if version < current.Version {
		return false
	}

	state := VirtualLineDecorationOwnerState{Version: version}
	if serialized != "" && serialized != "[]" {
		var raw []externalVirtualLineDecoration
		if err := json.Unmarshal([]byte(serialized), &raw); err != nil {
			log.Printf("virtuallinedecoration: unmarshal failed path=%q owner=%q version=%d bytes=%d err=%v", b.AbsPath, owner, version, len(serialized), err)
			return false
		}

		state.Entries = make([]VirtualLineDecorationEntry, 0, len(raw))
		for i, item := range raw {
			if entry, ok := normalizeVirtualLineDecoration(item, owner, i); ok {
				state.Entries = append(state.Entries, entry)
			}
		}
	}

	b.setVirtualLineDecorationOwnerState(owner, state)
	screen.Redraw()
	return true
}

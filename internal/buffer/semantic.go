package buffer

import (
	"encoding/json"
	"log"

	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/util"
)

// SemanticLineMatch stores semantic highlight group transitions for one line.
type SemanticLineMatch map[int]string

type externalSemanticSpan struct {
	Line   int    `json:"line"`
	Start  int    `json:"start"`
	Length int    `json:"length"`
	Group  string `json:"group"`
}

func (b *SharedBuffer) clearSemanticHighlights() {
	b.SemanticMatch = nil
	b.SemanticVersion = 0
}

// ClearSemanticHighlights removes all external semantic highlight state.
func (b *Buffer) ClearSemanticHighlights() {
	b.clearSemanticHighlights()
	screen.Redraw()
}

// SemanticLine returns semantic highlight transitions for a line.
func (b *Buffer) SemanticLine(line int) SemanticLineMatch {
	if b.SemanticMatch == nil {
		return nil
	}
	return b.SemanticMatch[line]
}

// SetSemanticHighlightsJSON parses semantic highlight spans and applies them.
func (b *Buffer) SetSemanticHighlightsJSON(serialized string, version int) bool {
	if version < b.SemanticVersion {
		return false
	}

	if serialized == "" || serialized == "[]" {
		b.clearSemanticHighlights()
		b.SemanticVersion = version
		screen.Redraw()
		return true
	}

	var spans []externalSemanticSpan
	if err := json.Unmarshal([]byte(serialized), &spans); err != nil {
		log.Printf("semantic: unmarshal failed path=%q version=%d bytes=%d err=%v", b.AbsPath, version, len(serialized), err)
		return false
	}

	matches := make(map[int]SemanticLineMatch)
	for _, span := range spans {
		if span.Length <= 0 || span.Line < 0 || span.Line >= b.LinesNum() || span.Start < 0 || span.Group == "" {
			continue
		}

		lineLen := util.CharacterCount(b.LineBytes(span.Line))
		if span.Start >= lineLen {
			continue
		}

		end := span.Start + span.Length
		if end > lineLen {
			end = lineLen
		}

		line := matches[span.Line]
		if line == nil {
			line = make(SemanticLineMatch)
			matches[span.Line] = line
		}

		line[span.Start] = span.Group
		if _, ok := line[end]; !ok {
			line[end] = ""
		}
	}

	if len(matches) == 0 {
		b.SemanticMatch = nil
	} else {
		b.SemanticMatch = matches
	}
	b.SemanticVersion = version
	screen.Redraw()
	return true
}

package display

import (
	"encoding/json"
	"log"

	"github.com/micro-editor/micro/v2/internal/screen"
)

type SidePaneLine struct {
	Text  string
	Group string
}

type SidePaneState struct {
	Version int
	Width   int
	Entries map[int]SidePaneLine
}

type externalSidePane struct {
	Width   int                    `json:"width"`
	Entries []externalSidePaneLine `json:"entries"`
}

type externalSidePaneLine struct {
	Line  int    `json:"line"`
	Text  string `json:"text"`
	Group string `json:"group"`
}

func normalizeSidePaneState(raw externalSidePane, version int, linesNum int) *SidePaneState {
	if raw.Width <= 0 {
		return nil
	}

	entries := make(map[int]SidePaneLine, len(raw.Entries))
	for _, entry := range raw.Entries {
		if entry.Line < 0 || entry.Line >= linesNum {
			continue
		}
		entries[entry.Line] = SidePaneLine{Text: entry.Text, Group: entry.Group}
	}

	return &SidePaneState{
		Version: version,
		Width:   raw.Width,
		Entries: entries,
	}
}

func (w *BufWindow) SetSidePaneJSON(serialized string, version int) bool {
	if version < w.sidePaneVersion {
		return false
	}

	w.sidePaneVersion = version
	if serialized == "" || serialized == "{}" {
		w.sidePane = nil
		w.updateDisplayInfo()
		w.Relocate()
		screen.Redraw()
		return true
	}

	var raw externalSidePane
	if err := json.Unmarshal([]byte(serialized), &raw); err != nil {
		log.Printf("sidepane: unmarshal failed path=%q version=%d bytes=%d err=%v", w.Buf.AbsPath, version, len(serialized), err)
		return false
	}

	w.sidePane = normalizeSidePaneState(raw, version, w.Buf.LinesNum())
	w.updateDisplayInfo()
	w.Relocate()
	screen.Redraw()
	return true
}

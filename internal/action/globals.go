package action

import (
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/info"
)

// InfoBar is the global info bar.
var InfoBar *InfoPane

// LogBufPane is a global log buffer.
var LogBufPane *BufPane

// InitGlobals initializes the log buffer and the info bar
func InitGlobals() {
	InfoBar = NewInfoBar()
	config.KeyMenuLineCount = func() int {
		if InfoBar == nil {
			return 0
		}
		return InfoBar.KeyMenuLineCount()
	}
	info.RootKeyMenuEntries = func() []info.KeyMenuEntry {
		return keyMenuEntriesForPrefix(nil)
	}
	buffer.LogBuf = buffer.NewBufferFromString("", "", buffer.BTLog)
	buffer.LogBuf.SetName("Log")
}

// GetInfoBar returns the infobar pane
func GetInfoBar() *InfoPane {
	return InfoBar
}

// WriteLog writes a string to the log buffer
func WriteLog(s string) {
	buffer.WriteLog(s)
	if LogBufPane != nil {
		LogBufPane.CursorEnd()
	}
}

// OpenLogBuf opens the log buffer from the current bufpane
// If the current bufpane is a log buffer nothing happens,
// otherwise the log buffer is opened in a horizontal split
func (h *BufPane) OpenLogBuf() {
	LogBufPane = h.HSplitBuf(buffer.LogBuf)
	LogBufPane.CursorEnd()
}

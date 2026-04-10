package action

import (
	"github.com/micro-editor/micro/v2/internal/buffer"
)

const maxJumpHistory = 100

// JumpLocation stores a cursor position with its associated file path.
type JumpLocation struct {
	Buf  *buffer.Buffer
	Path string
	Loc  buffer.Loc
}

// PushJump saves the current cursor position to the jump history.
// Call this before any action that moves the cursor a large distance.
// Exported so it can be called from Lua plugins via micro.PushJump().
func (h *BufPane) PushJump() {
	if h.jumpNavigating {
		return
	}
	path := h.Buf.AbsPath
	loc := h.Cursor.Loc

	// Truncate forward history when branching from a previous position
	h.jumpHistory = h.jumpHistory[:h.jumpIndex]

	// Don't record a duplicate of the last entry (same file and line)
	if h.jumpIndex > 0 {
		last := h.jumpHistory[h.jumpIndex-1]
		if last.Path == path && last.Loc.Y == loc.Y {
			return
		}
	}

	h.jumpHistory = append(h.jumpHistory, JumpLocation{Buf: h.Buf, Path: path, Loc: loc})
	if len(h.jumpHistory) > maxJumpHistory {
		h.jumpHistory = h.jumpHistory[1:]
	} else {
		h.jumpIndex = len(h.jumpHistory)
	}
}

// JumpBack navigates to the previous position in the jump history.
func (h *BufPane) JumpBack() bool {
	// Save current position first so we can jump forward back to it
	if h.jumpIndex == len(h.jumpHistory) {
		h.PushJump()
	}
	if h.jumpIndex <= 1 {
		InfoBar.Message("No previous jump")
		return false
	}
	h.jumpIndex--
	h.jumpNavigating = true
	h.gotoJumpLoc(h.jumpHistory[h.jumpIndex-1])
	h.jumpNavigating = false
	return true
}

// JumpForward navigates to the next position in the jump history.
func (h *BufPane) JumpForward() bool {
	if h.jumpIndex >= len(h.jumpHistory) {
		InfoBar.Message("No next jump")
		return false
	}
	h.jumpIndex++
	h.jumpNavigating = true
	h.gotoJumpLoc(h.jumpHistory[h.jumpIndex-1])
	h.jumpNavigating = false
	return true
}

// gotoJumpLoc opens the file if necessary and moves the cursor to the location.
func (h *BufPane) gotoJumpLoc(loc JumpLocation) {
	if loc.Buf != nil && bufferIsOpen(loc.Buf) {
		if loc.Buf != h.Buf {
			h.OpenBuffer(loc.Buf)
		}
	} else if loc.Path != h.Buf.AbsPath {
		buf, err := buffer.NewBufferFromFile(loc.Path, buffer.BTDefault)
		if err != nil {
			InfoBar.Error(err)
			return
		}
		h.OpenBuffer(buf)
	}
	h.GotoLoc(loc.Loc)
}

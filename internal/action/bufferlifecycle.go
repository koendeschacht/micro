package action

import "github.com/micro-editor/micro/v2/internal/buffer"

func bufferIsOpen(target *buffer.Buffer) bool {
	for _, b := range buffer.OpenBuffers {
		if b == target {
			return true
		}
	}
	return false
}

func visibleBuffers() map[*buffer.Buffer]struct{} {
	visible := make(map[*buffer.Buffer]struct{})
	if Tabs == nil {
		return visible
	}

	for _, tab := range Tabs.List {
		for _, pane := range tab.Panes {
			if bp, ok := pane.(*BufPane); ok {
				visible[bp.Buf] = struct{}{}
			}
		}
	}

	return visible
}

func isVisibleBuffer(target *buffer.Buffer) bool {
	_, ok := visibleBuffers()[target]
	return ok
}

func modifiedBuffers(current *buffer.Buffer) []*buffer.Buffer {
	seen := make(map[*buffer.SharedBuffer]struct{})
	buffers := make([]*buffer.Buffer, 0)

	add := func(b *buffer.Buffer) {
		if b == nil || !b.Modified() {
			return
		}
		if _, ok := seen[b.SharedBuffer]; ok {
			return
		}
		seen[b.SharedBuffer] = struct{}{}
		buffers = append(buffers, b)
	}

	add(current)
	for _, b := range buffer.OpenBuffers {
		add(b)
	}

	return buffers
}

func closeHiddenCleanBuffers() {
	visible := visibleBuffers()
	toClose := make([]*buffer.Buffer, 0)

	for _, b := range buffer.OpenBuffers {
		if _, ok := visible[b]; ok || b.Modified() {
			continue
		}
		toClose = append(toClose, b)
	}

	for _, b := range toClose {
		if bufferIsOpen(b) {
			b.Close()
		}
	}
}

// CloseHiddenCleanBuffers closes buffers that are no longer shown in any pane
// once they have been saved or autosaved.
func CloseHiddenCleanBuffers() {
	closeHiddenCleanBuffers()
}

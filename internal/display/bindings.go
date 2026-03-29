package display

import (
	"strings"

	"github.com/micro-editor/micro/v2/internal/info"
	"github.com/micro-editor/micro/v2/internal/keymenu"
)

func NewKeyMenuEntry(key, label string) info.KeyMenuEntry {
	return info.KeyMenuEntry{Key: key, Label: label}
}

func formatKeyName(name string) string {
	if len(name) == 1 {
		return strings.ToLower(name)
	}
	return name
}

func FormatToken(token keymenu.Token) string {
	if token.Mouse {
		return token.Key
	}

	key := formatKeyName(token.Key)
	prefix := ""
	if token.Meta {
		prefix += "Meta-"
	}
	if token.Alt {
		prefix += "M-"
	}
	if token.Ctrl {
		prefix += "^"
	}
	if token.Shift {
		prefix += "S-"
	}
	return prefix + key
}

func FormatBindingSequence(tokens []keymenu.Token) string {
	parts := make([]string, 0, len(tokens))
	for _, token := range tokens {
		parts = append(parts, FormatToken(token))
	}
	return strings.Join(parts, " ")
}

func bindingLabelForAction(action string) string {
	binding := keymenu.PreferredBindingForAction("buffer", action)
	if binding == nil {
		return ""
	}
	return FormatBindingSequence(binding.Sequence)
}

func keyMenuEntryWidth(entry info.KeyMenuEntry) int {
	return len(entry.Key) + 1 + len(entry.Label)
}

func layoutKeyMenuEntries(entries []info.KeyMenuEntry, width int) [][]info.KeyMenuEntry {
	if len(entries) == 0 {
		return nil
	}

	totalWidth := 0
	for i, entry := range entries {
		if i > 0 {
			totalWidth += 2
		}
		totalWidth += keyMenuEntryWidth(entry)
	}
	if totalWidth <= width {
		return [][]info.KeyMenuEntry{entries}
	}

	lines := [][]info.KeyMenuEntry{{}}
	lineWidth := 0
	for _, entry := range entries {
		entryWidth := keyMenuEntryWidth(entry)
		sepWidth := 0
		if len(lines[len(lines)-1]) > 0 {
			sepWidth = 2
		}
		if lineWidth > 0 && lineWidth+sepWidth+entryWidth > width {
			lines = append(lines, []info.KeyMenuEntry{})
			lineWidth = 0
			sepWidth = 0
		}
		lines[len(lines)-1] = append(lines[len(lines)-1], entry)
		lineWidth += sepWidth + entryWidth
	}
	return lines
}

package keymenu

import "strings"

var hiddenRootActions = map[string]struct{}{
	"CursorUp":                  {},
	"CursorDown":                {},
	"CursorLeft":                {},
	"CursorRight":               {},
	"SelectUp":                  {},
	"SelectDown":                {},
	"SelectLeft":                {},
	"SelectRight":               {},
	"WordLeft":                  {},
	"WordRight":                 {},
	"SelectWordLeft":            {},
	"SelectWordRight":           {},
	"StartOfTextToggle":         {},
	"EndOfLine":                 {},
	"SelectToStartOfTextToggle": {},
	"SelectToEndOfLine":         {},
	"CursorStart":               {},
	"CursorEnd":                 {},
	"SelectToStart":             {},
	"SelectToEnd":               {},
	"CursorPageUp":              {},
	"CursorPageDown":            {},
	"SelectPageUp":              {},
	"SelectPageDown":            {},
	"ParagraphPrevious":         {},
	"ParagraphNext":             {},
	"ToggleOverwriteMode":       {},
	"Delete":                    {},
	"Escape":                    {},
	"Backspace":                 {},
	"Autocomplete":              {},
	"CycleAutocompleteBack":     {},
	"InsertNewline":             {},
	"Undo":                      {},
	"Copy":                      {},
	"Cut":                       {},
	"Paste":                     {},
	"FindLiteral":               {},
	"DeleteWordLeft":            {},
	"lua:templates.expand":      {},
	"IndentSelection":           {},
	"InsertTab":                 {},
}

func primaryBindingAction(action string) string {
	if idx := strings.IndexAny(action, "&|,"); idx >= 0 {
		action = action[:idx]
	}
	return strings.TrimSpace(action)
}

func HideRootBinding(binding Binding) bool {
	sequence := binding.Sequence
	if len(sequence) == 0 {
		return false
	}

	for _, token := range sequence {
		if token.Mouse || strings.HasPrefix(token.Key, "Mouse") {
			return true
		}
	}

	_, ok := hiddenRootActions[primaryBindingAction(binding.Action)]
	return ok
}

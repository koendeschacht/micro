package keymenu

import (
	"strings"
)

type HiddenRootBinding struct {
	Sequence []Token
}

var trivialRootNavigationKeys = map[string]struct{}{
	"Up":       {},
	"Down":     {},
	"Left":     {},
	"Right":    {},
	"Home":     {},
	"End":      {},
	"PgUp":     {},
	"PgDn":     {},
	"PageUp":   {},
	"PageDown": {},
}

var trivialRootActions = map[string]struct{}{
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
}

func hiddenToken(key string, ctrl, alt, shift, meta bool) Token {
	return Token{
		Key:   key,
		Ctrl:  ctrl,
		Alt:   alt,
		Shift: shift,
		Meta:  meta,
	}
}

var hiddenRootBindings = []HiddenRootBinding{
	{Sequence: []Token{hiddenToken("Up", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("Down", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("Left", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("Right", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("Home", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("End", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("Insert", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("Delete", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("Esc", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("Backspace", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("Backspace", true, false, false, false)}},
	{Sequence: []Token{hiddenToken("Escape", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("Tab", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("Backtab", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("Enter", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("PgUp", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("PgDn", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("PgUp", false, false, true, false)}},
	{Sequence: []Token{hiddenToken("PgDn", false, false, true, false)}},
	{Sequence: []Token{hiddenToken("PageUp", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("PageDown", false, false, false, false)}},
	{Sequence: []Token{hiddenToken("PageUp", false, false, true, false)}},
	{Sequence: []Token{hiddenToken("PageDown", false, false, true, false)}},
	{Sequence: []Token{hiddenToken("Left", false, false, true, false)}},
	{Sequence: []Token{hiddenToken("Right", false, false, true, false)}},
	{Sequence: []Token{hiddenToken("Up", false, false, true, false)}},
	{Sequence: []Token{hiddenToken("Down", false, false, true, false)}},
	{Sequence: []Token{hiddenToken("Home", false, false, true, false)}},
	{Sequence: []Token{hiddenToken("End", false, false, true, false)}},
	{Sequence: []Token{hiddenToken("Home", true, false, false, false)}},
	{Sequence: []Token{hiddenToken("End", true, false, false, false)}},
	{Sequence: []Token{hiddenToken("Left", true, false, false, false)}},
	{Sequence: []Token{hiddenToken("Right", true, false, false, false)}},
	{Sequence: []Token{hiddenToken("Up", true, false, false, false)}},
	{Sequence: []Token{hiddenToken("Down", true, false, false, false)}},
	{Sequence: []Token{hiddenToken("PgUp", true, false, false, false)}},
	{Sequence: []Token{hiddenToken("PgDn", true, false, false, false)}},
	{Sequence: []Token{hiddenToken("PageUp", true, false, false, false)}},
	{Sequence: []Token{hiddenToken("PageDown", true, false, false, false)}},
}

func primaryBindingAction(action string) string {
	if idx := strings.IndexAny(action, "&|,"); idx >= 0 {
		action = action[:idx]
	}
	return strings.TrimSpace(action)
}

func hidesTrivialNavigationBinding(binding Binding) bool {
	if len(binding.Sequence) != 1 {
		return false
	}

	token := binding.Sequence[0]
	if _, ok := trivialRootNavigationKeys[token.Key]; !ok {
		return false
	}

	_, ok := trivialRootActions[primaryBindingAction(binding.Action)]
	return ok
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

	if hidesTrivialNavigationBinding(binding) {
		return true
	}

	for _, hidden := range hiddenRootBindings {
		if len(hidden.Sequence) != len(sequence) {
			continue
		}
		if HasPrefix(sequence, hidden.Sequence) {
			return true
		}
	}

	return false
}

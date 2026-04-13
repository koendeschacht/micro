package keymenu

import "testing"

func TestHideRootBindingHidesCtrlParagraphBindings(t *testing.T) {
	tests := []Binding{
		{
			Action:   "ParagraphPrevious",
			Sequence: []Token{{Key: "Up", Ctrl: true}},
		},
		{
			Action:   "ParagraphNext",
			Sequence: []Token{{Key: "Down", Ctrl: true}},
		},
		{
			Action:   "Cut|CutLine",
			Sequence: []Token{{Key: "x", Ctrl: true}},
		},
		{
			Action:   "Copy|CopyLine",
			Sequence: []Token{{Key: "c", Ctrl: true}},
		},
		{
			Action:   "Undo",
			Sequence: []Token{{Key: "z", Ctrl: true}},
		},
		{
			Action:   "Paste",
			Sequence: []Token{{Key: "v", Ctrl: true}},
		},
		{
			Action:   "FindLiteral",
			Sequence: []Token{{Key: "f", Ctrl: true}},
		},
		{
			Action:   "DeleteWordLeft",
			Sequence: []Token{{Key: "Backspace", Ctrl: true}},
		},
		{
			Action:   "lua:templates.expand|Autocomplete|IndentSelection|InsertTab",
			Sequence: []Token{{Key: "Tab"}},
		},
		{
			Action:   "CursorUp",
			Sequence: []Token{{Key: "p", Alt: true}},
		},
	}

	for _, binding := range tests {
		if !HideRootBinding(binding) {
			t.Fatalf("expected binding to be hidden: %+v", binding)
		}
	}
}

func TestHideRootBindingKeepsNonPrimaryHiddenActionsVisible(t *testing.T) {
	binding := Binding{
		Action:   "DiffPrevious|CursorStart",
		Sequence: []Token{{Key: "[", Alt: true}},
	}

	if HideRootBinding(binding) {
		t.Fatalf("expected binding to remain visible: %+v", binding)
	}
}

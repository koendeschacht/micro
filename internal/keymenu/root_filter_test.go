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
	}

	for _, binding := range tests {
		if !HideRootBinding(binding) {
			t.Fatalf("expected binding to be hidden: %+v", binding)
		}
	}
}

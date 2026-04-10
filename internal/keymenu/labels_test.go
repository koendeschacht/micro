package keymenu

import "testing"

func TestDescribeBindingActionUsesShortFriendlyLabels(t *testing.T) {
	tests := map[string]string{
		"Copy|CopyLine":                             "copy",
		"Escape,Deselect,ClearInfo":                 "escape then deselect then clear info",
		"Autocomplete|IndentSelection|InsertTab":    "tab",
		"command:definition|command:references":     "definition or references",
		"lua:comment.comment|command:lspcompletion": "comment or completion",
		"command-edit:goto |command:rename":         "goto line or rename",
		"SelectObjectParagraph":                     "paragraph",
		"NextSplit|FirstSplit":                      "next pane",
		"SpawnMultiCursorDown":                      "cursor down",
		"command:jnextdiag":                         "next diag",
	}

	for action, expected := range tests {
		if got := DescribeBindingAction(action); got != expected {
			t.Fatalf("DescribeBindingAction(%q) = %q, want %q", action, got, expected)
		}
	}
}

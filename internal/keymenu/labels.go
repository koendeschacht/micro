package keymenu

import (
	"strings"
	"unicode"
)

var actionLabels = map[string]string{
	"ClipboardHistory":       "clipboard",
	"CommandMode":            "command",
	"DiffNext":               "next diff",
	"DiffPrevious":           "prev diff",
	"FindLiteral":            "find text",
	"JumpBack":               "jump back",
	"JumpForward":            "jump forward",
	"ParagraphNext":          "next paragraph",
	"ParagraphPrevious":      "prev paragraph",
	"RemoveAllMultiCursors":  "clear cursors",
	"RemoveMultiCursor":      "remove cursor",
	"SkipMultiCursor":        "skip cursor",
	"SpawnMultiCursor":       "add cursor",
	"SpawnMultiCursorDown":   "cursor down",
	"SpawnMultiCursorSelect": "select cursor",
	"SpawnMultiCursorUp":     "cursor up",
	"ToggleHelp":             "help",
	"ToggleKeyMenu":          "key menu",
	"ToggleMacro":            "macro",
	"ToggleRuler":            "ruler",
}

var compoundActionLabels = map[string]string{
	"Copy|CopyLine":                          "copy",
	"Cut|CutLine":                            "cut",
	"NextSplit|FirstSplit":                   "next pane",
	"NextTab|FirstTab":                       "next tab",
	"PreviousTab|LastTab":                    "prev tab",
	"Autocomplete|IndentSelection|InsertTab": "tab",
	"CycleAutocompleteBack|OutdentSelection|OutdentLine": "backtab",
}

var commandLabels = map[string]string{
	"definition":      "definition",
	"copypath":        "path",
	"explore":         "explore",
	"format":          "format",
	"fzf":             "files",
	"fzfgrep":         "grep",
	"goto":            "goto line",
	"jnextdiag":       "next diag",
	"jprevdiag":       "prev diag",
	"kittyterm":       "terminal",
	"lspcompletion":   "completion",
	"pytestfile":      "test file",
	"pytestnode":      "test node",
	"pytestretry":     "test retry",
	"references":      "references",
	"rename":          "rename",
	"resethunk":       "reset hunk",
	"toggle softwrap": "soft wrap",
	"uvrunfile":       "run",
}

var luaLabels = map[string]string{
	"comment.comment":  "comment",
	"templates.expand": "template",
}

func describeTextObjectBinding(action string) string {
	for _, prefix := range []string{"YankObject", "DeleteObject", "SelectObject", "FindObject"} {
		if strings.HasPrefix(action, prefix) {
			switch strings.TrimPrefix(action, prefix) {
			case "Word":
				return "word"
			case "Function":
				return "function"
			case "Class":
				return "class"
			case "DoubleQuotes":
				return "double quotes"
			case "SingleQuotes":
				return "single quotes"
			case "Parens":
				return "parentheses"
			case "Brackets":
				return "brackets"
			case "Paragraph":
				return "paragraph"
			}
		}
	}
	return ""
}

func describeCommand(command string) string {
	command = strings.TrimSpace(command)
	if label, ok := commandLabels[command]; ok {
		return label
	}
	return strings.ReplaceAll(command, "-", " ")
}

func humanizeAction(action string) string {
	var out []rune
	for i, r := range action {
		if i > 0 && unicode.IsUpper(r) {
			prev := rune(action[i-1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				out = append(out, ' ')
			}
		}
		out = append(out, unicode.ToLower(r))
	}
	return string(out)
}

func describeSingleBindingAction(action string) string {
	action = strings.TrimSpace(action)
	if action == "" {
		return ""
	}

	if label := describeTextObjectBinding(action); label != "" {
		return label
	}

	if label, ok := actionLabels[action]; ok {
		return label
	}

	switch {
	case strings.HasPrefix(action, "command-edit:"):
		return describeCommand(strings.TrimSpace(strings.TrimPrefix(action, "command-edit:")))
	case strings.HasPrefix(action, "command:"):
		return describeCommand(strings.TrimSpace(strings.TrimPrefix(action, "command:")))
	case strings.HasPrefix(action, "lua:"):
		name := strings.TrimSpace(strings.TrimPrefix(action, "lua:"))
		if label, ok := luaLabels[name]; ok {
			return label
		}
		return name
	default:
		return humanizeAction(action)
	}
}

func splitBindingActions(action string) ([]string, []rune) {
	var actions []string
	var separators []rune
	start := 0
	for i, r := range action {
		if r != '&' && r != '|' && r != ',' {
			continue
		}
		actions = append(actions, strings.TrimSpace(action[start:i]))
		separators = append(separators, r)
		start = i + 1
	}
	actions = append(actions, strings.TrimSpace(action[start:]))
	return actions, separators
}

func describeBindingSeparator(separator rune) string {
	switch separator {
	case '&':
		return " and "
	case '|':
		return " or "
	case ',':
		return " then "
	default:
		return " "
	}
}

func DescribeBindingAction(action string) string {
	action = strings.TrimSpace(action)
	if label, ok := compoundActionLabels[action]; ok {
		return label
	}

	actions, separators := splitBindingActions(action)
	if len(actions) == 0 {
		return ""
	}

	label := describeSingleBindingAction(actions[0])
	if len(actions) == 1 {
		return label
	}

	var builder strings.Builder
	builder.WriteString(label)
	for i, separator := range separators {
		next := describeSingleBindingAction(actions[i+1])
		if next == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString(describeBindingSeparator(separator))
		}
		builder.WriteString(next)
	}

	return builder.String()
}

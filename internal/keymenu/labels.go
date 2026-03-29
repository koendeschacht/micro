package keymenu

import (
	"strings"
	"unicode"
)

var actionLabels = map[string]string{
	"ClipboardHistory": "copy history",
	"FindLiteral":      "find literal",
	"ToggleHelp":       "help",
	"ToggleKeyMenu":    "key menu",
	"ToggleMacro":      "record macro",
	"ToggleRuler":      "ruler",
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
	switch command {
	case "goto":
		return "go to line"
	case "format":
		return "format"
	case "rename":
		return "rename"
	default:
		return strings.ReplaceAll(command, "-", " ")
	}
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

func DescribeBindingAction(action string) string {
	if idx := strings.IndexAny(action, "&|,"); idx >= 0 {
		action = action[:idx]
	}
	action = strings.TrimSpace(action)

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
		return strings.TrimSpace(strings.TrimPrefix(action, "lua:"))
	default:
		return humanizeAction(action)
	}
}

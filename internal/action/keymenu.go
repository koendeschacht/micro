package action

import (
	"sort"
	"strings"
	"unicode"

	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/gdamore/tcell/v3"
)

type keyMenuOption struct {
	key       string
	summaries []string
}

var keyMenuActionLabels = map[string]string{
	"ClipboardHistory": "copy history",
	"FindLiteral":      "find literal",
	"ToggleHelp":       "help",
	"ToggleKeyMenu":    "key menu",
	"ToggleMacro":      "record macro",
	"ToggleRuler":      "ruler",
}

var leaderGroupLabels = map[string]string{
	"d": "delete",
	"c": "code",
	"f": "find",
	"h": "help",
	"m": "macro",
	"n": "navigation",
	"s": "settings",
	"v": "select",
	"y": "yank",
}

func refreshActiveKeyMenu() {
	if InfoBar == nil {
		return
	}

	if !config.GetGlobalOption("keymenu").(bool) {
		InfoBar.ClearKeyMenu()
		return
	}

	if Tabs == nil || len(Tabs.List) == 0 {
		InfoBar.ClearKeyMenu()
		return
	}

	if pane := MainTab().CurPane(); pane != nil {
		pane.refreshKeyMenu()
		return
	}

	InfoBar.ClearKeyMenu()
}

func (h *BufPane) refreshKeyMenu() {
	if InfoBar == nil {
		return
	}

	lines := h.keyMenuLines()
	if len(lines) == 0 {
		InfoBar.ClearKeyMenu()
		return
	}

	InfoBar.SetKeyMenu(lines)
}

func (h *BufPane) keyMenuLines() []string {
	options := h.keyMenuOptions()
	if len(options) == 0 {
		return nil
	}

	entries := make([]string, 0, len(options))
	for _, option := range options {
		entries = append(entries, option.key+": "+summarizeKeyMenu(option.summaries))
	}

	lines := []string{"", ""}
	for i, entry := range entries {
		line := i % len(lines)
		if lines[line] != "" {
			lines[line] += ", "
		}
		lines[line] += entry
	}

	return lines
}

func (h *BufPane) keyMenuOptions() []keyMenuOption {
	if !h.pendingKeySequence || h.keyCursor == nil || len(h.keyCursor.recordedEvents) == 0 {
		return nil
	}

	prefix := h.keyMenuPrefix()
	if len(prefix) == 0 {
		return nil
	}

	type optionData struct {
		key       string
		summaries map[string]struct{}
	}

	options := make(map[string]*optionData)
	for binding, action := range config.Bindings["buffer"] {
		if action == "" {
			continue
		}

		event, err := findEvent(binding)
		if err != nil {
			continue
		}

		seq, ok := event.(KeySequenceEvent)
		if !ok || len(seq.keys) <= len(prefix) || !hasEventPrefix(seq.keys, prefix) {
			continue
		}

		next := seq.keys[len(prefix)]
		name := next.Name()
		option, ok := options[name]
		if !ok {
			option = &optionData{
				key:       keyMenuLabel(next),
				summaries: make(map[string]struct{}),
			}
			options[name] = option
		}
		summary := describeBindingAction(action)
		if h.isLeaderMenuPrefix(prefix) {
			if label, ok := leaderGroupLabels[option.key]; ok {
				summary = label
			}
		}
		option.summaries[summary] = struct{}{}
	}

	result := make([]keyMenuOption, 0, len(options))
	for _, option := range options {
		summaries := make([]string, 0, len(option.summaries))
		for summary := range option.summaries {
			summaries = append(summaries, summary)
		}
		sort.Strings(summaries)
		result = append(result, keyMenuOption{key: option.key, summaries: summaries})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].key < result[j].key
	})

	return result
}

func (h *BufPane) keyMenuPrefix() []string {
	if !h.pendingKeySequence || h.keyCursor == nil || len(h.keyCursor.recordedEvents) == 0 {
		return nil
	}

	return eventNames(h.keyCursor.recordedEvents)
}

func (h *BufPane) isLeaderMenuPrefix(prefix []string) bool {
	if len(prefix) != 1 {
		return false
	}
	leader, err := findEvent(config.GetGlobalOption("leader").(string))
	if err != nil {
		return false
	}
	return prefix[0] == leader.Name()
}

func eventNames(events []Event) []string {
	names := make([]string, 0, len(events))
	for _, event := range events {
		names = append(names, event.Name())
	}
	return names
}

func hasEventPrefix(events []Event, prefix []string) bool {
	if len(events) < len(prefix) {
		return false
	}
	for i, name := range prefix {
		if events[i].Name() != name {
			return false
		}
	}
	return true
}

func keyMenuLabel(event Event) string {
	if key, ok := event.(KeyEvent); ok && key.code == tcell.KeyRune {
		return string(key.r)
	}
	return event.Name()
}

func summarizeKeyMenu(summaries []string) string {
	if len(summaries) == 0 {
		return ""
	}
	if len(summaries) == 1 {
		return summaries[0]
	}
	if len(summaries) == 2 {
		return summaries[0] + ", " + summaries[1]
	}
	return summaries[0] + ", " + summaries[1] + ", ..."
}

func describeBindingAction(action string) string {
	if idx := strings.IndexAny(action, "&|,"); idx >= 0 {
		action = action[:idx]
	}
	action = strings.TrimSpace(action)

	if label := describeTextObjectBinding(action); label != "" {
		return label
	}

	if label, ok := keyMenuActionLabels[action]; ok {
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

package action

import (
	"sort"

	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/display"
	"github.com/micro-editor/micro/v2/internal/info"
	"github.com/micro-editor/micro/v2/internal/keymenu"
)

type keyMenuOption struct {
	key       string
	summaries []string
}

var leaderGroupLabels = map[string]string{
	"d": "delete",
	"c": "code",
	"f": "find",
	"g": "git",
	"h": "help",
	"m": "macro",
	"n": "navigation",
	"r": "run",
	"s": "settings",
	"t": "tests",
	"v": "select",
	"w": "panes",
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

	entries := h.keyMenuEntries()
	if len(entries) == 0 {
		InfoBar.ClearKeyMenu()
		return
	}

	InfoBar.SetKeyMenu(entries)
}

func (h *BufPane) keyMenuEntries() []info.KeyMenuEntry {
	return keyMenuEntriesForPrefix(h.keyMenuPrefix())
}

func keyMenuOptionsForPrefix(prefix []string) []keyMenuOption {
	prefixTokens := make([]keymenu.Token, 0, len(prefix))
	for _, recorded := range prefix {
		event, err := findEvent(recorded)
		if err != nil {
			return nil
		}
		seq := keyMenuSequenceFromEvent(event)
		if len(seq) != 1 {
			return nil
		}
		prefixTokens = append(prefixTokens, seq[0])
	}

	type optionData struct {
		key       string
		summaries map[string]struct{}
	}

	leaderSeq := keyMenuSequenceFromEvent(mustEvent(config.GetGlobalOption("leader").(string)))
	var leaderToken *keymenu.Token
	if len(leaderSeq) == 1 {
		leaderToken = &leaderSeq[0]
	}
	options := make(map[string]*optionData)
	for _, binding := range keymenu.PaneBindings("buffer") {
		if binding.Action == "" {
			continue
		}
		if len(binding.Sequence) <= len(prefixTokens) || !keymenu.HasPrefix(binding.Sequence, prefixTokens) {
			continue
		}

		next := binding.Sequence[len(prefixTokens)]
		name := display.FormatToken(next)
		option, ok := options[name]
		if !ok {
			option = &optionData{
				key:       name,
				summaries: make(map[string]struct{}),
			}
			options[name] = option
		}

		summary := keymenu.DescribeBindingAction(binding.Action)
		if len(prefixTokens) == 1 && leaderToken != nil && keymenu.EqualTokens(prefixTokens[0], *leaderToken) {
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

func keyMenuEntriesForPrefix(prefix []string) []info.KeyMenuEntry {
	if len(prefix) == 0 {
		entries := []info.KeyMenuEntry{}

		leaderSeq := keyMenuSequenceFromEvent(mustEvent(config.GetGlobalOption("leader").(string)))
		leader := display.FormatBindingSequence(leaderSeq)
		if leader != "" && len(leaderSeq) == 1 {
			entries = append(entries, display.NewKeyMenuEntry(leader, "leader"))
		}
		for _, binding := range keymenu.PaneBindings("buffer") {
			if binding.Action == "" {
				continue
			}
			if keymenu.HideRootBinding(binding) {
				continue
			}
			if len(binding.Sequence) > 1 && len(leaderSeq) == 1 && keymenu.EqualTokens(binding.Sequence[0], leaderSeq[0]) {
				continue
			}
			label := keymenu.DescribeBindingAction(binding.Action)
			if label == "" {
				continue
			}
			entries = append(entries, display.NewKeyMenuEntry(display.FormatBindingSequence(binding.Sequence), label))
		}

		sort.Slice(entries, func(i, j int) bool {
			left := entries[i].Key
			right := entries[j].Key
			if left != right {
				return left < right
			}
			return entries[i].Label < entries[j].Label
		})

		return entries
	}

	options := keyMenuOptionsForPrefix(prefix)
	if len(options) == 0 {
		return nil
	}

	entries := make([]info.KeyMenuEntry, 0, len(options))
	for _, option := range options {
		entries = append(entries, display.NewKeyMenuEntry(option.key, summarizeKeyMenu(option.summaries)))
	}

	return entries
}

func mustEvent(binding string) Event {
	event, err := findEvent(binding)
	if err != nil {
		panic(err)
	}
	return event
}

func (h *BufPane) keyMenuPrefix() []string {
	if !h.pendingKeySequence || h.keyCursor == nil || len(h.keyCursor.recordedEvents) == 0 {
		return nil
	}

	return eventNames(h.keyCursor.recordedEvents)
}

func eventNames(events []Event) []string {
	names := make([]string, 0, len(events))
	for _, event := range events {
		names = append(names, event.Name())
	}
	return names
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

package buffer

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"sort"
	"strings"

	"github.com/micro-editor/micro/v2/internal/util"
)

// A Completer is a function that takes a buffer and returns info
// describing what autocompletions should be inserted at the current
// cursor location
// It returns a list of string suggestions which will be inserted at
// the current cursor location if selected as well as a list of
// suggestion names which can be displayed in an autocomplete box or
// other UI element
type Completer func(*Buffer) ([]string, []string)

// CompletionItem is a single completion candidate from a provider.
type CompletionItem struct {
	Completion string
	Suggestion string
	Insert     string
	Provider   string
	Edits      []Delta
	SortText   string
	Preselect  bool
	Deprecated bool
}

// CompletionProvider returns completion items for the current cursor location.
type CompletionProvider func(*Buffer) []CompletionItem

// MinAutomaticCompletionChars is the minimum typed prefix length before
// automatic inline completion is shown.
const MinAutomaticCompletionChars = 3

const (
	completionProviderBuffer = "buffer"
	completionProviderLSP    = "lsp"
)

var completionUsage = map[string]int{}

// ClearAutocomplete clears all active completion UI state.
func (b *SharedBuffer) ClearAutocomplete() {
	b.Suggestions = nil
	b.Completions = nil
	b.CompletionValues = nil
	b.CompletionSources = nil
	b.CompletionEdits = nil
	b.CurSuggestion = -1
	b.HasSuggestions = false
	b.CompletionMenu = false
	b.CompletionStart = Loc{}
	b.CompletionEnd = Loc{}
	b.GhostText = ""
	b.GhostAt = Loc{}
}

func completionItemsFromCompleter(b *Buffer, c Completer) []CompletionItem {
	completions, suggestions := c(b)
	if len(completions) != len(suggestions) {
		return nil
	}

	items := make([]CompletionItem, 0, len(completions))
	for i := range completions {
		items = append(items, CompletionItem{
			Completion: completions[i],
			Suggestion: suggestions[i],
			Insert:     suggestions[i],
			Provider:   completionProviderBuffer,
		})
	}
	return items
}

// BufferCompleteProvider returns buffer word completions as completion items.
func BufferCompleteProvider(b *Buffer) []CompletionItem {
	items := completionItemsFromCompleter(b, BufferComplete)
	filtered := items[:0]
	for _, item := range items {
		if item.Completion == "" {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

// HasTextRightOfCursor reports whether the active cursor has text after it on the line.
func (b *Buffer) HasTextRightOfCursor() bool {
	c := b.GetActiveCursor()
	return c.X < util.CharacterCount(b.LineBytes(c.Y))
}

// CurrentWordLength returns the number of characters in the current word prefix.
func (b *Buffer) CurrentWordLength() int {
	input, argstart := b.GetWord()
	if argstart == -1 {
		return 0
	}
	return util.CharacterCount(input)
}

// CurrentWordRange returns the current word bounds around the active cursor.
func (b *Buffer) CurrentWordRange() (Loc, Loc, bool) {
	c := b.GetActiveCursor()
	input, argstart := b.GetWord()
	if argstart == -1 || len(input) == 0 {
		return c.Loc, c.Loc, false
	}

	start := Loc{X: argstart, Y: c.Y}
	end := c.Loc
	lineLen := util.CharacterCount(b.LineBytes(c.Y))
	for end.X < lineLen && util.IsWordChar(b.RuneAt(end)) {
		end = end.Move(1, b)
	}
	return start, end, true
}

// HasGhostCompletion returns whether an inline completion preview is active.
func (b *Buffer) HasGhostCompletion() bool {
	return b.GhostText != ""
}

// AcceptGhostCompletion inserts the active inline completion preview.
func (b *Buffer) AcceptGhostCompletion() bool {
	if !b.HasGhostCompletion() {
		return false
	}

	b.applyCompletionByIndex(0)
	if value := b.completionValueAt(0); value != "" {
		completionUsage[value]++
	}
	return true
}

func (b *SharedBuffer) setCompletionItems(items []CompletionItem) {
	b.Completions = make([]string, 0, len(items))
	b.Suggestions = make([]string, 0, len(items))
	b.CompletionValues = make([]string, 0, len(items))
	b.CompletionSources = make([]string, 0, len(items))
	b.CompletionEdits = make([][]Delta, 0, len(items))
	for _, item := range items {
		b.Completions = append(b.Completions, item.Completion)
		b.Suggestions = append(b.Suggestions, item.Suggestion)
		b.CompletionValues = append(b.CompletionValues, firstNonEmpty(item.Insert, item.Suggestion))
		b.CompletionSources = append(b.CompletionSources, item.Provider)
		b.CompletionEdits = append(b.CompletionEdits, item.Edits)
	}
}

func (b *Buffer) completionValueAt(idx int) string {
	if idx >= 0 && idx < len(b.CompletionValues) && b.CompletionValues[idx] != "" {
		return b.CompletionValues[idx]
	}
	if idx >= 0 && idx < len(b.Suggestions) {
		return b.Suggestions[idx]
	}
	return ""
}

func (b *Buffer) completionProviderAt(idx int) string {
	if idx >= 0 && idx < len(b.CompletionSources) {
		return b.CompletionSources[idx]
	}
	return ""
}

func (b *Buffer) completionEditsAt(idx int) []Delta {
	if idx >= 0 && idx < len(b.CompletionEdits) {
		return b.CompletionEdits[idx]
	}
	return nil
}

// ShowGhostCompletion displays a non-inserted inline completion preview.
func (b *Buffer) ShowGhostCompletion(item CompletionItem) bool {
	if item.Completion == "" {
		return false
	}

	b.ClearAutocomplete()
	b.setCompletionItems([]CompletionItem{item})
	b.CurSuggestion = 0
	b.CompletionStart, b.CompletionEnd, _ = b.CurrentWordRange()
	b.GhostAt = b.GetActiveCursor().Loc
	b.GhostText = item.Completion
	return true
}

// ShowCompletionMenu activates popup menu mode without mutating the buffer.
func (b *Buffer) ShowCompletionMenu(items []CompletionItem) bool {
	if len(items) == 0 {
		return false
	}

	b.ClearAutocomplete()
	b.setCompletionItems(items)
	b.CompletionMenu = true
	b.CurSuggestion = -1
	b.CompletionStart, b.CompletionEnd, _ = b.CurrentWordRange()
	return true
}

// CycleCompletionMenu cycles an active menu completion session.
func (b *Buffer) CycleCompletionMenu(forward bool) bool {
	if len(b.Completions) == 0 {
		return false
	}
	if b.CurSuggestion == -1 {
		if forward {
			b.CurSuggestion = 0
		} else {
			b.CurSuggestion = len(b.Suggestions) - 1
		}
		b.CompletionMenu = true
		return true
	}
	if forward {
		b.CurSuggestion++
	} else {
		b.CurSuggestion--
	}
	if b.CurSuggestion >= len(b.Suggestions) {
		b.CurSuggestion = 0
	} else if b.CurSuggestion < 0 {
		b.CurSuggestion = len(b.Suggestions) - 1
	}
	b.CompletionMenu = true
	return true
}

// AcceptCompletionMenu inserts the currently selected popup completion.
func (b *Buffer) AcceptCompletionMenu() bool {
	if !b.CompletionMenu || b.CurSuggestion < 0 || b.CurSuggestion >= len(b.Suggestions) {
		return false
	}

	value := b.completionValueAt(b.CurSuggestion)
	if value == "" {
		return false
	}
	b.applyCompletionByIndex(b.CurSuggestion)
	completionUsage[value]++
	return true
}

func sortDeltasDescending(deltas []Delta) {
	sort.Slice(deltas, func(i, j int) bool {
		if deltas[i].Start.Y != deltas[j].Start.Y {
			return deltas[i].Start.Y > deltas[j].Start.Y
		}
		if deltas[i].Start.X != deltas[j].Start.X {
			return deltas[i].Start.X > deltas[j].Start.X
		}
		if deltas[i].End.Y != deltas[j].End.Y {
			return deltas[i].End.Y > deltas[j].End.Y
		}
		return deltas[i].End.X > deltas[j].End.X
	})
}

func insertedTextEnd(start Loc, text []byte) Loc {
	linecount := bytes.Count(text, []byte{'\n'})
	if linecount == 0 {
		return Loc{X: start.X + util.CharacterCount(text), Y: start.Y}
	}
	lastnl := bytes.LastIndex(text, []byte{'\n'})
	return Loc{X: util.CharacterCount(text[lastnl+1:]), Y: start.Y + linecount}
}

func (b *Buffer) applyCompletionByIndex(idx int) {
	if idx < 0 || idx >= len(b.Suggestions) {
		return
	}
	value := b.completionValueAt(idx)
	if value == "" {
		return
	}
	mainEdit := Delta{Text: []byte(value), Start: b.CompletionStart, End: b.CompletionEnd}
	deltas := []Delta{mainEdit}
	if edits := b.completionEditsAt(idx); len(edits) > 0 {
		deltas = append(deltas, edits...)
	}
	end := insertedTextEnd(mainEdit.Start, mainEdit.Text)
	sortDeltasDescending(deltas)
	b.MultipleReplace(deltas)
	b.GetActiveCursor().ResetSelection()
	b.GetActiveCursor().GotoLoc(end)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func completionItemKey(item CompletionItem) string {
	return firstNonEmpty(item.Insert, item.Suggestion) + "\x00" + item.Suggestion
}

func completionProviderPriority(provider string) int {
	switch provider {
	case completionProviderLSP:
		return 100
	case completionProviderBuffer:
		return 10
	default:
		return 0
	}
}

func dedupeCompletionItems(items []CompletionItem) []CompletionItem {
	bestByKey := make(map[string]CompletionItem, len(items))
	order := make([]string, 0, len(items))
	for _, item := range items {
		key := completionItemKey(item)
		best, ok := bestByKey[key]
		if !ok {
			bestByKey[key] = item
			order = append(order, key)
			continue
		}
		if completionProviderPriority(item.Provider) > completionProviderPriority(best.Provider) {
			bestByKey[key] = item
		}
	}
	out := items[:0]
	for _, key := range order {
		out = append(out, bestByKey[key])
	}
	return out
}

func (b *Buffer) activeCompletionItems() []CompletionItem {
	if len(b.Suggestions) == 0 {
		return nil
	}
	items := make([]CompletionItem, 0, len(b.Suggestions))
	for i, suggestion := range b.Suggestions {
		completion := ""
		if i < len(b.Completions) {
			completion = b.Completions[i]
		}
		insert := suggestion
		if value := b.completionValueAt(i); value != "" {
			insert = value
		}
		items = append(items, CompletionItem{
			Completion: completion,
			Suggestion: suggestion,
			Insert:     insert,
			Provider:   b.completionProviderAt(i),
			Edits:      b.completionEditsAt(i),
		})
	}
	return items
}

func (b *Buffer) selectedCompletionItemKey() string {
	if b.CurSuggestion < 0 || b.CurSuggestion >= len(b.Suggestions) {
		return ""
	}
	insert := b.completionValueAt(b.CurSuggestion)
	if insert == "" {
		insert = b.Suggestions[b.CurSuggestion]
	}
	return insert + "\x00" + b.Suggestions[b.CurSuggestion]
}

func completionItemIndexByKey(items []CompletionItem, key string) int {
	if key == "" {
		return -1
	}
	for i, item := range items {
		if completionItemKey(item) == key {
			return i
		}
	}
	return -1
}

func filterCompletionItemsByProvider(items []CompletionItem, provider string) []CompletionItem {
	out := items[:0]
	for _, item := range items {
		if item.Provider == provider {
			continue
		}
		out = append(out, item)
	}
	return out
}

func (b *Buffer) displayCompletionItems(items []CompletionItem, start, end Loc, selectedKey string) bool {
	prefixBytes := b.Substr(start, b.GetActiveCursor().Loc)
	prefix := string(prefixBytes)
	if len(items) == 0 {
		b.ClearAutocomplete()
		return false
	}

	b.ClearAutocomplete()
	b.CompletionStart = start
	b.CompletionEnd = end

	if len(items) == 1 && !b.HasTextRightOfCursor() && end == b.GetActiveCursor().Loc {
		insert := firstNonEmpty(items[0].Insert, items[0].Suggestion)
		if strings.HasPrefix(insert, prefix) {
			items[0].Completion = insert[len(prefix):]
			return b.ShowGhostCompletion(items[0])
		}
	}
	ok := b.ShowCompletionMenu(items)
	if !ok {
		return false
	}
	b.CurSuggestion = completionItemIndexByKey(items, selectedKey)
	return true
}

// DropProviderCompletions removes all active completion items from a provider and
// preserves the rest of the completion UI if possible.
func (b *Buffer) DropProviderCompletions(provider string) bool {
	if !(b.CompletionMenu || b.HasGhostCompletion()) {
		return false
	}
	selectedKey := b.selectedCompletionItemKey()
	items := filterCompletionItemsByProvider(b.activeCompletionItems(), provider)
	return b.displayCompletionItems(items, b.CompletionStart, b.CompletionEnd, selectedKey)
}

func candidatePrefixScore(prefix, value, label string) int {
	text := firstNonEmpty(value, label)
	if prefix == "" {
		return 0
	}
	if text == prefix {
		return -100000
	}
	if strings.HasPrefix(text, prefix) {
		return 800 + len(prefix)*20
	}
	if strings.HasPrefix(label, prefix) {
		return 700 + len(prefix)*20
	}
	return -100000
}

func candidateUnderscorePenalty(prefix, label string) int {
	if strings.HasPrefix(label, "_") && !strings.HasPrefix(prefix, "_") {
		return -250
	}
	return 0
}

func rankCompletionItems(prefix string, items []CompletionItem) []CompletionItem {
	type rankedItem struct {
		item  CompletionItem
		score int
	}

	ranked := make([]rankedItem, 0, len(items))
	for _, item := range items {
		insert := firstNonEmpty(item.Insert, item.Suggestion)
		score := candidatePrefixScore(prefix, insert, item.Suggestion)
		if score <= -100000 {
			continue
		}
		if item.Preselect {
			score += 5000
		}
		if item.Deprecated {
			score -= 500
		}
		score += candidateUnderscorePenalty(prefix, item.Suggestion)
		score += completionUsage[insert] * 50
		ranked = append(ranked, rankedItem{item: item, score: score})
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		leftSort := firstNonEmpty(ranked[i].item.SortText, ranked[i].item.Suggestion)
		rightSort := firstNonEmpty(ranked[j].item.SortText, ranked[j].item.Suggestion)
		if leftSort != rightSort {
			return leftSort < rightSort
		}
		leftInsert := firstNonEmpty(ranked[i].item.Insert, ranked[i].item.Suggestion)
		rightInsert := firstNonEmpty(ranked[j].item.Insert, ranked[j].item.Suggestion)
		if leftInsert != rightInsert {
			return leftInsert < rightInsert
		}
		if ranked[i].item.Suggestion != ranked[j].item.Suggestion {
			return ranked[i].item.Suggestion < ranked[j].item.Suggestion
		}
		return ranked[i].item.Completion < ranked[j].item.Completion
	})

	items = items[:0]
	for _, item := range ranked {
		items = append(items, item.item)
	}
	return items
}

type externalCompletionItem struct {
	Insert     string                   `json:"insert"`
	Label      string                   `json:"label"`
	Edits      []externalCompletionEdit `json:"additionalTextEdits"`
	SortText   string                   `json:"sortText"`
	Preselect  bool                     `json:"preselect"`
	Deprecated bool                     `json:"deprecated"`
}

type externalCompletionEdit struct {
	Text  string `json:"text"`
	Start struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"start"`
	End struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"end"`
}

// ShowExternalCompletionsJSON parses external completion items, ranks them, and
// routes them through the shared ghost-text/popup UI.
func (b *Buffer) ShowExternalCompletionsJSON(serialized string, startX, startY, endX, endY int) bool {
	items := make([]CompletionItem, 0)
	var externalItems []externalCompletionItem
	if err := json.Unmarshal([]byte(serialized), &externalItems); err != nil {
		return false
	}
	for _, item := range externalItems {
		insert := firstNonEmpty(item.Insert, item.Label)
		label := firstNonEmpty(item.Label, insert)
		edits := make([]Delta, 0, len(item.Edits))
		for _, edit := range item.Edits {
			edits = append(edits, Delta{
				Text:  []byte(edit.Text),
				Start: Loc{X: edit.Start.X, Y: edit.Start.Y},
				End:   Loc{X: edit.End.X, Y: edit.End.Y},
			})
		}
		items = append(items, CompletionItem{
			Completion: insert,
			Suggestion: label,
			Insert:     insert,
			Provider:   completionProviderLSP,
			Edits:      edits,
			SortText:   item.SortText,
			Preselect:  item.Preselect,
			Deprecated: item.Deprecated,
		})
	}
	if len(items) == 0 {
		return false
	}

	start := Loc{X: startX, Y: startY}
	end := Loc{X: endX, Y: endY}
	selectedKey := ""
	if (b.CompletionMenu || b.HasGhostCompletion()) && b.CompletionStart == start && b.CompletionEnd == end {
		selectedKey = b.selectedCompletionItemKey()
		items = append(items, b.activeCompletionItems()...)
		items = dedupeCompletionItems(items)
	}
	prefixBytes := b.Substr(start, b.GetActiveCursor().Loc)
	prefix := string(prefixBytes)
	items = rankCompletionItems(prefix, items)
	if len(items) == 0 {
		b.ClearAutocomplete()
		return false
	}
	return b.displayCompletionItems(items, start, end, selectedKey)
}

// StartCompletion resolves completion items and chooses between inline and menu modes.
func (b *Buffer) StartCompletion(provider CompletionProvider) bool {
	items := provider(b)
	if len(items) == 0 {
		return false
	}
	if len(items) == 1 && !b.HasTextRightOfCursor() {
		return b.ShowGhostCompletion(items[0])
	}
	return b.ShowCompletionMenu(items)
}

// StartAutomaticCompletion shows ghost text for a single sensible completion,
// or a popup when multiple completions are available.
func (b *Buffer) StartAutomaticCompletion(provider CompletionProvider) bool {
	items := provider(b)
	if len(items) == 0 {
		return false
	}
	if len(items) == 1 && !b.HasTextRightOfCursor() {
		return b.ShowGhostCompletion(items[0])
	}
	if len(items) == 1 {
		return false
	}
	return b.ShowCompletionMenu(items)
}

func (b *Buffer) GetSuggestions() {

}

// Autocomplete starts the autocomplete process
func (b *Buffer) Autocomplete(c Completer) bool {
	b.ClearAutocomplete()
	b.Completions, b.Suggestions = c(b)
	if len(b.Completions) != len(b.Suggestions) || len(b.Completions) == 0 {
		return false
	}
	b.CurSuggestion = -1
	b.CycleAutocomplete(true)
	return true
}

// CycleAutocomplete moves to the next suggestion
func (b *Buffer) CycleAutocomplete(forward bool) {
	prevSuggestion := b.CurSuggestion

	if forward {
		b.CurSuggestion++
	} else {
		b.CurSuggestion--
	}
	if b.CurSuggestion >= len(b.Suggestions) {
		b.CurSuggestion = 0
	} else if b.CurSuggestion < 0 {
		b.CurSuggestion = len(b.Suggestions) - 1
	}

	c := b.GetActiveCursor()
	start := c.Loc
	end := c.Loc
	if prevSuggestion < len(b.Suggestions) && prevSuggestion >= 0 {
		start = end.Move(-util.CharacterCountInString(b.Completions[prevSuggestion]), b)
	}

	b.Replace(start, end, b.Completions[b.CurSuggestion])
	if len(b.Suggestions) > 1 {
		b.HasSuggestions = true
	}
}

// GetWord gets the most recent word separated by any separator
// (whitespace, punctuation, any non alphanumeric character)
func (b *Buffer) GetWord() ([]byte, int) {
	c := b.GetActiveCursor()
	l := b.LineBytes(c.Y)
	l = util.SliceStart(l, c.X)

	if c.X == 0 || util.IsWhitespace(b.RuneAt(c.Loc.Move(-1, b))) {
		return []byte{}, -1
	}

	if util.IsNonWordChar(b.RuneAt(c.Loc.Move(-1, b))) {
		return []byte{}, c.X
	}

	args := bytes.FieldsFunc(l, util.IsNonWordChar)
	input := args[len(args)-1]
	return input, c.X - util.CharacterCount(input)
}

// GetArg gets the most recent word (separated by ' ' only)
func (b *Buffer) GetArg() (string, int) {
	c := b.GetActiveCursor()
	l := b.LineBytes(c.Y)
	l = util.SliceStart(l, c.X)

	args := bytes.Split(l, []byte{' '})
	input := string(args[len(args)-1])
	argstart := 0
	for i, a := range args {
		if i == len(args)-1 {
			break
		}
		argstart += util.CharacterCount(a) + 1
	}

	return input, argstart
}

// FileComplete autocompletes filenames
func FileComplete(b *Buffer) ([]string, []string) {
	c := b.GetActiveCursor()
	input, argstart := b.GetArg()

	sep := string(os.PathSeparator)
	dirs := strings.Split(input, sep)

	var files []fs.DirEntry
	var err error
	if len(dirs) > 1 {
		directories := strings.Join(dirs[:len(dirs)-1], sep) + sep

		directories, _ = util.ReplaceHome(directories)
		files, err = os.ReadDir(directories)
	} else {
		files, err = os.ReadDir(".")
	}

	if err != nil {
		return nil, nil
	}

	var suggestions []string
	for _, f := range files {
		name := f.Name()
		if f.IsDir() {
			name += sep
		}
		if strings.HasPrefix(name, dirs[len(dirs)-1]) {
			suggestions = append(suggestions, name)
		}
	}

	sort.Strings(suggestions)
	completions := make([]string, len(suggestions))
	for i := range suggestions {
		var complete string
		if len(dirs) > 1 {
			complete = strings.Join(dirs[:len(dirs)-1], sep) + sep + suggestions[i]
		} else {
			complete = suggestions[i]
		}
		completions[i] = util.SliceEndStr(complete, c.X-argstart)
	}

	return completions, suggestions
}

// BufferComplete autocompletes based on previous words in the buffer
func BufferComplete(b *Buffer) ([]string, []string) {
	c := b.GetActiveCursor()
	input, argstart := b.GetWord()

	if argstart == -1 {
		return []string{}, []string{}
	}

	inputLen := util.CharacterCount(input)

	suggestionsSet := make(map[string]struct{})

	var suggestions []string
	for i := c.Y; i >= 0; i-- {
		l := b.LineBytes(i)
		words := bytes.FieldsFunc(l, util.IsNonWordChar)
		for _, w := range words {
			if bytes.HasPrefix(w, input) && util.CharacterCount(w) > inputLen {
				strw := string(w)
				if _, ok := suggestionsSet[strw]; !ok {
					suggestionsSet[strw] = struct{}{}
					suggestions = append(suggestions, strw)
				}
			}
		}
	}
	for i := c.Y + 1; i < b.LinesNum(); i++ {
		l := b.LineBytes(i)
		words := bytes.FieldsFunc(l, util.IsNonWordChar)
		for _, w := range words {
			if bytes.HasPrefix(w, input) && util.CharacterCount(w) > inputLen {
				strw := string(w)
				if _, ok := suggestionsSet[strw]; !ok {
					suggestionsSet[strw] = struct{}{}
					suggestions = append(suggestions, strw)
				}
			}
		}
	}
	completions := make([]string, len(suggestions))
	for i := range suggestions {
		completions[i] = util.SliceEndStr(suggestions[i], c.X-argstart)
	}

	return completions, suggestions
}

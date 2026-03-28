package buffer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func singleItemProvider(item CompletionItem) CompletionProvider {
	return func(*Buffer) []CompletionItem {
		return []CompletionItem{item}
	}
}

func TestStartCompletionUsesGhostTextForSingleCompletionAtLineEnd(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("pri", "", BTDefault)
	b.GetActiveCursor().GotoLoc(Loc{3, 0})

	ok := b.StartCompletion(singleItemProvider(CompletionItem{
		Completion: "nt",
		Suggestion: "print",
	}))

	assert.True(ok)
	assert.True(b.HasGhostCompletion())
	assert.False(b.CompletionMenu)
	assert.Equal("pri", string(b.Bytes()))
	assert.Equal("nt", b.GhostText)

	ok = b.AcceptGhostCompletion()

	assert.True(ok)
	assert.Equal("print", string(b.Bytes()))
	assert.False(b.HasGhostCompletion())
}

func TestStartCompletionUsesMenuWhenTextExistsToTheRight(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("pri()", "", BTDefault)
	b.GetActiveCursor().GotoLoc(Loc{3, 0})

	ok := b.StartCompletion(singleItemProvider(CompletionItem{
		Completion: "nt",
		Suggestion: "print",
	}))

	assert.True(ok)
	assert.False(b.HasGhostCompletion())
	assert.True(b.CompletionMenu)
	assert.Equal(-1, b.CurSuggestion)
	assert.Equal("pri()", string(b.Bytes()))
	assert.Equal([]string{"print"}, b.Suggestions)

	ok = b.AcceptCompletionMenu()

	assert.False(ok)
	assert.Equal("pri()", string(b.Bytes()))

	ok = b.CycleCompletionMenu(true)

	assert.True(ok)
	assert.Equal(0, b.CurSuggestion)

	ok = b.AcceptCompletionMenu()

	assert.True(ok)
	assert.Equal("print()", string(b.Bytes()))
}

func TestBufferCompleteProviderFeedsCoreCompletionFlow(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("hello\nhe", "", BTDefault)
	b.GetActiveCursor().GotoLoc(Loc{2, 1})

	ok := b.StartCompletion(BufferCompleteProvider)

	assert.True(ok)
	assert.True(b.HasGhostCompletion())
	assert.Equal("llo", b.GhostText)
	assert.Equal([]string{"hello"}, b.Suggestions)
}

func TestStartAutomaticCompletionRequiresSingleCandidateAndLineEnd(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("hello\nhe", "", BTDefault)
	b.GetActiveCursor().GotoLoc(Loc{2, 1})

	ok := b.StartAutomaticCompletion(BufferCompleteProvider)

	assert.True(ok)
	assert.True(b.HasGhostCompletion())

	b = NewBufferFromString("hello\nhelp\nhe", "", BTDefault)
	b.GetActiveCursor().GotoLoc(Loc{2, 2})

	ok = b.StartAutomaticCompletion(BufferCompleteProvider)

	assert.True(ok)
	assert.False(b.HasGhostCompletion())
	assert.True(b.CompletionMenu)
}

func TestShowExternalCompletionsJSONUsesGhostForSinglePrefixMatch(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("pri", "", BTDefault)
	b.GetActiveCursor().GotoLoc(Loc{3, 0})

	ok := b.ShowExternalCompletionsJSON(`[{"insert":"print","label":"print","sortText":"2","preselect":false,"deprecated":false}]`, 0, 0, 3, 0)

	assert.True(ok)
	assert.True(b.HasGhostCompletion())
	assert.Equal("nt", b.GhostText)
}

func TestShowExternalCompletionsJSONUsesPopupAndPreselectRanking(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("pr", "", BTDefault)
	b.GetActiveCursor().GotoLoc(Loc{2, 0})

	ok := b.ShowExternalCompletionsJSON(`[{"insert":"print","label":"print","sortText":"2","preselect":false,"deprecated":false},{"insert":"property","label":"property","sortText":"9","preselect":true,"deprecated":false}]`, 0, 0, 2, 0)

	assert.True(ok)
	assert.True(b.CompletionMenu)
	assert.Equal([]string{"property", "print"}, b.Suggestions)
}

func TestShowExternalCompletionsJSONFiltersExactAndCaseMismatchedItems(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("filter", "", BTDefault)
	b.GetActiveCursor().GotoLoc(Loc{6, 0})

	ok := b.ShowExternalCompletionsJSON(`[{"insert":"filter","label":"filter","sortText":"1","preselect":false,"deprecated":false},{"insert":"FilterMap","label":"FilterMap","sortText":"2","preselect":false,"deprecated":false}]`, 0, 0, 6, 0)

	assert.False(ok)
	assert.False(b.HasGhostCompletion())
	assert.False(b.CompletionMenu)
}

func TestShowExternalCompletionsJSONMergesWithExistingPopupItems(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("fil", "", BTDefault)
	b.GetActiveCursor().GotoLoc(Loc{3, 0})

	ok := b.ShowCompletionMenu([]CompletionItem{
		{Completion: "ter", Suggestion: "filter", Insert: "filter"},
		{Completion: "e", Suggestion: "file", Insert: "file"},
	})
	assert.True(ok)
	assert.True(b.CompletionMenu)
	b.CycleCompletionMenu(true)
	assert.Equal(0, b.CurSuggestion)
	selected := b.Suggestions[b.CurSuggestion]

	ok = b.ShowExternalCompletionsJSON(`[{"insert":"filter","label":"filter","sortText":"1","preselect":false,"deprecated":false}]`, 0, 0, 3, 0)

	assert.True(ok)
	assert.True(b.CompletionMenu)
	assert.False(b.HasGhostCompletion())
	assert.Len(b.Suggestions, 2)
	assert.Contains(b.Suggestions, "filter")
	assert.Contains(b.Suggestions, "file")
	assert.GreaterOrEqual(b.CurSuggestion, 0)
	assert.Equal(selected, b.Suggestions[b.CurSuggestion])
}

func TestShowExternalCompletionsJSONSortIsDeterministicOnTies(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("fi", "", BTDefault)
	b.GetActiveCursor().GotoLoc(Loc{2, 0})

	ok := b.ShowExternalCompletionsJSON(`[
		{"insert":"filter","label":"filter","sortText":"","preselect":false,"deprecated":false},
		{"insert":"finalize","label":"finalize","sortText":"","preselect":false,"deprecated":false},
		{"insert":"find","label":"find","sortText":"","preselect":false,"deprecated":false}
	]`, 0, 0, 2, 0)

	assert.True(ok)
	assert.True(b.CompletionMenu)
	assert.Equal([]string{"filter", "finalize", "find"}, b.Suggestions)

	ok = b.ShowExternalCompletionsJSON(`[
		{"insert":"find","label":"find","sortText":"","preselect":false,"deprecated":false},
		{"insert":"filter","label":"filter","sortText":"","preselect":false,"deprecated":false},
		{"insert":"finalize","label":"finalize","sortText":"","preselect":false,"deprecated":false}
	]`, 0, 0, 2, 0)

	assert.True(ok)
	assert.Equal([]string{"filter", "finalize", "find"}, b.Suggestions)
}

func TestShowExternalCompletionsJSONPreservesFreshProviderMetadataWhenMerging(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("pass", "", BTDefault)
	b.GetActiveCursor().GotoLoc(Loc{4, 0})

	ok := b.ShowExternalCompletionsJSON(`[
		{"insert":"passes_through_hut=","label":"passes_through_hut=","sortText":"2","preselect":true,"deprecated":false},
		{"insert":"passes_through_hut_id=","label":"passes_through_hut_id=","sortText":"1","preselect":false,"deprecated":false}
	]`, 0, 0, 4, 0)

	assert.True(ok)
	assert.Equal([]string{"passes_through_hut=", "passes_through_hut_id="}, b.Suggestions)

	b.CycleCompletionMenu(true)
	assert.Equal(0, b.CurSuggestion)

	ok = b.ShowExternalCompletionsJSON(`[
		{"insert":"passes_through_hut=","label":"passes_through_hut=","sortText":"2","preselect":true,"deprecated":false},
		{"insert":"passes_through_hut_id=","label":"passes_through_hut_id=","sortText":"1","preselect":false,"deprecated":false}
	]`, 0, 0, 4, 0)

	assert.True(ok)
	assert.Equal([]string{"passes_through_hut=", "passes_through_hut_id="}, b.Suggestions)
	assert.Equal(0, b.CurSuggestion)
}

func TestDedupeCompletionItemsPrefersLSPOverBuffer(t *testing.T) {
	assert := assert.New(t)

	items := dedupeCompletionItems([]CompletionItem{
		{Suggestion: "filter", Insert: "filter", Provider: "buffer", SortText: "9"},
		{Suggestion: "filter", Insert: "filter", Provider: "lsp", SortText: "1", Preselect: true},
	})

	assert.Len(items, 1)
	assert.Equal("lsp", items[0].Provider)
	assert.Equal("1", items[0].SortText)
	assert.True(items[0].Preselect)

	items = dedupeCompletionItems([]CompletionItem{
		{Suggestion: "filter", Insert: "filter", Provider: "lsp", SortText: "1", Preselect: true},
		{Suggestion: "filter", Insert: "filter", Provider: "buffer", SortText: "9"},
	})

	assert.Len(items, 1)
	assert.Equal("lsp", items[0].Provider)
	assert.Equal("1", items[0].SortText)
	assert.True(items[0].Preselect)
}

func TestDropProviderCompletionsKeepsBufferPopup(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("fil", "", BTDefault)
	b.GetActiveCursor().GotoLoc(Loc{3, 0})
	b.CompletionStart = Loc{0, 0}
	b.CompletionEnd = Loc{3, 0}
	ok := b.ShowCompletionMenu([]CompletionItem{
		{Completion: "ter", Suggestion: "filter", Insert: "filter", Provider: "buffer"},
		{Completion: "e", Suggestion: "file", Insert: "file", Provider: "buffer"},
		{Completion: "ter", Suggestion: "filter", Insert: "filter", Provider: "lsp"},
	})
	assert.True(ok)
	assert.True(b.CompletionMenu)

	ok = b.DropProviderCompletions("lsp")

	assert.True(ok)
	assert.True(b.CompletionMenu)
	assert.False(b.HasGhostCompletion())
	assert.Equal([]string{"filter", "file"}, b.Suggestions)
	assert.Equal([]string{"buffer", "buffer"}, b.CompletionSources)
}

func TestAcceptCompletionMenuAppliesAdditionalTextEdits(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("print(r)", "", BTDefault)
	b.GetActiveCursor().GotoLoc(Loc{7, 0})

	ok := b.ShowExternalCompletionsJSON(`[
		{"insert":"re","label":"re","additionalTextEdits":[{"text":"import re\n","start":{"x":0,"y":0},"end":{"x":0,"y":0}}],"sortText":"1","preselect":true,"deprecated":false}
	]`, 6, 0, 7, 0)

	assert.True(ok)
	assert.True(b.CompletionMenu)
	b.CycleCompletionMenu(true)

	ok = b.AcceptCompletionMenu()

	assert.True(ok)
	assert.Equal("import re\nprint(re)", string(b.Bytes()))
	assert.Equal(Loc{8, 0}, b.GetActiveCursor().Loc)
}

func TestAcceptCompletionMenuMovesCursorToInsertedEnd(t *testing.T) {
	assert := assert.New(t)

	b := NewBufferFromString("pass", "", BTDefault)
	b.GetActiveCursor().GotoLoc(Loc{4, 0})

	ok := b.ShowCompletionMenu([]CompletionItem{{
		Completion: "es_through_hut=",
		Suggestion: "passes_through_hut=",
		Insert:     "passes_through_hut=",
		Provider:   "lsp",
	}})
	assert.True(ok)

	b.CycleCompletionMenu(true)
	ok = b.AcceptCompletionMenu()

	assert.True(ok)
	assert.Equal("passes_through_hut=", string(b.Bytes()))
	assert.Equal(Loc{19, 0}, b.GetActiveCursor().Loc)
}

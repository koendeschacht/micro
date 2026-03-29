package action

import (
	"regexp"
	"strings"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/util"
)

var pythonDefRegex = regexp.MustCompile(`^([ \t]*)(?:async\s+def|def)\s+\w+`)
var pythonFuncNameRegex = regexp.MustCompile(`^\s*(?:async\s+def|def)\s+(\w+)`)
var pythonClassRegex = regexp.MustCompile(`^([ \t]*)class\s+\w+`)
var pythonClassNameRegex = regexp.MustCompile(`^\s*class\s+(\w+)`)

type textObjectTarget string

const (
	textObjectWord         textObjectTarget = "Word"
	textObjectFunction     textObjectTarget = "Function"
	textObjectClass        textObjectTarget = "Class"
	textObjectDoubleQuotes textObjectTarget = "DoubleQuotes"
	textObjectSingleQuotes textObjectTarget = "SingleQuotes"
	textObjectParens       textObjectTarget = "Parens"
	textObjectBrackets     textObjectTarget = "Brackets"
	textObjectParagraph    textObjectTarget = "Paragraph"
)

func (h *BufPane) selectTextObject(target textObjectTarget) bool {
	start, end, ok := h.textObjectRange(target)
	if !ok || !start.LessThan(end) {
		InfoBar.Message("No ", describeTextObject(target), " found")
		return true
	}

	h.Cursor.Deselect(true)
	h.Cursor.SetSelectionStart(start)
	h.Cursor.SetSelectionEnd(end)
	h.Cursor.OrigSelection = h.Cursor.CurSelection
	h.Cursor.Loc = end
	h.Relocate()
	return true
}

func (h *BufPane) textObjectRange(target textObjectTarget) (buffer.Loc, buffer.Loc, bool) {
	var (
		start buffer.Loc
		end   buffer.Loc
		ok    bool
	)

	switch target {
	case textObjectWord:
		start, end, ok = h.wordObjectRange()
	case textObjectFunction:
		start, end, ok = h.functionObjectRange()
	case textObjectClass:
		start, end, ok = h.classObjectRange()
	case textObjectDoubleQuotes:
		start, end, ok = h.quoteObjectRange('"')
	case textObjectSingleQuotes:
		start, end, ok = h.quoteObjectRange('\'')
	case textObjectParens:
		start, end, ok = h.delimitedObjectRange('(', ')')
	case textObjectBrackets:
		start, end, ok = h.delimitedObjectRange('[', ']')
	case textObjectParagraph:
		start, end, ok = h.paragraphObjectRange()
	}
	return start, end, ok
}

func (h *BufPane) findTextObject(target textObjectTarget) bool {
	text, searchFrom, ok := h.searchTextObject(target)
	if !ok || text == "" {
		InfoBar.Message("No ", describeTextObject(target), " found")
		return true
	}
	return h.literalSearchFrom(text, searchFrom)
}

func (h *BufPane) yankTextObject(target textObjectTarget) bool {
	origLoc := h.Cursor.Loc
	origLastVisualX := h.Cursor.LastVisualX
	origLastWrappedVisualX := h.Cursor.LastWrappedVisualX

	if !h.selectTextObject(target) {
		return false
	}
	if !h.Cursor.HasSelection() {
		return true
	}
	h.startTextObjectPreview(func() {
		h.Copy()
		h.Cursor.ResetSelection()
		h.Cursor.Loc = origLoc
		h.Cursor.LastVisualX = origLastVisualX
		h.Cursor.LastWrappedVisualX = origLastWrappedVisualX
		h.Relocate()
	})
	return true
}

func (h *BufPane) deleteTextObject(target textObjectTarget) bool {
	if !h.selectTextObject(target) {
		return false
	}
	if !h.Cursor.HasSelection() {
		return true
	}
	h.startTextObjectPreview(func() {
		h.Cut()
	})
	return true
}

func (h *BufPane) findWordObject() bool         { return h.findTextObject(textObjectWord) }
func (h *BufPane) findFunctionObject() bool     { return h.findTextObject(textObjectFunction) }
func (h *BufPane) findClassObject() bool        { return h.findTextObject(textObjectClass) }
func (h *BufPane) findDoubleQuotesObject() bool { return h.findTextObject(textObjectDoubleQuotes) }
func (h *BufPane) findSingleQuotesObject() bool { return h.findTextObject(textObjectSingleQuotes) }
func (h *BufPane) findParensObject() bool       { return h.findTextObject(textObjectParens) }
func (h *BufPane) findBracketsObject() bool     { return h.findTextObject(textObjectBrackets) }
func (h *BufPane) findParagraphObject() bool    { return h.findTextObject(textObjectParagraph) }

func (h *BufPane) selectWordObject() bool         { return h.selectTextObject(textObjectWord) }
func (h *BufPane) selectFunctionObject() bool     { return h.selectTextObject(textObjectFunction) }
func (h *BufPane) selectClassObject() bool        { return h.selectTextObject(textObjectClass) }
func (h *BufPane) selectDoubleQuotesObject() bool { return h.selectTextObject(textObjectDoubleQuotes) }
func (h *BufPane) selectSingleQuotesObject() bool { return h.selectTextObject(textObjectSingleQuotes) }
func (h *BufPane) selectParensObject() bool       { return h.selectTextObject(textObjectParens) }
func (h *BufPane) selectBracketsObject() bool     { return h.selectTextObject(textObjectBrackets) }
func (h *BufPane) selectParagraphObject() bool    { return h.selectTextObject(textObjectParagraph) }

func (h *BufPane) yankWordObject() bool         { return h.yankTextObject(textObjectWord) }
func (h *BufPane) yankFunctionObject() bool     { return h.yankTextObject(textObjectFunction) }
func (h *BufPane) yankClassObject() bool        { return h.yankTextObject(textObjectClass) }
func (h *BufPane) yankDoubleQuotesObject() bool { return h.yankTextObject(textObjectDoubleQuotes) }
func (h *BufPane) yankSingleQuotesObject() bool { return h.yankTextObject(textObjectSingleQuotes) }
func (h *BufPane) yankParensObject() bool       { return h.yankTextObject(textObjectParens) }
func (h *BufPane) yankBracketsObject() bool     { return h.yankTextObject(textObjectBrackets) }
func (h *BufPane) yankParagraphObject() bool    { return h.yankTextObject(textObjectParagraph) }

func (h *BufPane) deleteWordObject() bool         { return h.deleteTextObject(textObjectWord) }
func (h *BufPane) deleteFunctionObject() bool     { return h.deleteTextObject(textObjectFunction) }
func (h *BufPane) deleteClassObject() bool        { return h.deleteTextObject(textObjectClass) }
func (h *BufPane) deleteDoubleQuotesObject() bool { return h.deleteTextObject(textObjectDoubleQuotes) }
func (h *BufPane) deleteSingleQuotesObject() bool { return h.deleteTextObject(textObjectSingleQuotes) }
func (h *BufPane) deleteParensObject() bool       { return h.deleteTextObject(textObjectParens) }
func (h *BufPane) deleteBracketsObject() bool     { return h.deleteTextObject(textObjectBrackets) }
func (h *BufPane) deleteParagraphObject() bool    { return h.deleteTextObject(textObjectParagraph) }

func (h *BufPane) searchTextObject(target textObjectTarget) (string, buffer.Loc, bool) {
	if target == textObjectFunction {
		if name, loc, ok := h.functionSearchName(); ok {
			return name, loc, true
		}
		return "", buffer.Loc{}, false
	}
	if target == textObjectClass {
		if name, loc, ok := h.classSearchName(); ok {
			return name, loc, true
		}
		return "", buffer.Loc{}, false
	}

	start, end, ok := h.textObjectRange(target)
	if !ok || !start.LessThan(end) {
		return "", buffer.Loc{}, false
	}
	return string(h.Buf.Substr(start, end)), end, true
}

func (h *BufPane) functionSearchName() (string, buffer.Loc, bool) {
	if h.Buf.FileType() != "python" {
		return "", buffer.Loc{}, false
	}

	start, end, ok := h.pythonFunctionObjectRange()
	if !ok {
		return "", buffer.Loc{}, false
	}

	for y := start.Y; y < h.Buf.LinesNum() && y <= end.Y; y++ {
		match := pythonFuncNameRegex.FindStringSubmatch(h.Buf.Line(y))
		if match != nil {
			return match[1], buffer.Loc{X: util.CharacterCountInString(h.Buf.Line(y)), Y: y}, true
		}
	}

	return "", buffer.Loc{}, false
}

func (h *BufPane) classSearchName() (string, buffer.Loc, bool) {
	if h.Buf.FileType() != "python" {
		return "", buffer.Loc{}, false
	}

	start, end, ok := h.pythonClassObjectRange()
	if !ok {
		return "", buffer.Loc{}, false
	}

	for y := start.Y; y < h.Buf.LinesNum() && y <= end.Y; y++ {
		match := pythonClassNameRegex.FindStringSubmatch(h.Buf.Line(y))
		if match != nil {
			return match[1], buffer.Loc{X: util.CharacterCountInString(h.Buf.Line(y)), Y: y}, true
		}
	}

	return "", buffer.Loc{}, false
}

func (h *BufPane) literalSearchFrom(text string, from buffer.Loc) bool {
	match, found, err := h.Buf.FindNext(text, h.Buf.Start(), h.Buf.End(), from, true, false)
	if err != nil {
		InfoBar.Error(err)
		return true
	}
	if !found {
		h.Cursor.ResetSelection()
		InfoBar.Message("No matches found")
		return true
	}

	h.Cursor.SetSelectionStart(match[0])
	h.Cursor.SetSelectionEnd(match[1])
	h.Cursor.OrigSelection[0] = h.Cursor.CurSelection[0]
	h.Cursor.OrigSelection[1] = h.Cursor.CurSelection[1]
	h.GotoLoc(h.Cursor.CurSelection[1])
	h.Buf.LastSearch = text
	h.Buf.LastSearchRegex = false
	h.Buf.HighlightSearch = h.Buf.Settings["hlsearch"].(bool)
	return true
}

func (h *BufPane) wordObjectRange() (buffer.Loc, buffer.Loc, bool) {
	loc := h.Cursor.Loc
	if len(h.Buf.LineBytes(loc.Y)) == 0 || !util.IsWordChar(h.Buf.RuneAt(loc)) {
		return buffer.Loc{}, buffer.Loc{}, false
	}

	start := loc
	for start.X > 0 && util.IsWordChar(h.Buf.RuneAt(start.Move(-1, h.Buf))) {
		start.X--
	}

	end := loc.Move(1, h.Buf)
	lineLen := util.CharacterCount(h.Buf.LineBytes(loc.Y))
	for end.X < lineLen && util.IsWordChar(h.Buf.RuneAt(end)) {
		end.X++
	}

	return start, end, true
}

func (h *BufPane) paragraphObjectRange() (buffer.Loc, buffer.Loc, bool) {
	line := h.Cursor.Y
	if h.Buf.LinesNum() == 0 {
		return buffer.Loc{}, buffer.Loc{}, false
	}

	if strings.TrimSpace(h.Buf.Line(line)) == "" {
		found := false
		for y := line + 1; y < h.Buf.LinesNum(); y++ {
			if strings.TrimSpace(h.Buf.Line(y)) != "" {
				line = y
				found = true
				break
			}
		}
		if !found {
			for y := h.Cursor.Y - 1; y >= 0; y-- {
				if strings.TrimSpace(h.Buf.Line(y)) != "" {
					line = y
					found = true
					break
				}
			}
		}
		if !found {
			return buffer.Loc{}, buffer.Loc{}, false
		}
	}

	startY := line
	for startY > 0 && strings.TrimSpace(h.Buf.Line(startY-1)) != "" {
		startY--
	}

	endY := line
	for endY+1 < h.Buf.LinesNum() && strings.TrimSpace(h.Buf.Line(endY+1)) != "" {
		endY++
	}

	return buffer.Loc{X: 0, Y: startY}, buffer.Loc{X: 0, Y: endY + 1}, true
}

func (h *BufPane) quoteObjectRange(quote rune) (buffer.Loc, buffer.Loc, bool) {
	line := []rune(h.Buf.Line(h.Cursor.Y))
	if len(line) == 0 {
		return buffer.Loc{}, buffer.Loc{}, false
	}

	x := h.Cursor.X
	if x >= len(line) {
		x = len(line) - 1
	}

	open := -1
	for i := x; i >= 0; i-- {
		if line[i] == quote && !isEscapedRune(line, i) {
			open = i
			break
		}
	}
	if open == -1 {
		return buffer.Loc{}, buffer.Loc{}, false
	}

	close := -1
	for i := open + 1; i < len(line); i++ {
		if line[i] == quote && !isEscapedRune(line, i) {
			close = i
			if i >= h.Cursor.X {
				break
			}
		}
	}
	if close == -1 || close <= open {
		return buffer.Loc{}, buffer.Loc{}, false
	}

	if h.Cursor.X < open || h.Cursor.X > close {
		return buffer.Loc{}, buffer.Loc{}, false
	}

	return buffer.Loc{X: open + 1, Y: h.Cursor.Y}, buffer.Loc{X: close, Y: h.Cursor.Y}, true
}

func (h *BufPane) delimitedObjectRange(open, close rune) (buffer.Loc, buffer.Loc, bool) {
	content := []rune(util.String(h.Buf.Bytes()))
	if len(content) == 0 {
		return buffer.Loc{}, buffer.Loc{}, false
	}

	offset := h.locToRuneOffset(h.Cursor.Loc)
	if offset >= len(content) {
		offset = len(content) - 1
	}

	openIndex := -1
	depth := 0
	for i := offset; i >= 0; i-- {
		switch content[i] {
		case close:
			depth++
		case open:
			if depth == 0 {
				openIndex = i
				i = -1
				continue
			}
			depth--
		}
	}
	if openIndex == -1 {
		return buffer.Loc{}, buffer.Loc{}, false
	}

	closeIndex := -1
	depth = 0
	for i := openIndex + 1; i < len(content); i++ {
		switch content[i] {
		case open:
			depth++
		case close:
			if depth == 0 {
				closeIndex = i
				i = len(content)
				continue
			}
			depth--
		}
	}
	if closeIndex == -1 || offset < openIndex || offset > closeIndex {
		return buffer.Loc{}, buffer.Loc{}, false
	}

	return h.runeOffsetToLoc(openIndex + 1), h.runeOffsetToLoc(closeIndex), true
}

func (h *BufPane) functionObjectRange() (buffer.Loc, buffer.Loc, bool) {
	if h.Buf.FileType() == "python" {
		return h.pythonFunctionObjectRange()
	}
	return h.delimitedObjectRange('{', '}')
}

func (h *BufPane) classObjectRange() (buffer.Loc, buffer.Loc, bool) {
	if h.Buf.FileType() == "python" {
		return h.pythonClassObjectRange()
	}
	return h.delimitedObjectRange('{', '}')
}

func (h *BufPane) pythonFunctionObjectRange() (buffer.Loc, buffer.Loc, bool) {
	if start, end, ok := h.pythonBlockRangeForDecorator(pythonDefRegex); ok {
		return start, end, true
	}

	for y := h.Cursor.Y; y >= 0; y-- {
		indent, ok := pythonBlockIndent(h.Buf.Line(y), pythonDefRegex)
		if !ok {
			continue
		}
		start, end := h.pythonBlockBounds(y, indent)
		if !h.Cursor.Loc.LessThan(start) && h.Cursor.Loc.LessThan(end) {
			return start, end, true
		}
	}

	return buffer.Loc{}, buffer.Loc{}, false
}

func (h *BufPane) pythonClassObjectRange() (buffer.Loc, buffer.Loc, bool) {
	if start, end, ok := h.pythonBlockRangeForDecorator(pythonClassRegex); ok {
		return start, end, true
	}

	for y := h.Cursor.Y; y >= 0; y-- {
		indent, ok := pythonBlockIndent(h.Buf.Line(y), pythonClassRegex)
		if !ok {
			continue
		}
		start, end := h.pythonBlockBounds(y, indent)
		if !h.Cursor.Loc.LessThan(start) && h.Cursor.Loc.LessThan(end) {
			return start, end, true
		}
	}

	return buffer.Loc{}, buffer.Loc{}, false
}

func (h *BufPane) pythonBlockRangeForDecorator(pattern *regexp.Regexp) (buffer.Loc, buffer.Loc, bool) {
	line := h.Buf.Line(h.Cursor.Y)
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "@") {
		return buffer.Loc{}, buffer.Loc{}, false
	}

	indent := leadingIndentWidth(line)
	for y := h.Cursor.Y + 1; y < h.Buf.LinesNum(); y++ {
		next := h.Buf.Line(y)
		if strings.TrimSpace(next) == "" {
			continue
		}
		nextIndent, ok := pythonBlockIndent(next, pattern)
		if ok && nextIndent == indent {
			start, end := h.pythonBlockBounds(y, nextIndent)
			if h.Cursor.Loc.Y >= start.Y && h.Cursor.Loc.LessThan(end) {
				return start, end, true
			}
		}
		break
	}

	return buffer.Loc{}, buffer.Loc{}, false
}

func (h *BufPane) pythonBlockBounds(defY, indent int) (buffer.Loc, buffer.Loc) {
	startY := defY
	for startY > 0 {
		line := h.Buf.Line(startY - 1)
		if strings.TrimSpace(line) == "" {
			break
		}
		if strings.HasPrefix(strings.TrimSpace(line), "@") && leadingIndentWidth(line) == indent {
			startY--
			continue
		}
		break
	}

	for y := defY + 1; y < h.Buf.LinesNum(); y++ {
		line := h.Buf.Line(y)
		if strings.TrimSpace(line) == "" {
			continue
		}
		if leadingIndentWidth(line) <= indent {
			return buffer.Loc{X: 0, Y: startY}, buffer.Loc{X: 0, Y: y}
		}
	}

	return buffer.Loc{X: 0, Y: startY}, h.Buf.End()
}

func pythonBlockIndent(line string, pattern *regexp.Regexp) (int, bool) {
	match := pattern.FindStringSubmatch(line)
	if match == nil {
		return 0, false
	}
	return leadingIndentWidth(match[1]), true
}

func leadingIndentWidth(line string) int {
	count := 0
	for _, r := range line {
		if r != ' ' && r != '\t' {
			break
		}
		count++
	}
	return count
}

func isEscapedRune(line []rune, idx int) bool {
	escapes := 0
	for i := idx - 1; i >= 0 && line[i] == '\\'; i-- {
		escapes++
	}
	return escapes%2 == 1
}

func (h *BufPane) locToRuneOffset(loc buffer.Loc) int {
	offset := 0
	for y := 0; y < loc.Y; y++ {
		offset += util.CharacterCountInString(h.Buf.Line(y)) + 1
	}
	return offset + loc.X
}

func (h *BufPane) runeOffsetToLoc(offset int) buffer.Loc {
	if offset <= 0 {
		return h.Buf.Start()
	}

	remaining := offset
	for y := 0; y < h.Buf.LinesNum(); y++ {
		lineLen := util.CharacterCountInString(h.Buf.Line(y))
		if remaining <= lineLen {
			return buffer.Loc{X: remaining, Y: y}
		}
		remaining -= lineLen + 1
	}

	return h.Buf.End()
}

func describeTextObject(target textObjectTarget) string {
	switch target {
	case textObjectWord:
		return "word"
	case textObjectFunction:
		return "function"
	case textObjectClass:
		return "class"
	case textObjectDoubleQuotes:
		return "double quotes"
	case textObjectSingleQuotes:
		return "single quotes"
	case textObjectParens:
		return "parentheses"
	case textObjectBrackets:
		return "brackets"
	case textObjectParagraph:
		return "paragraph"
	default:
		return "text object"
	}
}

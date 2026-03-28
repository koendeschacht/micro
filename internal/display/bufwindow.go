package display

import (
	"strconv"
	"strings"

	runewidth "github.com/mattn/go-runewidth"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/util"
	"github.com/micro-editor/tcell/v2"
)

// The BufWindow provides a way of displaying a certain section of a buffer.
type BufWindow struct {
	*View

	// Buffer being shown in this window
	Buf *buffer.Buffer

	active bool

	sline *StatusLine

	bufWidth         int
	bufHeight        int
	gutterOffset     int
	hasMessage       bool
	maxLineNumLength int
	drawDivider      bool
}

// NewBufWindow creates a new window at a location in the screen with a width and height
func NewBufWindow(x, y, width, height int, buf *buffer.Buffer) *BufWindow {
	w := new(BufWindow)
	w.View = new(View)
	w.X, w.Y, w.Width, w.Height = x, y, width, height
	w.SetBuffer(buf)
	w.active = true

	w.sline = NewStatusLine(w)

	return w
}

// SetBuffer sets this window's buffer.
func (w *BufWindow) SetBuffer(b *buffer.Buffer) {
	w.Buf = b
	b.OptionCallback = func(option string, nativeValue any) {
		if option == "softwrap" {
			if nativeValue.(bool) {
				w.StartCol = 0
			} else {
				w.StartLine.Row = 0
			}
		}

		if option == "softwrap" || option == "wordwrap" {
			w.Relocate()
			for _, c := range w.Buf.GetCursors() {
				c.LastWrappedVisualX = c.GetVisualX(true)
			}
		}

		if option == "diffgutter" || option == "ruler" || option == "scrollbar" ||
			option == "statusline" {
			w.updateDisplayInfo()
			w.Relocate()
		}
	}
	b.GetVisualX = func(loc buffer.Loc) int {
		return w.VLocFromLoc(loc).VisualX
	}
}

// GetView gets the view.
func (w *BufWindow) GetView() *View {
	return w.View
}

// SetView sets the view.
func (w *BufWindow) SetView(view *View) {
	w.View = view
}

// Resize resizes this window.
func (w *BufWindow) Resize(width, height int) {
	w.Width, w.Height = width, height
	w.updateDisplayInfo()

	w.Relocate()
}

// SetActive marks the window as active.
func (w *BufWindow) SetActive(b bool) {
	w.active = b
}

// IsActive returns true if this window is active.
func (w *BufWindow) IsActive() bool {
	return w.active
}

// BufView returns the width, height and x,y location of the actual buffer.
// It is not exactly the same as the whole window which also contains gutter,
// ruler, scrollbar and statusline.
func (w *BufWindow) BufView() View {
	return View{
		X:         w.X + w.gutterOffset,
		Y:         w.Y,
		Width:     w.bufWidth,
		Height:    w.bufHeight,
		StartLine: w.StartLine,
		StartCol:  w.StartCol,
	}
}

func (w *BufWindow) updateDisplayInfo() {
	b := w.Buf

	w.drawDivider = false
	if !b.Settings["statusline"].(bool) {
		_, h := screen.Screen.Size()
		infoY := h
		if config.GetGlobalOption("infobar").(bool) {
			infoY--
		}
		if w.Y+w.Height != infoY {
			w.drawDivider = true
		}
	}

	w.bufHeight = w.Height
	if b.Settings["statusline"].(bool) || w.drawDivider {
		w.bufHeight--
	}

	scrollbarWidth := 0
	if w.Buf.Settings["scrollbar"].(bool) && w.Buf.LinesNum() > w.Height && w.Width > 0 {
		scrollbarWidth = 1
	}

	w.hasMessage = len(b.Messages) > 0

	// We need to know the string length of the largest line number
	// so we can pad appropriately when displaying line numbers
	w.maxLineNumLength = len(strconv.Itoa(b.LinesNum()))

	w.gutterOffset = 0
	if w.hasMessage {
		w.gutterOffset += 2
	}
	if b.Settings["diffgutter"].(bool) {
		w.gutterOffset++
	}
	if b.Settings["ruler"].(bool) {
		w.gutterOffset += w.maxLineNumLength + 1
	}

	if w.gutterOffset > w.Width-scrollbarWidth {
		w.gutterOffset = w.Width - scrollbarWidth
	}

	prevBufWidth := w.bufWidth
	w.bufWidth = w.Width - w.gutterOffset - scrollbarWidth

	if w.bufWidth != prevBufWidth && w.Buf.Settings["softwrap"].(bool) {
		for _, c := range w.Buf.GetCursors() {
			c.LastWrappedVisualX = c.GetVisualX(true)
		}
	}
}

func (w *BufWindow) getStartInfo(n, lineN int) ([]byte, int, int, *tcell.Style) {
	tabsize := util.IntOpt(w.Buf.Settings["tabsize"])
	width := 0
	bloc := buffer.Loc{0, lineN}
	b := w.Buf.LineBytes(lineN)
	curStyle := config.DefStyle
	var s *tcell.Style
	for len(b) > 0 {
		r, _, size := util.DecodeCharacter(b)

		curStyle, found := w.getStyle(curStyle, bloc)
		if found {
			s = &curStyle
		}

		w := 0
		switch r {
		case '\t':
			ts := tabsize - (width % tabsize)
			w = ts
		default:
			w = runewidth.RuneWidth(r)
		}
		if width+w > n {
			return b, n - width, bloc.X, s
		}
		width += w
		b = b[size:]
		bloc.X++
	}
	return b, n - width, bloc.X, s
}

// Clear resets all cells in this window to the default style
func (w *BufWindow) Clear() {
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			screen.SetContent(w.X+x, w.Y+y, ' ', nil, config.DefStyle)
		}
	}
}

// Relocate moves the view window so that the cursor is in view
// This is useful if the user has scrolled far away, and then starts typing
// Returns true if the window location is moved
func (w *BufWindow) Relocate() bool {
	b := w.Buf
	height := w.bufHeight
	ret := false
	activeC := w.Buf.GetActiveCursor()
	scrollmargin := int(b.Settings["scrollmargin"].(float64))

	c := w.SLocFromLoc(activeC.Loc)
	bStart := SLoc{0, 0}
	bEnd := w.SLocFromLoc(b.End())

	if c.LessThan(w.Scroll(w.StartLine, scrollmargin)) && c.GreaterThan(w.Scroll(bStart, scrollmargin-1)) {
		w.StartLine = w.Scroll(c, -scrollmargin)
		ret = true
	} else if c.LessThan(w.StartLine) {
		w.StartLine = c
		ret = true
	}
	if c.GreaterThan(w.Scroll(w.StartLine, height-1-scrollmargin)) && c.LessEqual(w.Scroll(bEnd, -scrollmargin)) {
		w.StartLine = w.Scroll(c, -height+1+scrollmargin)
		ret = true
	} else if c.GreaterThan(w.Scroll(bEnd, -scrollmargin)) && c.GreaterThan(w.Scroll(w.StartLine, height-1)) {
		w.StartLine = w.Scroll(bEnd, -height+1)
		ret = true
	}

	// horizontal relocation (scrolling)
	if !b.Settings["softwrap"].(bool) {
		cx := activeC.GetVisualX(false)
		rw := runewidth.RuneWidth(activeC.RuneUnder(activeC.X))
		if rw == 0 {
			rw = 1 // tab or newline
		}

		if cx < w.StartCol {
			w.StartCol = cx
			ret = true
		}
		if cx+rw > w.StartCol+w.bufWidth {
			w.StartCol = cx - w.bufWidth + rw
			ret = true
		}
	}
	return ret
}

// LocFromVisual takes a visual location (x and y position) and returns the
// position in the buffer corresponding to the visual location
// If the requested position does not correspond to a buffer location it returns
// the nearest position
func (w *BufWindow) LocFromVisual(svloc buffer.Loc) buffer.Loc {
	vx := svloc.X - w.X - w.gutterOffset
	if vx < 0 {
		vx = 0
	}
	vloc := VLoc{
		SLoc:    w.Scroll(w.StartLine, svloc.Y-w.Y),
		VisualX: vx + w.StartCol,
	}
	return w.LocFromVLoc(vloc)
}

func (w *BufWindow) drawGutter(vloc *buffer.Loc, bloc *buffer.Loc) {
	char := ' '
	s := config.DefStyle
	for _, m := range w.Buf.Messages {
		if m.Start.Y == bloc.Y || m.End.Y == bloc.Y {
			s = m.Style()
			char = '>'
			break
		}
	}
	for i := 0; i < 2 && vloc.X < w.gutterOffset; i++ {
		screen.SetContent(w.X+vloc.X, w.Y+vloc.Y, char, nil, s)
		vloc.X++
	}
}

func (w *BufWindow) drawDiffGutter(backgroundStyle tcell.Style, softwrapped bool, vloc *buffer.Loc, bloc *buffer.Loc) {
	if vloc.X >= w.gutterOffset {
		return
	}

	symbol := ' '
	styleName := ""

	switch w.Buf.DiffStatus(bloc.Y) {
	case buffer.DSAdded:
		symbol = '\u258C' // Left half block
		styleName = "diff-added"
	case buffer.DSModified:
		symbol = '\u258C' // Left half block
		styleName = "diff-modified"
	case buffer.DSDeletedAbove:
		if !softwrapped {
			symbol = '\u2594' // Upper one eighth block
			styleName = "diff-deleted"
		}
	}

	style := backgroundStyle
	if s, ok := config.Colorscheme[styleName]; ok {
		foreground, _, _ := s.Decompose()
		style = style.Foreground(foreground)
	}

	screen.SetContent(w.X+vloc.X, w.Y+vloc.Y, symbol, nil, style)
	vloc.X++
}

func (w *BufWindow) drawLineNum(lineNumStyle tcell.Style, softwrapped bool, vloc *buffer.Loc, bloc *buffer.Loc) {
	cursorLine := w.Buf.GetActiveCursor().Loc.Y
	var lineInt int
	if w.Buf.Settings["relativeruler"] == false || cursorLine == bloc.Y {
		lineInt = bloc.Y + 1
	} else {
		lineInt = bloc.Y - cursorLine
	}
	lineNum := []rune(strconv.Itoa(util.Abs(lineInt)))

	// Write the spaces before the line number if necessary
	for i := 0; i < w.maxLineNumLength-len(lineNum) && vloc.X < w.gutterOffset; i++ {
		screen.SetContent(w.X+vloc.X, w.Y+vloc.Y, ' ', nil, lineNumStyle)
		vloc.X++
	}
	// Write the actual line number
	for i := 0; i < len(lineNum) && vloc.X < w.gutterOffset; i++ {
		if softwrapped || (w.bufWidth == 0 && w.Buf.Settings["softwrap"] == true) {
			screen.SetContent(w.X+vloc.X, w.Y+vloc.Y, ' ', nil, lineNumStyle)
		} else {
			screen.SetContent(w.X+vloc.X, w.Y+vloc.Y, lineNum[i], nil, lineNumStyle)
		}
		vloc.X++
	}

	// Write the extra space
	if vloc.X < w.gutterOffset {
		screen.SetContent(w.X+vloc.X, w.Y+vloc.Y, ' ', nil, lineNumStyle)
		vloc.X++
	}
}

// getStyle returns the highlight style for the given character position
// If there is no change to the current highlight style it just returns that
func (w *BufWindow) getStyle(style tcell.Style, bloc buffer.Loc) (tcell.Style, bool) {
	if group, ok := w.Buf.Match(bloc.Y)[bloc.X]; ok {
		s := config.GetColor(group.String())
		return s, true
	}
	return style, false
}

func (w *BufWindow) showCursor(x, y int, main bool) {
	if w.active {
		if main {
			screen.ShowCursor(x, y)
		} else {
			screen.ShowFakeCursorMulti(x, y)
		}
	}
}

func trimToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	curWidth := 0
	var out []rune
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if curWidth+rw > width {
			break
		}
		out = append(out, r)
		curWidth += rw
	}
	return string(out)
}

func popupMaxTextWidth(lines []string) int {
	width := 1
	for _, line := range lines {
		width = util.Max(width, runewidth.StringWidth(line))
	}
	return width
}

func drawPopupLines(x, y, boxWidth, padLeft int, lines []string, rowStyle func(int) tcell.Style, textStyle func(int) tcell.Style) {
	for i, line := range lines {
		style := rowStyle(i)
		for dx := 0; dx < boxWidth; dx++ {
			screen.SetContent(x+dx, y+i, ' ', nil, style)
		}

		text := trimToWidth(line, util.Max(0, boxWidth-padLeft))
		textX := x + padLeft
		for _, r := range text {
			screen.SetContent(textX, y+i, r, nil, textStyle(i))
			textX += runewidth.RuneWidth(r)
		}
	}
}

func (w *BufWindow) clampPopupX(anchor buffer.Loc, boxWidth int) int {
	bufX := w.X + w.gutterOffset
	popupX := bufX + anchor.X - w.StartCol
	maxPopupX := bufX + w.bufWidth - boxWidth
	return util.Clamp(popupX, bufX, maxPopupX)
}

func wrapTextToWidth(text string, width int) []string {
	if width <= 0 {
		return nil
	}

	text = strings.ReplaceAll(text, "\t", "    ")
	rawLines := strings.Split(text, "\n")
	wrapped := make([]string, 0, len(rawLines))

	for _, rawLine := range rawLines {
		if rawLine == "" {
			wrapped = append(wrapped, "")
			continue
		}

		remaining := rawLine
		for len(remaining) > 0 {
			if runewidth.StringWidth(remaining) <= width {
				wrapped = append(wrapped, remaining)
				break
			}

			curWidth := 0
			splitByte := 0
			lastSpaceByte := -1
			for idx, r := range remaining {
				rw := runewidth.RuneWidth(r)
				if rw <= 0 {
					rw = 1
				}
				if curWidth+rw > width {
					break
				}
				curWidth += rw
				splitByte = idx + len(string(r))
				if r == ' ' {
					lastSpaceByte = idx
				}
			}

			if splitByte == 0 {
				_, _, size := util.DecodeCharacterInString(remaining)
				splitByte = size
			}

			lineEnd := splitByte
			nextStart := splitByte
			if lastSpaceByte >= 0 {
				lineEnd = lastSpaceByte
				nextStart = lastSpaceByte + 1
			}

			line := strings.TrimRight(remaining[:lineEnd], " ")
			if line == "" {
				line = trimToWidth(remaining[:splitByte], width)
				nextStart = splitByte
			}
			wrapped = append(wrapped, line)

			remaining = strings.TrimLeft(remaining[nextStart:], " ")
		}
	}

	if len(wrapped) == 0 {
		return []string{""}
	}
	return wrapped
}

func messageContainsCursor(m *buffer.Message, loc buffer.Loc) bool {
	start, end := m.Start, m.End
	if end.LessThan(start) {
		start, end = end, start
	}
	if start == end {
		return loc.Y == start.Y
	}
	return !loc.LessThan(start) && loc.LessThan(end)
}

func messageContainsLine(m *buffer.Message, line int) bool {
	startY := util.Min(m.Start.Y, m.End.Y)
	endY := util.Max(m.Start.Y, m.End.Y)
	return line >= startY && line <= endY
}

func messagePriority(m *buffer.Message) int {
	switch m.Kind {
	case buffer.MTError:
		return 0
	case buffer.MTWarning:
		return 1
	default:
		return 2
	}
}

func messageSpan(m *buffer.Message, b *buffer.Buffer) int {
	return m.Start.Diff(m.End, b)
}

func (w *BufWindow) currentDiagnosticMessage() *buffer.Message {
	if !w.active {
		return nil
	}

	cursor := w.Buf.GetActiveCursor().Loc
	var best *buffer.Message
	bestMatchRank := 3
	bestPriority := 3
	bestSpan := 0

	for _, m := range w.Buf.Messages {
		if m.Owner != "lsp" || m.Msg == "" {
			continue
		}

		matchRank := 3
		switch {
		case messageContainsCursor(m, cursor):
			matchRank = 0
		case m.Start.Y == cursor.Y:
			matchRank = 1
		case messageContainsLine(m, cursor.Y):
			matchRank = 2
		default:
			continue
		}

		priority := messagePriority(m)
		span := messageSpan(m, w.Buf)
		if best == nil || matchRank < bestMatchRank ||
			(matchRank == bestMatchRank && priority < bestPriority) ||
			(matchRank == bestMatchRank && priority == bestPriority && span < bestSpan) {
			best = m
			bestMatchRank = matchRank
			bestPriority = priority
			bestSpan = span
		}
	}

	return best
}

func (w *BufWindow) displayDiagnosticPopup() {
	m := w.currentDiagnosticMessage()
	if m == nil || w.bufWidth <= 0 || w.bufHeight <= 0 {
		return
	}

	bufY := w.Y
	anchor := w.VLocFromLoc(m.Start)
	anchorY := bufY + w.Diff(w.StartLine, anchor.SLoc)
	if anchorY < bufY || anchorY >= bufY+w.bufHeight {
		return
	}

	const popupPadX = 1
	const popupPadY = 0
	maxTextWidth := util.Min(w.bufWidth-2-(popupPadX*2), 80)
	if maxTextWidth <= 0 {
		return
	}

	lines := wrapTextToWidth(m.Msg, maxTextWidth)
	if len(lines) == 0 {
		return
	}

	maxPopupHeight := w.bufHeight
	if maxPopupHeight < 3 {
		return
	}
	maxContentLines := maxPopupHeight - 2 - (popupPadY * 2)
	if maxContentLines <= 0 {
		return
	}
	if len(lines) > maxContentLines {
		lines = lines[:maxContentLines]
		if maxContentLines > 0 {
			last := trimToWidth(lines[maxContentLines-1], util.Max(0, maxTextWidth-1))
			lines[maxContentLines-1] = last + "…"
		}
	}

	textWidth := popupMaxTextWidth(lines)
	boxWidth := textWidth + (popupPadX * 2)
	boxHeight := len(lines) + (popupPadY * 2)
	if boxWidth > w.bufWidth {
		boxWidth = w.bufWidth
	}

	popupX := w.clampPopupX(buffer.Loc{X: anchor.VisualX}, boxWidth)

	popupY := anchorY + 1
	maxPopupY := bufY + w.bufHeight - boxHeight
	popupY = util.Clamp(popupY, bufY, maxPopupY)

	popupStyle := config.DefStyle.Reverse(true)
	if style, ok := config.Colorscheme["diagnostic.popup"]; ok {
		popupStyle = style
	} else if style, ok := config.Colorscheme["autocomplete.popup"]; ok {
		popupStyle = style
	}

	textStyle := popupStyle
	switch m.Kind {
	case buffer.MTError:
		if style, ok := config.Colorscheme["diagnostic.error"]; ok {
			textStyle = style
		}
	case buffer.MTWarning:
		if style, ok := config.Colorscheme["diagnostic.warning"]; ok {
			textStyle = style
		}
	default:
		if style, ok := config.Colorscheme["diagnostic.info"]; ok {
			textStyle = style
		}
	}

	drawPopupLines(popupX, popupY+popupPadY, boxWidth, popupPadX, lines,
		func(int) tcell.Style { return popupStyle },
		func(int) tcell.Style { return textStyle },
	)
}

func (w *BufWindow) displayCompletionPopup() {
	b := w.Buf
	if !w.active || !b.CompletionMenu || len(b.Suggestions) == 0 {
		return
	}

	bufX := w.X + w.gutterOffset
	bufY := w.Y
	anchor := w.VLocFromLoc(b.CompletionStart)
	cursor := w.VLocFromLoc(b.GetActiveCursor().Loc)
	popupX := w.clampPopupX(buffer.Loc{X: anchor.VisualX}, 0)
	popupY := bufY + w.Diff(w.StartLine, cursor.SLoc) + 1
	if popupY < bufY || popupY >= w.Y+w.Height {
		return
	}
	availableWidth := bufX + w.bufWidth - popupX
	availableHeight := w.Y + w.Height - popupY
	if availableWidth < 4 || availableHeight < 3 {
		return
	}

	const maxPopupItems = 8
	visibleItems := util.Min(len(b.Suggestions), util.Min(maxPopupItems, availableHeight-2))
	if visibleItems <= 0 {
		return
	}

	startIdx := 0
	if b.CurSuggestion >= visibleItems {
		startIdx = b.CurSuggestion - visibleItems + 1
	}
	if startIdx+visibleItems > len(b.Suggestions) {
		startIdx = len(b.Suggestions) - visibleItems
	}
	if startIdx < 0 {
		startIdx = 0
	}

	textWidth := popupMaxTextWidth(b.Suggestions)
	const popupPadLeft = 0
	const popupPadRight = 0
	textWidth = util.Min(textWidth, availableWidth-(popupPadLeft+popupPadRight)-2)
	if textWidth <= 0 {
		return
	}

	popupStyle := config.DefStyle.Reverse(true)
	if style, ok := config.Colorscheme["autocomplete.popup"]; ok {
		popupStyle = style
	} else if style, ok := config.Colorscheme["statusline.suggestions"]; ok {
		popupStyle = style
	} else if style, ok := config.Colorscheme["statusline"]; ok {
		popupStyle = style
	}
	selectedStyle := popupStyle.Foreground(tcell.GetColor("#1F2330")).Background(tcell.GetColor("#5FC6C6"))
	if style, ok := config.Colorscheme["autocomplete.selected"]; ok {
		selectedStyle = style
	}
	boxWidth := textWidth + popupPadLeft + popupPadRight + 2
	popupX = w.clampPopupX(buffer.Loc{X: anchor.VisualX}, boxWidth)

	lines := make([]string, visibleItems)
	for i := 0; i < visibleItems; i++ {
		idx := startIdx + i
		lines[i] = b.Suggestions[idx]
	}

	drawPopupLines(popupX, popupY, boxWidth, 1+popupPadLeft, lines,
		func(i int) tcell.Style {
			idx := startIdx + i
			if idx == b.CurSuggestion {
				return selectedStyle
			}
			return popupStyle
		},
		func(i int) tcell.Style {
			idx := startIdx + i
			if idx == b.CurSuggestion {
				return selectedStyle
			}
			return popupStyle
		},
	)
}

// displayBuffer draws the buffer being shown in this window on the screen.Screen
func (w *BufWindow) displayBuffer() {
	b := w.Buf

	if w.Height <= 0 || w.Width <= 0 {
		return
	}

	maxWidth := w.gutterOffset + w.bufWidth

	if b.ModifiedThisFrame {
		if b.Settings["diffgutter"].(bool) {
			b.UpdateDiff()
		}
		b.ModifiedThisFrame = false
	}

	var matchingBraces []buffer.Loc
	// bracePairs is defined in buffer.go
	if b.Settings["matchbrace"].(bool) {
		for _, c := range b.GetCursors() {
			if c.HasSelection() {
				continue
			}

			mb, left, found := b.FindMatchingBrace(c.Loc)
			if found {
				matchingBraces = append(matchingBraces, mb)
				if !left {
					if b.Settings["matchbracestyle"].(string) != "highlight" {
						matchingBraces = append(matchingBraces, c.Loc)
					}
				} else {
					matchingBraces = append(matchingBraces, c.Loc.Move(-1, b))
				}
			}
		}
	}

	lineNumStyle := config.DefStyle
	if style, ok := config.Colorscheme["line-number"]; ok {
		lineNumStyle = style
	}
	curNumStyle := config.DefStyle
	if style, ok := config.Colorscheme["current-line-number"]; ok {
		if !b.Settings["cursorline"].(bool) {
			curNumStyle = lineNumStyle
		} else {
			curNumStyle = style
		}
	}

	softwrap := b.Settings["softwrap"].(bool)
	wordwrap := softwrap && b.Settings["wordwrap"].(bool)

	tabsize := util.IntOpt(b.Settings["tabsize"])
	colorcolumn := util.IntOpt(b.Settings["colorcolumn"])

	// this represents the current draw position
	// within the current window
	vloc := buffer.Loc{X: 0, Y: 0}
	if softwrap {
		// the start line may be partially out of the current window
		vloc.Y = -w.StartLine.Row
	}

	// this represents the current draw position in the buffer (char positions)
	bloc := buffer.Loc{X: -1, Y: w.StartLine.Line}

	cursors := b.GetCursors()

	curStyle := config.DefStyle

	// Parse showchars which is in the format of key1=val1,key2=val2,...
	spacechars := " "
	tabchars := b.Settings["indentchar"].(string)
	var indentspacechars string
	var indenttabchars string
	for _, entry := range strings.Split(b.Settings["showchars"].(string), ",") {
		split := strings.SplitN(entry, "=", 2)
		if len(split) < 2 {
			continue
		}
		key, val := split[0], split[1]
		switch key {
		case "space":
			spacechars = val
		case "tab":
			tabchars = val
		case "ispace":
			indentspacechars = val
		case "itab":
			indenttabchars = val
		}
	}

	for ; vloc.Y < w.bufHeight; vloc.Y++ {
		vloc.X = 0

		currentLine := false
		for _, c := range cursors {
			if !c.HasSelection() && bloc.Y == c.Y && w.active {
				currentLine = true
				break
			}
		}

		s := lineNumStyle
		if currentLine {
			s = curNumStyle
		}

		if vloc.Y >= 0 {
			if w.hasMessage {
				w.drawGutter(&vloc, &bloc)
			}

			if b.Settings["diffgutter"].(bool) {
				w.drawDiffGutter(s, false, &vloc, &bloc)
			}

			if b.Settings["ruler"].(bool) {
				w.drawLineNum(s, false, &vloc, &bloc)
			}
		} else {
			vloc.X = w.gutterOffset
		}

		bline := b.LineBytes(bloc.Y)
		blineLen := util.CharacterCount(bline)

		leadingwsEnd := len(util.GetLeadingWhitespace(bline))
		trailingwsStart := blineLen - util.CharacterCount(util.GetTrailingWhitespace(bline))

		line, nColsBeforeStart, bslice, startStyle := w.getStartInfo(w.StartCol, bloc.Y)
		if startStyle != nil {
			curStyle = *startStyle
		}
		bloc.X = bslice

		// returns the rune to be drawn, style of it and if the bg should be preserved
		getRuneStyle := func(r rune, style tcell.Style, showoffset int, linex int, isplaceholder bool) (rune, tcell.Style, bool) {
			if nColsBeforeStart > 0 || vloc.Y < 0 || isplaceholder {
				return r, style, false
			}

			for _, mb := range matchingBraces {
				if mb.X == bloc.X && mb.Y == bloc.Y {
					if b.Settings["matchbracestyle"].(string) == "highlight" {
						if s, ok := config.Colorscheme["match-brace"]; ok {
							return r, s, false
						} else {
							return r, style.Reverse(true), false
						}
					} else {
						return r, style.Underline(true), false
					}
				}
			}

			if r != '\t' && r != ' ' {
				return r, style, false
			}

			var indentrunes []rune
			switch r {
			case '\t':
				if bloc.X < leadingwsEnd && indenttabchars != "" {
					indentrunes = []rune(indenttabchars)
				} else {
					indentrunes = []rune(tabchars)
				}
			case ' ':
				if linex%tabsize == 0 && bloc.X < leadingwsEnd && indentspacechars != "" {
					indentrunes = []rune(indentspacechars)
				} else {
					indentrunes = []rune(spacechars)
				}
			}

			var drawrune rune
			if showoffset < len(indentrunes) {
				drawrune = indentrunes[showoffset]
			} else {
				// use space if no showchars or after we showed showchars
				drawrune = ' '
			}

			if s, ok := config.Colorscheme["indent-char"]; ok {
				fg, _, _ := s.Decompose()
				style = style.Foreground(fg)
			}

			preservebg := false
			if b.Settings["hltaberrors"].(bool) && bloc.X < leadingwsEnd {
				if s, ok := config.Colorscheme["tab-error"]; ok {
					if b.Settings["tabstospaces"].(bool) && r == '\t' {
						fg, _, _ := s.Decompose()
						style = style.Background(fg)
						preservebg = true
					} else if !b.Settings["tabstospaces"].(bool) && r == ' ' {
						fg, _, _ := s.Decompose()
						style = style.Background(fg)
						preservebg = true
					}
				}
			}

			if b.Settings["hltrailingws"].(bool) {
				if s, ok := config.Colorscheme["trailingws"]; ok {
					if bloc.X >= trailingwsStart && bloc.X < blineLen {
						hl := true
						for _, c := range cursors {
							if c.NewTrailingWsY == bloc.Y {
								hl = false
								break
							}
						}
						if hl {
							fg, _, _ := s.Decompose()
							style = style.Background(fg)
							preservebg = true
						}
					}
				}
			}

			return drawrune, style, preservebg
		}

		draw := func(r rune, combc []rune, style tcell.Style, highlight bool, showcursor bool, preservebg bool) {
			defer func() {
				if nColsBeforeStart <= 0 {
					vloc.X++
				}
				nColsBeforeStart--
			}()

			if nColsBeforeStart > 0 || vloc.Y < 0 {
				return
			}

			if highlight {
				if w.Buf.HighlightSearch && w.Buf.SearchMatch(bloc) {
					style = config.DefStyle.Reverse(true)
					if s, ok := config.Colorscheme["hlsearch"]; ok {
						style = s
					}
				}

				_, origBg, _ := style.Decompose()
				_, defBg, _ := config.DefStyle.Decompose()

				// syntax or hlsearch highlighting with non-default background takes precedence
				// over cursor-line and color-column
				if !preservebg && origBg != defBg {
					preservebg = true
				}

				for _, c := range cursors {
					if c.HasSelection() &&
						(bloc.GreaterEqual(c.CurSelection[0]) && bloc.LessThan(c.CurSelection[1]) ||
							bloc.LessThan(c.CurSelection[0]) && bloc.GreaterEqual(c.CurSelection[1])) {
						// The current character is selected
						style = config.DefStyle.Reverse(true)

						if s, ok := config.Colorscheme["selection"]; ok {
							style = s
						}
					}

					if b.Settings["cursorline"].(bool) && w.active && !preservebg &&
						!c.HasSelection() && c.Y == bloc.Y {
						if s, ok := config.Colorscheme["cursor-line"]; ok {
							fg, _, _ := s.Decompose()
							style = style.Background(fg)
						}
					}
				}

				for _, m := range b.Messages {
					if bloc.GreaterEqual(m.Start) && bloc.LessThan(m.End) ||
						bloc.LessThan(m.End) && bloc.GreaterEqual(m.Start) {
						style = style.Underline(true)
						break
					}
				}

				if s, ok := config.Colorscheme["color-column"]; ok {
					if colorcolumn != 0 && vloc.X-w.gutterOffset+w.StartCol == colorcolumn && !preservebg {
						fg, _, _ := s.Decompose()
						style = style.Background(fg)
					}
				}
			}

			screen.SetContent(w.X+vloc.X, w.Y+vloc.Y, r, combc, style)

			if showcursor {
				for _, c := range cursors {
					if c.X == bloc.X && c.Y == bloc.Y && !c.HasSelection() {
						w.showCursor(w.X+vloc.X, w.Y+vloc.Y, c.Num == 0)
					}
				}
			}
		}

		wrap := func() {
			vloc.X = 0

			if vloc.Y >= 0 {
				if w.hasMessage {
					w.drawGutter(&vloc, &bloc)
				}
				if b.Settings["diffgutter"].(bool) {
					w.drawDiffGutter(lineNumStyle, true, &vloc, &bloc)
				}

				// This will draw an empty line number because the current line is wrapped
				if b.Settings["ruler"].(bool) {
					w.drawLineNum(lineNumStyle, true, &vloc, &bloc)
				}
			} else {
				vloc.X = w.gutterOffset
			}
		}

		type glyph struct {
			r     rune
			combc []rune
			style tcell.Style
			width int
		}

		var word []glyph
		if wordwrap {
			word = make([]glyph, 0, w.bufWidth)
		} else {
			word = make([]glyph, 0, 1)
		}
		wordwidth := 0

		totalwidth := w.StartCol - nColsBeforeStart
		for len(line) > 0 && vloc.X < maxWidth {
			r, combc, size := util.DecodeCharacter(line)
			line = line[size:]

			loc := buffer.Loc{X: bloc.X + len(word), Y: bloc.Y}
			curStyle, _ = w.getStyle(curStyle, loc)

			width := 0

			linex := totalwidth
			switch r {
			case '\t':
				ts := tabsize - (totalwidth % tabsize)
				width = util.Min(ts, maxWidth-vloc.X)
				totalwidth += ts
			default:
				width = runewidth.RuneWidth(r)
				totalwidth += width
			}

			word = append(word, glyph{r, combc, curStyle, width})
			wordwidth += width

			// Collect a complete word to know its width.
			// If wordwrap is off, every single character is a complete "word".
			if wordwrap {
				if !util.IsWhitespace(r) && len(line) > 0 && wordwidth < w.bufWidth {
					continue
				}
			}

			// If a word (or just a wide rune) does not fit in the window
			if vloc.X+wordwidth > maxWidth && vloc.X > w.gutterOffset {
				for vloc.X < maxWidth {
					draw(' ', nil, config.DefStyle, false, false, true)
				}

				// We either stop or we wrap to draw the word in the next line
				if !softwrap {
					break
				} else {
					vloc.Y++
					if vloc.Y >= w.bufHeight {
						break
					}
					wrap()
				}
			}

			for _, r := range word {
				drawrune, drawstyle, preservebg := getRuneStyle(r.r, r.style, 0, linex, false)
				draw(drawrune, r.combc, drawstyle, true, true, preservebg)

				// Draw extra characters for tabs or wide runes
				for i := 1; i < r.width; i++ {
					if r.r == '\t' {
						drawrune, drawstyle, preservebg = getRuneStyle('\t', r.style, i, linex+i, false)
					} else {
						drawrune, drawstyle, preservebg = getRuneStyle(' ', r.style, i, linex+i, true)
					}
					draw(drawrune, nil, drawstyle, true, false, preservebg)
				}
				bloc.X++
			}

			word = word[:0]
			wordwidth = 0

			// If we reach the end of the window then we either stop or we wrap for softwrap
			if vloc.X >= maxWidth {
				if !softwrap {
					break
				} else {
					vloc.Y++
					if vloc.Y >= w.bufHeight {
						break
					}
					wrap()
				}
			}
		}

		style := config.DefStyle
		for _, c := range cursors {
			if b.Settings["cursorline"].(bool) && w.active &&
				!c.HasSelection() && c.Y == bloc.Y {
				if s, ok := config.Colorscheme["cursor-line"]; ok {
					fg, _, _ := s.Decompose()
					style = style.Background(fg)
				}
			}
		}
		ghostX := vloc.X
		ghostY := vloc.Y
		for i := vloc.X; i < maxWidth; i++ {
			curStyle := style
			if s, ok := config.Colorscheme["color-column"]; ok {
				if colorcolumn != 0 && i-w.gutterOffset+w.StartCol == colorcolumn {
					fg, _, _ := s.Decompose()
					curStyle = style.Background(fg)
				}
			}
			screen.SetContent(i+w.X, vloc.Y+w.Y, ' ', nil, curStyle)
		}

		if vloc.X != maxWidth {
			// Display newline within a selection
			drawrune, drawstyle, preservebg := getRuneStyle(' ', config.DefStyle, 0, totalwidth, true)
			draw(drawrune, nil, drawstyle, true, true, preservebg)
		}

		if b.GhostText != "" && b.GhostAt.Y == bloc.Y && b.GhostAt.X == bloc.X {
			ghostStyle := style.Dim(true)
			if s, ok := config.Colorscheme["autocomplete-ghost"]; ok {
				ghostStyle = s
			}
			x := ghostX
			visualWidth := totalwidth
			for _, r := range b.GhostText {
				width := runewidth.RuneWidth(r)
				drawRune := r
				if r == '\t' {
					width = tabsize - (visualWidth % tabsize)
					drawRune = ' '
				}
				if width <= 0 || x >= maxWidth {
					break
				}
				screen.SetContent(w.X+x, w.Y+ghostY, drawRune, nil, ghostStyle)
				x++
				for i := 1; i < width && x < maxWidth; i++ {
					screen.SetContent(w.X+x, w.Y+ghostY, ' ', nil, ghostStyle)
					x++
				}
				visualWidth += width
			}
			for _, c := range cursors {
				if c.X == b.GhostAt.X && c.Y == b.GhostAt.Y && !c.HasSelection() {
					w.showCursor(w.X+ghostX, w.Y+ghostY, c.Num == 0)
				}
			}
		}

		bloc.X = w.StartCol
		bloc.Y++
		if bloc.Y >= b.LinesNum() {
			break
		}
	}
}

func (w *BufWindow) displayStatusLine() {
	if w.Buf.Settings["statusline"].(bool) {
		w.sline.Display()
	} else if w.drawDivider {
		divchars := config.GetGlobalOption("divchars").(string)
		if util.CharacterCountInString(divchars) != 2 {
			divchars = "|-"
		}

		_, _, size := util.DecodeCharacterInString(divchars)
		divchar, combc, _ := util.DecodeCharacterInString(divchars[size:])

		dividerStyle := config.DefStyle
		if style, ok := config.Colorscheme["divider"]; ok {
			dividerStyle = style
		}

		divreverse := config.GetGlobalOption("divreverse").(bool)
		if divreverse {
			dividerStyle = dividerStyle.Reverse(true)
		}

		for x := w.X; x < w.X+w.Width; x++ {
			screen.SetContent(x, w.Y+w.Height-1, divchar, combc, dividerStyle)
		}
	}
}

func (w *BufWindow) displayScrollBar() {
	if w.Buf.Settings["scrollbar"].(bool) && w.Buf.LinesNum() > w.Height {
		scrollX := w.X + w.Width - 1
		barsize := int(float64(w.Height) / float64(w.Buf.LinesNum()) * float64(w.Height))
		if barsize < 1 {
			barsize = 1
		}
		barstart := w.Y + int(float64(w.StartLine.Line)/float64(w.Buf.LinesNum())*float64(w.Height))

		scrollBarStyle := config.DefStyle.Reverse(true)
		if style, ok := config.Colorscheme["scrollbar"]; ok {
			scrollBarStyle = style
		}

		scrollBarChar := config.GetGlobalOption("scrollbarchar").(string)
		if util.CharacterCountInString(scrollBarChar) != 1 {
			scrollBarChar = "|"
		}
		scrollBarRune := []rune(scrollBarChar)

		for y := barstart; y < util.Min(barstart+barsize, w.Y+w.bufHeight); y++ {
			screen.SetContent(scrollX, y, scrollBarRune[0], nil, scrollBarStyle)
		}
	}
}

// Display displays the buffer and the statusline
func (w *BufWindow) Display() {
	w.updateDisplayInfo()

	w.displayStatusLine()
	w.displayScrollBar()
	w.displayBuffer()
	w.displayDiagnosticPopup()
	w.displayCompletionPopup()
}

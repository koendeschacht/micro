package display

import (
	"strings"

	"github.com/gdamore/tcell/v3"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/info"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/util"
)

const infoPopupMinWidth = 24

type popupRect struct {
	X, Y           int
	Width, Height  int
	InnerX, InnerY int
	InnerW, InnerH int
	MessageY       int
	InputY         int
}

type InfoWindow struct {
	*info.InfoBuf
	*View

	hscroll int
}

func colorschemeSymbol(name string, fallback string) string {
	if symbol := config.GetColorschemeSymbolOrDefault(name); symbol != "" {
		return symbol
	}
	return fallback
}

func colorschemeStyle(name string, fallback tcell.Style) tcell.Style {
	if style, ok := config.Colorscheme[name]; ok {
		return style
	}
	return fallback
}

func (i *InfoWindow) defStyle() tcell.Style {
	defStyle := config.DefStyle
	if style, ok := config.Colorscheme["message"]; ok {
		defStyle = style
	}
	return defStyle
}

func (i *InfoWindow) popupStyle() tcell.Style {
	if style, ok := config.Colorscheme["message-popup"]; ok {
		return style
	}
	if style, ok := config.Colorscheme["diagnostic.popup"]; ok {
		return style
	}
	if style, ok := config.Colorscheme["autocomplete.popup"]; ok {
		return style
	}
	return config.DefStyle.Background(tcell.GetColor("#1F2330"))
}

func (i *InfoWindow) mergePopupStyle(style tcell.Style) tcell.Style {
	base := i.popupStyle()
	fg := style.GetForeground()
	if fg == tcell.ColorDefault {
		fg = base.GetForeground()
	}
	return base.Foreground(fg).
		Bold(style.HasBold()).
		Italic(style.HasItalic()).
		Underline(style.HasUnderline())
}

func (i *InfoWindow) keyMenuKeyStyle() tcell.Style {
	style := i.defStyle()
	accent := style.GetForeground()
	bold := false
	italic := false
	underline := false

	if accentStyle, ok := config.Colorscheme["statement"]; ok {
		accent = accentStyle.GetForeground()
		bold = accentStyle.HasBold()
		italic = accentStyle.HasItalic()
		underline = accentStyle.HasUnderline()
	} else if accentStyle, ok := config.Colorscheme["tabbar.active"]; ok {
		accent = accentStyle.GetForeground()
		bold = accentStyle.HasBold()
		italic = accentStyle.HasItalic()
		underline = accentStyle.HasUnderline()
	}

	return i.mergePopupStyle(style.Foreground(accent).Bold(bold).Italic(italic).Underline(underline))
}

func (i *InfoWindow) successStyle() tcell.Style {
	base := i.defStyle()
	return colorschemeStyle("success-message", base.Foreground(tcell.ColorGreen).Bold(true))
}

func (i *InfoWindow) errStyle() tcell.Style {
	base := i.defStyle()
	return colorschemeStyle("error-message", base.Foreground(tcell.ColorWhite).Bold(true))
}

func (i *InfoWindow) promptStyle() tcell.Style {
	base := i.defStyle()
	if style, ok := config.Colorscheme["prompt-message"]; ok {
		return style
	}
	if accent, ok := config.Colorscheme["statement"]; ok {
		return base.Foreground(accent.GetForeground()).Bold(true)
	}
	return base.Foreground(tcell.ColorBlue).Bold(true)
}

func (i *InfoWindow) currentMessageStyle() tcell.Style {
	if i.HasError {
		return i.errStyle()
	}
	if i.HasPrompt {
		return i.promptStyle()
	}
	if i.MsgKind == info.MsgSuccess {
		return i.successStyle()
	}
	return i.defStyle()
}

func (i *InfoWindow) popupBorderStyle() tcell.Style {
	base := i.popupStyle()
	name := "message-border"
	fallback := base.Foreground(tcell.ColorAqua).Bold(true)

	switch {
	case i.HasError:
		name = "error-message-border"
		fallback = base.Foreground(tcell.ColorRed).Bold(true)
	case i.HasPrompt:
		name = "prompt-message-border"
		fallback = base.Foreground(tcell.ColorBlue).Bold(true)
	case i.MsgKind == info.MsgSuccess:
		name = "success-message-border"
		fallback = base.Foreground(tcell.ColorGreen).Bold(true)
	}

	if style, ok := config.Colorscheme[name]; ok {
		return i.mergePopupStyle(style)
	}
	return fallback
}

func (i *InfoWindow) messagePrefix() string {
	if i.HasError {
		return colorschemeSymbol("error-message", "") + "  "
	}
	if i.HasPrompt {
		return colorschemeSymbol("prompt-message", "") + "  "
	}
	if i.MsgKind == info.MsgSuccess {
		return colorschemeSymbol("success-message", "") + "  "
	}
	return colorschemeSymbol("message", "") + "  "
}

func NewInfoWindow(b *info.InfoBuf) *InfoWindow {
	iw := new(InfoWindow)
	iw.InfoBuf = b
	iw.View = new(View)

	iw.Width, iw.Y = screen.Screen.Size()
	iw.Y--

	return iw
}

func (i *InfoWindow) Resize(w, h int) {
	i.Width = w
	i.Y = h
}

func (i *InfoWindow) SetBuffer(b *buffer.Buffer) {
	i.InfoBuf.Buffer = b
}

func (i *InfoWindow) Relocate() bool   { return false }
func (i *InfoWindow) GetView() *View   { return i.View }
func (i *InfoWindow) SetView(v *View)  {}
func (i *InfoWindow) SetActive(b bool) {}
func (i *InfoWindow) IsActive() bool   { return true }

func (i *InfoWindow) popupInnerWidthLimit() int {
	return util.Max(1, i.Width-2)
}

func keyMenuLayoutWidth(lines [][]info.KeyMenuEntry) int {
	width := 0
	for _, line := range lines {
		lineWidth := 0
		for idx, entry := range line {
			if idx > 0 {
				lineWidth += 2
			}
			lineWidth += keyMenuEntryWidth(entry)
		}
		width = util.Max(width, lineWidth)
	}
	return width
}

func (i *InfoWindow) keyMenuEntries() []info.KeyMenuEntry {
	if len(i.KeyMenu) > 0 {
		return i.KeyMenu
	}
	if info.RootKeyMenuEntries != nil {
		return info.RootKeyMenuEntries()
	}
	return nil
}

func (i *InfoWindow) showKeyMenu() bool {
	if i.HasPrompt || i.HasMessage || i.HasError {
		return false
	}
	return config.GetGlobalOption("keymenu").(bool) || len(i.KeyMenu) > 0
}

func (i *InfoWindow) keyMenuLayout(width int) [][]info.KeyMenuEntry {
	if !i.showKeyMenu() {
		return nil
	}
	return layoutKeyMenuEntries(i.keyMenuEntries(), width)
}

func (i *InfoWindow) KeyMenuLineCount() int {
	if !i.showKeyMenu() {
		return 0
	}
	return len(i.keyMenuLayout(i.popupInnerWidthLimit()))
}

func (i *InfoWindow) messageLines(innerW int) []string {
	if !i.HasPrompt && !i.HasMessage && !i.HasError {
		return nil
	}
	if i.HasPrompt {
		return wrapTextToWidth(i.messagePrefix()+i.Msg, innerW)
	}
	return wrapTextToWidth(i.messagePrefix()+i.Msg, innerW)
}

func (i *InfoWindow) hasInputLine() bool {
	return i.HasPrompt && !i.HasYN
}

func (i *InfoWindow) popupInnerWidth() int {
	maxInnerW := i.popupInnerWidthLimit()
	minInnerW := util.Min(maxInnerW, infoPopupMinWidth)
	innerW := 0

	if i.showKeyMenu() {
		innerW = util.Max(innerW, keyMenuLayoutWidth(i.keyMenuLayout(maxInnerW)))
	}

	if i.HasPrompt {
		for _, line := range strings.Split(i.messagePrefix()+i.Msg, "\n") {
			innerW = util.Max(innerW, runewidth.StringWidth(line))
		}
		if i.hasInputLine() {
			innerW = util.Max(innerW, runewidth.StringWidth(string(i.Buffer.LineBytes(0))))
		}
	} else if i.HasMessage || i.HasError {
		for _, line := range strings.Split(i.messagePrefix()+i.Msg, "\n") {
			innerW = util.Max(innerW, runewidth.StringWidth(line))
		}
	}

	if innerW == 0 {
		innerW = minInnerW
	}

	return util.Clamp(innerW, minInnerW, maxInnerW)
}

func (i *InfoWindow) showPopup() bool {
	if i.HasPrompt {
		return true
	}
	if !config.GlobalSettings["infobar"].(bool) {
		return false
	}
	return i.showKeyMenu() || i.HasMessage || i.HasError
}

func (i *InfoWindow) popupGeometry() (popupRect, [][]info.KeyMenuEntry, []string, bool) {
	i.ExpireMessage()
	if !i.showPopup() || i.Width < 3 || i.Y < 2 {
		return popupRect{}, nil, nil, false
	}

	innerW := i.popupInnerWidth()
	keyLayout := i.keyMenuLayout(innerW)
	messageLines := i.messageLines(innerW)
	inputRows := 0
	if i.hasInputLine() {
		inputRows = 1
	}
	contentRows := len(keyLayout) + len(messageLines) + inputRows
	maxContentRows := util.Max(1, i.Y-2)

	if contentRows == 0 {
		return popupRect{}, nil, nil, false
	}

	if contentRows > maxContentRows {
		overflow := contentRows - maxContentRows
		if overflow >= len(keyLayout) {
			keyRows := len(keyLayout)
			keyLayout = nil
			overflow -= keyRows
			if len(messageLines) == 0 {
				messageLines = nil
			} else if overflow > 0 && overflow < len(messageLines) {
				messageLines = messageLines[overflow:]
			} else if overflow >= len(messageLines) {
				messageLines = messageLines[len(messageLines)-1:]
			}
		} else {
			keyLayout = keyLayout[overflow:]
		}
		contentRows = len(keyLayout) + len(messageLines) + inputRows
	}

	boxW := util.Min(i.Width, innerW+2)
	messageY := i.Y - (contentRows + 1) + len(keyLayout)
	inputY := -1
	if inputRows > 0 {
		inputY = messageY + len(messageLines)
	}
	rect := popupRect{
		X:        0,
		Y:        i.Y - (contentRows + 2),
		Width:    boxW,
		Height:   contentRows + 2,
		InnerX:   1,
		InnerY:   i.Y - (contentRows + 1),
		InnerW:   boxW - 2,
		InnerH:   contentRows,
		MessageY: messageY,
		InputY:   inputY,
	}
	return rect, keyLayout, messageLines, true
}

func (i *InfoWindow) LocFromVisual(vloc buffer.Loc) buffer.Loc {
	c := i.Buffer.GetActiveCursor()
	l := i.Buffer.LineBytes(0)
	view := i.BufView()
	return buffer.Loc{c.GetCharPosInLine(l, vloc.X-view.X), 0}
}

func (i *InfoWindow) BufView() View {
	rect, _, _, ok := i.popupGeometry()
	if !ok || !i.hasInputLine() {
		return View{X: 0, Y: i.Y, Width: i.Width, Height: 1, StartLine: SLoc{0, 0}, StartCol: 0}
	}
	return View{
		X:         rect.InnerX,
		Y:         rect.InputY,
		Width:     rect.InnerW,
		Height:    1,
		StartLine: SLoc{0, 0},
		StartCol:  0,
	}
}

func (i *InfoWindow) Scroll(s SLoc, n int) SLoc       { return s }
func (i *InfoWindow) Diff(s1, s2 SLoc) int            { return 0 }
func (i *InfoWindow) SLocFromLoc(loc buffer.Loc) SLoc { return SLoc{0, 0} }

func (i *InfoWindow) VLocFromLoc(loc buffer.Loc) VLoc {
	view := i.BufView()
	return VLoc{SLoc{0, 0}, view.X + loc.X}
}

func (i *InfoWindow) LocFromVLoc(vloc VLoc) buffer.Loc {
	view := i.BufView()
	return buffer.Loc{X: util.Max(0, vloc.VisualX-view.X), Y: 0}
}

func (i *InfoWindow) IsVirtualRow(s SLoc) bool { return false }

func (i *InfoWindow) Clear() {}

func (i *InfoWindow) displayBuffer(rect popupRect, y int, style tcell.Style) {
	b := i.Buffer
	line := b.LineBytes(0)
	activeC := b.GetActiveCursor()

	blocX := 0
	vlocX := rect.InnerX
	maxX := rect.InnerX + rect.InnerW

	tabsize := 4
	line, nColsBeforeStart, bslice := util.SliceVisualEnd(line, blocX, tabsize)
	blocX = bslice

	draw := func(r rune, combc []rune, drawStyle tcell.Style) {
		if nColsBeforeStart <= 0 && vlocX < maxX {
			bloc := buffer.Loc{X: blocX, Y: 0}
			if activeC.HasSelection() &&
				(bloc.GreaterEqual(activeC.CurSelection[0]) && bloc.LessThan(activeC.CurSelection[1]) ||
					bloc.LessThan(activeC.CurSelection[0]) && bloc.GreaterEqual(activeC.CurSelection[1])) {
				drawStyle = i.mergePopupStyle(i.defStyle().Reverse(true))
				if s, ok := config.Colorscheme["selection"]; ok {
					drawStyle = s
				}
			}

			screen.SetContent(vlocX, y, r, combc, drawStyle)
			vlocX += runewidth.RuneWidth(r)
		}
		nColsBeforeStart--
	}

	totalwidth := blocX - nColsBeforeStart
	for len(line) > 0 && vlocX < maxX {
		curVX := vlocX
		curBX := blocX
		r, combc, size := util.DecodeCharacter(line)

		width := 0
		switch r {
		case '\t':
			width = tabsize - (totalwidth % tabsize)
			for j := 0; j < width && vlocX < maxX; j++ {
				draw(' ', nil, style)
			}
		default:
			width = runewidth.RuneWidth(r)
			draw(r, combc, style)
		}

		blocX++
		line = line[size:]

		if activeC.X == curBX && curVX < maxX {
			screen.ShowCursor(curVX, y)
		}
		totalwidth += width
	}
	if activeC.X == blocX && vlocX < maxX {
		screen.ShowCursor(vlocX, y)
	}
}

func (i *InfoWindow) drawPopupFrame(rect popupRect, baseStyle tcell.Style, borderStyle tcell.Style) {
	for y := 0; y < rect.Height; y++ {
		for x := 0; x < rect.Width; x++ {
			screen.SetContent(rect.X+x, rect.Y+y, ' ', nil, baseStyle)
		}
	}

	for x := 1; x < rect.Width-1; x++ {
		screen.SetContent(rect.X+x, rect.Y, '─', nil, borderStyle)
		screen.SetContent(rect.X+x, rect.Y+rect.Height-1, '─', nil, borderStyle)
	}
	for y := 1; y < rect.Height-1; y++ {
		screen.SetContent(rect.X, rect.Y+y, '│', nil, borderStyle)
		screen.SetContent(rect.X+rect.Width-1, rect.Y+y, '│', nil, borderStyle)
	}
	screen.SetContent(rect.X, rect.Y, '╭', nil, borderStyle)
	screen.SetContent(rect.X+rect.Width-1, rect.Y, '╮', nil, borderStyle)
	screen.SetContent(rect.X, rect.Y+rect.Height-1, '╰', nil, borderStyle)
	screen.SetContent(rect.X+rect.Width-1, rect.Y+rect.Height-1, '╯', nil, borderStyle)
}

func (i *InfoWindow) displayKeyMenu(rect popupRect, lines [][]info.KeyMenuEntry) {
	keyStyle := i.keyMenuKeyStyle()
	baseStyle := i.popupStyle()

	for row, entries := range lines {
		y := rect.InnerY + row
		x := rect.InnerX
		for _, entry := range entries {
			if x > rect.InnerX {
				for _, r := range "  " {
					if x >= rect.InnerX+rect.InnerW {
						break
					}
					screen.SetContent(x, y, r, nil, baseStyle)
					x++
				}
			}
			for _, r := range entry.Key {
				if x >= rect.InnerX+rect.InnerW {
					break
				}
				screen.SetContent(x, y, r, nil, keyStyle)
				x += runewidth.RuneWidth(r)
			}
			if x < rect.InnerX+rect.InnerW {
				screen.SetContent(x, y, ' ', nil, baseStyle)
				x++
			}
			for _, r := range entry.Label {
				if x >= rect.InnerX+rect.InnerW {
					break
				}
				screen.SetContent(x, y, r, nil, baseStyle)
				x += runewidth.RuneWidth(r)
			}
		}
	}
}

func (i *InfoWindow) totalSize() int {
	sum := 0
	for _, n := range i.Suggestions {
		sum += runewidth.StringWidth(n) + 1
	}
	return sum
}

func (i *InfoWindow) scrollToSuggestion(width int) {
	x := 0
	s := i.totalSize()

	for j, n := range i.Suggestions {
		c := util.CharacterCountInString(n)
		if j == i.CurSuggestion {
			if x+c >= i.hscroll+width {
				i.hscroll = util.Clamp(x+c+1-width, 0, util.Max(0, s-width))
			} else if x < i.hscroll {
				i.hscroll = util.Clamp(x-1, 0, util.Max(0, s-width))
			}
			break
		}
		x += c + 1
	}

	if s-width <= 0 {
		i.hscroll = 0
	}
}

func (i *InfoWindow) displaySuggestions(rect popupRect) {
	if !i.HasSuggestions || len(i.Suggestions) <= 1 || rect.Y <= 0 || rect.InnerW <= 0 {
		return
	}

	i.scrollToSuggestion(rect.InnerW)
	x := rect.InnerX - i.hscroll
	done := false

	statusLineStyle := config.DefStyle.Reverse(true)
	if style, ok := config.Colorscheme["statusline.suggestions"]; ok {
		statusLineStyle = style
	} else if style, ok := config.Colorscheme["statusline"]; ok {
		statusLineStyle = style
	}

	draw := func(r rune, s tcell.Style) {
		y := rect.Y - 1
		rw := runewidth.RuneWidth(r)
		for j := 0; j < rw; j++ {
			c := r
			if j > 0 {
				c = ' '
			}

			switch {
			case x == rect.InnerX+rect.InnerW-1 && !done:
				screen.SetContent(rect.InnerX+rect.InnerW-1, y, '>', nil, s)
				x++
				done = true
				break
			case x == rect.InnerX && i.hscroll > 0:
				screen.SetContent(rect.InnerX, y, '<', nil, s)
			case x >= rect.InnerX && x < rect.InnerX+rect.InnerW:
				screen.SetContent(x, y, c, nil, s)
			}
			x++
		}
	}

	for fillX := rect.InnerX; fillX < rect.InnerX+rect.InnerW; fillX++ {
		screen.SetContent(fillX, rect.Y-1, ' ', nil, statusLineStyle)
	}

	for j, suggestion := range i.Suggestions {
		style := statusLineStyle
		if i.CurSuggestion == j {
			style = style.Reverse(true)
		}
		for _, r := range suggestion {
			draw(r, style)
		}
		draw(' ', statusLineStyle)
	}
}

func (i *InfoWindow) Display() {
	rect, keyLayout, messageLines, ok := i.popupGeometry()
	if !ok {
		return
	}

	baseStyle := i.popupStyle()
	borderStyle := i.popupBorderStyle()
	messageStyle := i.mergePopupStyle(i.currentMessageStyle())

	i.drawPopupFrame(rect, baseStyle, borderStyle)

	if len(keyLayout) > 0 {
		i.displayKeyMenu(rect, keyLayout)
	}

	if len(messageLines) > 0 {
		drawPopupLines(rect.InnerX, rect.MessageY, rect.InnerW, 0, messageLines,
			func(int) tcell.Style { return baseStyle },
			func(int) tcell.Style { return messageStyle },
		)
	}

	if i.hasInputLine() && rect.InputY >= 0 {
		drawPopupLines(rect.InnerX, rect.InputY, rect.InnerW, 0, []string{""},
			func(int) tcell.Style { return baseStyle },
			func(int) tcell.Style { return messageStyle },
		)
		i.displayBuffer(rect, rect.InputY, messageStyle)
	}

	i.displaySuggestions(rect)
}

package display

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v3"
	luar "layeh.com/gopher-luar"

	runewidth "github.com/mattn/go-runewidth"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/util"
	lua "github.com/yuin/gopher-lua"
)

// StatusLine represents the information line at the bottom
// of each window
// It gives information such as filename, whether the file has been
// modified, filetype, cursor location
type StatusLine struct {
	Info map[string]func(*buffer.Buffer) string

	win *BufWindow
}

const statusStyleMarker = byte(0x1f)

type statusCell struct {
	r     rune
	combc []rune
	style tcell.Style
}

var statusInfo = map[string]func(*buffer.Buffer) string{
	"filename": func(b *buffer.Buffer) string {
		return b.GetName()
	},
	"line": func(b *buffer.Buffer) string {
		return strconv.Itoa(b.GetActiveCursor().Y + 1)
	},
	"col": func(b *buffer.Buffer) string {
		return strconv.Itoa(b.GetActiveCursor().X + 1)
	},
	"modified": func(b *buffer.Buffer) string {
		if b.Modified() {
			return "+ "
		}
		if b.Type.Readonly {
			return "[ro] "
		}
		return ""
	},
	"overwrite": func(b *buffer.Buffer) string {
		if b.OverwriteMode && !b.Type.Readonly {
			return "[ovwr] "
		}
		return ""
	},
	"lines": func(b *buffer.Buffer) string {
		return strconv.Itoa(b.LinesNum())
	},
	"percentage": func(b *buffer.Buffer) string {
		return strconv.Itoa((b.GetActiveCursor().Y + 1) * 100 / b.LinesNum())
	},
}

func SetStatusInfoFnLua(fn string) {
	luaFn := strings.Split(fn, ".")
	if len(luaFn) <= 1 {
		return
	}
	plName, plFn := luaFn[0], luaFn[1]
	pl := config.FindPlugin(plName)
	if pl == nil {
		return
	}
	statusInfo[fn] = func(b *buffer.Buffer) string {
		if pl == nil || !pl.IsLoaded() {
			return ""
		}
		val, err := pl.Call(plFn, luar.New(ulua.L, b))
		if err == nil {
			if v, ok := val.(lua.LString); !ok {
				screen.TermMessage(plFn, "should return a string")
				return ""
			} else {
				return string(v)
			}
		}
		return ""
	}
}

// NewStatusLine returns a statusline bound to a window
func NewStatusLine(win *BufWindow) *StatusLine {
	s := new(StatusLine)
	s.win = win
	return s
}

func findOpt(b *buffer.Buffer, opt string) any {
	if b != nil {
		if val, ok := b.Settings[opt]; ok {
			return val
		}
	}
	return "null"
}

var formatParser = regexp.MustCompile(`\$\(.+?\)`)

// FormatStatusText expands the statusline template directives in format.
func FormatStatusText(b *buffer.Buffer, format string) string {
	text := []byte(format)
	text = formatParser.ReplaceAllFunc(text, func(match []byte) []byte {
		name := match[2 : len(match)-1]
		if bytes.HasPrefix(name, []byte("opt")) {
			option := name[4:]
			return fmt.Append(nil, findOpt(b, string(option)))
		} else if bytes.HasPrefix(name, []byte("symbol")) {
			if symbol := config.GetColorschemeSymbolOrDefault(string(name[7:])); symbol != "" {
				return []byte(symbol)
			}
			return []byte{}
		} else if bytes.HasPrefix(name, []byte("bind")) {
			binding := string(name[5:])
			resolved := bindingLabelForAction(binding)
			if resolved == "" {
				return []byte("null")
			}
			return []byte(resolved)
		} else if b != nil {
			if fn, ok := statusInfo[string(name)]; ok {
				return []byte(fn(b))
			}
		}
		return []byte{}
	})
	return string(text)
}

func truncateStart(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if runewidth.StringWidth(text) <= maxWidth {
		return text
	}
	if maxWidth == 1 {
		return "…"
	}

	width := 1
	start := len(text)
	for start > 0 {
		r, _, size := util.DecodeCharacterInString(text[:start])
		rw := runewidth.RuneWidth(r)
		if rw <= 0 {
			rw = 1
		}
		if width+rw > maxWidth {
			break
		}
		start -= size
		width += rw
	}
	return "…" + text[start:]
}

func truncateStatusCellsStart(cells []statusCell, maxWidth int) []statusCell {
	if maxWidth <= 0 || len(cells) == 0 {
		return nil
	}
	if statusCellsWidth(cells) <= maxWidth {
		return cells
	}
	if maxWidth == 1 {
		return []statusCell{{r: '…', style: cells[0].style}}
	}

	width := 1
	start := len(cells)
	for start > 0 {
		rw := runewidth.RuneWidth(cells[start-1].r)
		if rw <= 0 {
			rw = 1
		}
		if width+rw > maxWidth {
			break
		}
		start--
		width += rw
	}

	trimmed := make([]statusCell, 0, len(cells)-start+1)
	ellipsisStyle := cells[0].style
	if start < len(cells) {
		ellipsisStyle = cells[start].style
	}
	trimmed = append(trimmed, statusCell{r: '…', style: ellipsisStyle})
	trimmed = append(trimmed, cells[start:]...)
	return trimmed
}

func fitFilenameInStatusline(leftCells []statusCell, filename string, availableWidth int) []statusCell {
	if filename == "" || availableWidth <= 0 {
		return leftCells
	}

	leftLen := statusCellsWidth(leftCells)
	if leftLen <= availableWidth {
		return leftCells
	}

	filenameRunes := []rune(filename)
	filenameIdx := -1
	for i := 0; i <= len(leftCells)-len(filenameRunes); i++ {
		match := true
		for j, r := range filenameRunes {
			if leftCells[i+j].r != r {
				match = false
				break
			}
		}
		if match {
			filenameIdx = i
			break
		}
	}
	if filenameIdx < 0 {
		return leftCells
	}

	filenameCells := leftCells[filenameIdx : filenameIdx+len(filenameRunes)]
	filenameWidth := statusCellsWidth(filenameCells)
	nonFilenameWidth := leftLen - filenameWidth
	if nonFilenameWidth >= availableWidth {
		trimmedFilename := truncateStatusCellsStart(filenameCells, 1)
		trimmed := append([]statusCell{}, leftCells[:filenameIdx]...)
		trimmed = append(trimmed, trimmedFilename...)
		trimmed = append(trimmed, leftCells[filenameIdx+len(filenameRunes):]...)
		return trimmed
	}

	maxFilenameWidth := availableWidth - nonFilenameWidth
	trimmedFilename := truncateStatusCellsStart(filenameCells, maxFilenameWidth)
	trimmed := append([]statusCell{}, leftCells[:filenameIdx]...)
	trimmed = append(trimmed, trimmedFilename...)
	trimmed = append(trimmed, leftCells[filenameIdx+len(filenameRunes):]...)
	return trimmed
}

func mergeStatusStyle(base, accent tcell.Style) tcell.Style {
	fg := accent.GetForeground()
	if fg == tcell.ColorDefault {
		fg = base.GetForeground()
	}
	return base.Foreground(fg).
		Bold(accent.HasBold()).
		Italic(accent.HasItalic()).
		Underline(accent.HasUnderline())
}

func parseStatusCells(text []byte, baseStyle tcell.Style) []statusCell {
	cells := make([]statusCell, 0, len(text))
	style := baseStyle

	for len(text) > 0 {
		if text[0] == statusStyleMarker {
			if end := bytes.IndexByte(text[1:], statusStyleMarker); end >= 0 {
				styleName := string(text[1 : 1+end])
				if styleName == "" {
					style = baseStyle
				} else if s, ok := config.GetColorschemeStyle(styleName); ok {
					style = mergeStatusStyle(baseStyle, s)
				} else {
					style = baseStyle
				}
				text = text[end+2:]
				continue
			}
		}

		r, combc, size := util.DecodeCharacter(text)
		cells = append(cells, statusCell{r: r, combc: combc, style: style})
		text = text[size:]
	}

	return cells
}

func statusCellsWidth(cells []statusCell) int {
	width := 0
	for _, cell := range cells {
		rw := runewidth.RuneWidth(cell.r)
		if rw <= 0 {
			rw = 1
		}
		width += rw
	}
	return width
}

// Display draws the statusline to the screen
func (s *StatusLine) Display() {
	// We'll draw the line at the lowest line in the window
	y := s.win.Height + s.win.Y - 1

	winX := s.win.X

	b := s.win.Buf
	// autocomplete suggestions (for the buffer, not for the infowindow)
	if b.HasSuggestions && len(b.Suggestions) > 1 {
		statusLineStyle := config.DefStyle.Reverse(true)
		if style, ok := config.Colorscheme["statusline.suggestions"]; ok {
			statusLineStyle = style
		} else if style, ok := config.Colorscheme["statusline"]; ok {
			statusLineStyle = style
		}
		x := 0
		for j, sug := range b.Suggestions {
			style := statusLineStyle
			if b.CurSuggestion == j {
				style = style.Reverse(true)
			}
			for _, r := range sug {
				screen.SetContent(winX+x, y, r, nil, style)
				x++
				if x >= s.win.Width {
					return
				}
			}
			screen.SetContent(winX+x, y, ' ', nil, statusLineStyle)
			x++
			if x >= s.win.Width {
				return
			}
		}

		for x < s.win.Width {
			screen.SetContent(winX+x, y, ' ', nil, statusLineStyle)
			x++
		}
		return
	}

	leftText := []byte(FormatStatusText(s.win.Buf, s.win.Buf.Settings["statusformatl"].(string)))
	rightText := []byte(FormatStatusText(s.win.Buf, s.win.Buf.Settings["statusformatr"].(string)))

	statusLineStyle := config.DefStyle.Reverse(true)
	if s.win.IsActive() {
		if style, ok := config.Colorscheme["statusline"]; ok {
			statusLineStyle = style
		}
	} else {
		if style, ok := config.Colorscheme["statusline.inactive"]; ok {
			statusLineStyle = style
		} else if style, ok := config.Colorscheme["statusline"]; ok {
			statusLineStyle = style
		}
	}

	rightCells := parseStatusCells(rightText, statusLineStyle)
	rightLen := statusCellsWidth(rightCells)
	leftCells := parseStatusCells(leftText, statusLineStyle)
	leftCells = fitFilenameInStatusline(leftCells, s.win.Buf.GetName(), s.win.Width-rightLen)
	leftLen := statusCellsWidth(leftCells)
	leftCellIdx := 0
	rightCellIdx := 0

	for x := 0; x < s.win.Width; x++ {
		if x < leftLen && leftCellIdx < len(leftCells) {
			cell := leftCells[leftCellIdx]
			leftCellIdx++
			rw := runewidth.RuneWidth(cell.r)
			for j := 0; j < rw; j++ {
				c := cell.r
				combc := cell.combc
				if j > 0 {
					c = ' '
					combc = nil
					x++
				}
				screen.SetContent(winX+x, y, c, combc, cell.style)
			}
		} else if x >= s.win.Width-rightLen && rightCellIdx < len(rightCells) {
			cell := rightCells[rightCellIdx]
			rightCellIdx++
			rw := runewidth.RuneWidth(cell.r)
			for j := 0; j < rw; j++ {
				c := cell.r
				combc := cell.combc
				if j > 0 {
					c = ' '
					combc = nil
					x++
				}
				screen.SetContent(winX+x, y, c, combc, cell.style)
			}
		} else {
			screen.SetContent(winX+x, y, ' ', nil, statusLineStyle)
		}
	}
}

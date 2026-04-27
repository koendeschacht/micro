package display

import (
	"testing"

	"github.com/gdamore/tcell/v3"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/keymenu"
	"github.com/stretchr/testify/assert"
)

func init() {
	config.InitRuntimeFiles(false)
	config.InitGlobalSettings()
}

func TestTruncateStart(t *testing.T) {
	assert.Equal(t, "", truncateStart("abcdef", 0))
	assert.Equal(t, "…", truncateStart("abcdef", 1))
	assert.Equal(t, "…ef", truncateStart("abcdef", 3))
	assert.Equal(t, "abcdef", truncateStart("abcdef", 6))
}

func TestFitFilenameInStatusline(t *testing.T) {
	left := parseStatusCells([]byte("/very/long/path/to/file.txt + (10,2) | ft:go"), config.DefStyle)
	trimmedCells := fitFilenameInStatusline(left, "/very/long/path/to/file.txt", 28)
	var trimmed []rune
	for _, cell := range trimmedCells {
		trimmed = append(trimmed, cell.r)
	}
	trimmedText := string(trimmed)

	assert.Contains(t, trimmedText, "file.txt + (10,2) | ft:go")
	assert.Contains(t, trimmedText, "…")
	assert.LessOrEqual(t, runewidth.StringWidth(trimmedText), 28)
	assert.NotContains(t, trimmedText, "/very/long/path/to/file.txt")
}

func TestFormatStatusText(t *testing.T) {
	keymenu.ResetBindings()
	keymenu.SetBinding("buffer", "Alt-g", "ToggleKeyMenu", []keymenu.Token{{Key: "g", Alt: true}})
	keymenu.SetBinding("buffer", "Ctrl-g", "ToggleHelp", []keymenu.Token{{Key: "g", Ctrl: true}})

	b := buffer.NewBufferFromString("hello", "/tmp/example.txt", buffer.BTDefault)
	b.Settings["encoding"] = "utf-8"

	formatted := FormatStatusText(b, "$(bind:ToggleKeyMenu) $(symbol:statusline.bindings) $(filename) $(opt:encoding) $(bind:ToggleHelp)")

	assert.Equal(t, "M-g  /tmp/example.txt utf-8 ^g", formatted)
}

func TestParseStatusCells(t *testing.T) {
	config.DefStyle = config.StringToStyle("#c8d3f5,#1e2030")
	config.Colorscheme = map[string]tcell.Style{
		"statusline":               config.DefStyle,
		"statusline.git.clean":     config.StringToStyle("#c3e88d,#1e2030"),
		"statusline.git.unsynced":  config.StringToStyle("#ffc777,#1e2030"),
		"statusline.git.dirty":     config.StringToStyle("#ff757f,#1e2030"),
	}

	styled := []byte("main " + string(statusStyleMarker) + "statusline.git.clean" + string(statusStyleMarker) + "branch" + string(statusStyleMarker) + string(statusStyleMarker) + " =")
	cells := parseStatusCells(styled, config.Colorscheme["statusline"])

	assert.Equal(t, 13, statusCellsWidth(cells))
	assert.Equal(t, 'm', cells[0].r)
	assert.Equal(t, 'b', cells[5].r)
	assert.Equal(t, config.StringToStyle("#c3e88d,#1e2030").GetForeground(), cells[5].style.GetForeground())
	assert.Equal(t, config.Colorscheme["statusline"].GetForeground(), cells[len(cells)-1].style.GetForeground())
}

func TestFitStyledFilenameInStatusline(t *testing.T) {
	config.DefStyle = config.StringToStyle("#c8d3f5,#1e2030")
	config.Colorscheme = map[string]tcell.Style{
		"statusline":               config.DefStyle,
		"statusline.path":          config.StringToStyle("#828bb8,#1e2030"),
		"statusline.path.filename": config.StringToStyle("#ffffff,#1e2030"),
	}

	styled := []byte(
		string(statusStyleMarker) + "statusline.path" + string(statusStyleMarker) + "/very/long/path/to/" +
			string(statusStyleMarker) + "statusline.path.filename" + string(statusStyleMarker) + "file.txt" +
			string(statusStyleMarker) + string(statusStyleMarker) + " + (10,2)",
	)
	leftCells := parseStatusCells(styled, config.Colorscheme["statusline"])
	trimmedCells := fitFilenameInStatusline(leftCells, "/very/long/path/to/file.txt", 18)

	var trimmed []rune
	for _, cell := range trimmedCells {
		trimmed = append(trimmed, cell.r)
	}
	trimmedText := string(trimmed)

	assert.Equal(t, "…file.txt + (10,2)", trimmedText)
	assert.Equal(t, config.StringToStyle("#ffffff,#1e2030").GetForeground(), trimmedCells[0].style.GetForeground())
	assert.Equal(t, config.StringToStyle("#ffffff,#1e2030").GetForeground(), trimmedCells[1].style.GetForeground())
}

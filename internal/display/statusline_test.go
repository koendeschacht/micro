package display

import (
	"testing"

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
	left := []byte("/very/long/path/to/file.txt + (10,2) | ft:go")
	trimmed := string(fitFilenameInStatusline(left, "/very/long/path/to/file.txt", 28))

	assert.Contains(t, trimmed, "file.txt + (10,2) | ft:go")
	assert.Contains(t, trimmed, "…")
	assert.LessOrEqual(t, runewidth.StringWidth(trimmed), 28)
	assert.NotContains(t, trimmed, "/very/long/path/to/file.txt")
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

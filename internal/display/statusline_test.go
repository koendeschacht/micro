package display

import (
	"testing"

	runewidth "github.com/mattn/go-runewidth"
	"github.com/stretchr/testify/assert"
)

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

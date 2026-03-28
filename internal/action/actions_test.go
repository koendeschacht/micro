package action

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClipboardHistoryPreviewEscapesLineBreaks(t *testing.T) {
	assert.Equal(t, `alpha\nbeta\ngamma`, clipboardHistoryPreview("alpha\r\nbeta\rgamma"))
}

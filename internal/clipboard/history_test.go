package clipboard

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHistoryTracksNewestWrites(t *testing.T) {
	oldHistory := history
	defer func() {
		history = oldHistory
	}()

	history = nil

	assert.NoError(t, write("first", ClipboardReg, Internal))
	assert.NoError(t, write("second", ClipboardReg, Internal))
	assert.Equal(t, []string{"second", "first"}, GetHistory())
}

func TestGetHistoryKeepsLastTenWrites(t *testing.T) {
	oldHistory := history
	defer func() {
		history = oldHistory
	}()

	history = nil
	for i := 0; i < 12; i++ {
		assert.NoError(t, write(fmt.Sprintf("value-%d", i), ClipboardReg, Internal))
	}

	assert.Equal(t, []string{
		"value-11",
		"value-10",
		"value-9",
		"value-8",
		"value-7",
		"value-6",
		"value-5",
		"value-4",
		"value-3",
		"value-2",
	}, GetHistory())
}

func TestGetHistoryIncludesPrimaryWrites(t *testing.T) {
	oldHistory := history
	defer func() {
		history = oldHistory
	}()

	history = nil

	assert.NoError(t, write("clipboard", ClipboardReg, Internal))
	assert.NoError(t, write("primary", PrimaryReg, Internal))
	assert.Equal(t, []string{"primary", "clipboard"}, GetHistory())
}

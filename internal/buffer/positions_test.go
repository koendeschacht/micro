package buffer

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRememberedPositionRestore(t *testing.T) {
	oldConfigDir := config.ConfigDir
	oldRemembered := rememberedPositions
	oldSeq := rememberedPositionSeq
	oldOption := config.GlobalSettings["remember_position"]

	config.ConfigDir = t.TempDir()
	rememberedPositions = make(map[string]rememberedPosition)
	rememberedPositionSeq = 0
	config.GlobalSettings["remember_position"] = true

	t.Cleanup(func() {
		config.ConfigDir = oldConfigDir
		rememberedPositions = oldRemembered
		rememberedPositionSeq = oldSeq
		config.GlobalSettings["remember_position"] = oldOption
	})

	path := filepath.Join(config.ConfigDir, "sample.txt")
	require.NoError(t, os.WriteFile(path, []byte("alpha\nbeta\ngamma\n"), 0o644))

	rememberedPositions[path] = rememberedPosition{Loc: Loc{2, 1}, Seq: 1}
	rememberedPositionSeq = 1

	b, err := NewBufferFromFile(path, BTDefault)
	require.NoError(t, err)
	t.Cleanup(func() {
		b.Close()
	})

	assert.Equal(t, Loc{2, 1}, b.GetActiveCursor().Loc)
}

func TestRememberedPositionDoesNotOverrideExplicitStart(t *testing.T) {
	oldConfigDir := config.ConfigDir
	oldRemembered := rememberedPositions
	oldSeq := rememberedPositionSeq
	oldOption := config.GlobalSettings["remember_position"]

	config.ConfigDir = t.TempDir()
	rememberedPositions = make(map[string]rememberedPosition)
	rememberedPositionSeq = 0
	config.GlobalSettings["remember_position"] = true

	t.Cleanup(func() {
		config.ConfigDir = oldConfigDir
		rememberedPositions = oldRemembered
		rememberedPositionSeq = oldSeq
		config.GlobalSettings["remember_position"] = oldOption
	})

	path := filepath.Join(config.ConfigDir, "sample.txt")
	require.NoError(t, os.WriteFile(path, []byte("alpha\nbeta\ngamma\n"), 0o644))

	rememberedPositions[path] = rememberedPosition{Loc: Loc{2, 1}, Seq: 1}
	rememberedPositionSeq = 1

	b, err := NewBufferFromFileWithCommand(path, BTDefault, Command{StartCursor: Loc{1, 0}})
	require.NoError(t, err)
	t.Cleanup(func() {
		b.Close()
	})

	assert.Equal(t, Loc{1, 0}, b.GetActiveCursor().Loc)
}

func TestRememberedPositionsSaveLoadAndTrim(t *testing.T) {
	oldConfigDir := config.ConfigDir
	oldRemembered := rememberedPositions
	oldSeq := rememberedPositionSeq
	oldOption := config.GlobalSettings["remember_position"]

	config.ConfigDir = t.TempDir()
	rememberedPositions = make(map[string]rememberedPosition)
	rememberedPositionSeq = 0
	config.GlobalSettings["remember_position"] = true

	t.Cleanup(func() {
		config.ConfigDir = oldConfigDir
		rememberedPositions = oldRemembered
		rememberedPositionSeq = oldSeq
		config.GlobalSettings["remember_position"] = oldOption
	})

	oldestPath := filepath.Join(config.ConfigDir, "file-0")
	for i := 0; i < maxRememberedPositions+1; i++ {
		rememberedPositions[filepath.Join(config.ConfigDir, fmt.Sprintf("file-%d", i))] = rememberedPosition{
			Loc: Loc{X: i, Y: i},
			Seq: uint64(i + 1),
		}
	}
	rememberedPositionSeq = uint64(maxRememberedPositions + 1)
	trimRememberedPositions()

	assert.Len(t, rememberedPositions, maxRememberedPositions)
	assert.NotContains(t, rememberedPositions, oldestPath)

	require.NoError(t, SaveRememberedPositions())
	rememberedPositions = make(map[string]rememberedPosition)
	rememberedPositionSeq = 0
	require.NoError(t, LoadRememberedPositions())

	assert.Len(t, rememberedPositions, maxRememberedPositions)
	assert.Equal(t, uint64(maxRememberedPositions+1), rememberedPositionSeq)
}

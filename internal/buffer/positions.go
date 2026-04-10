package buffer

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/util"
)

const maxRememberedPositions = 1000

type rememberedPosition struct {
	Loc Loc
	Seq uint64
}

type rememberedPositionFile struct {
	NextSeq   uint64
	Positions map[string]rememberedPosition
}

var rememberedPositions = make(map[string]rememberedPosition)
var rememberedPositionSeq uint64

func rememberPositionEnabled() bool {
	v, ok := config.GlobalSettings["remember_position"]
	if !ok {
		return false
	}
	enabled, ok := v.(bool)
	return ok && enabled
}

func rememberedPositionsPath() string {
	return filepath.Join(config.ConfigDir, "buffers", "positions")
}

func ensureBuffersDir() error {
	return os.MkdirAll(filepath.Join(config.ConfigDir, "buffers"), os.ModePerm)
}

func trimRememberedPositions() {
	for len(rememberedPositions) > maxRememberedPositions {
		oldestPath := ""
		oldestSeq := ^uint64(0)
		for path, pos := range rememberedPositions {
			if pos.Seq < oldestSeq {
				oldestPath = path
				oldestSeq = pos.Seq
			}
		}
		delete(rememberedPositions, oldestPath)
	}
}

// LoadRememberedPositions restores the remembered cursor positions map.
func LoadRememberedPositions() error {
	rememberedPositions = make(map[string]rememberedPosition)
	rememberedPositionSeq = 0

	if err := ensureBuffersDir(); err != nil {
		return err
	}

	file, err := os.Open(rememberedPositionsPath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	var positions rememberedPositionFile
	if err := gob.NewDecoder(file).Decode(&positions); err != nil {
		return err
	}

	if positions.Positions != nil {
		rememberedPositions = positions.Positions
	}
	rememberedPositionSeq = positions.NextSeq
	for _, pos := range rememberedPositions {
		if pos.Seq > rememberedPositionSeq {
			rememberedPositionSeq = pos.Seq
		}
	}
	trimRememberedPositions()
	return nil
}

// SaveRememberedPositions writes the remembered cursor positions map to disk.
func SaveRememberedPositions() error {
	if !rememberPositionEnabled() {
		return nil
	}

	if err := ensureBuffersDir(); err != nil {
		return err
	}

	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(rememberedPositionFile{
		NextSeq:   rememberedPositionSeq,
		Positions: rememberedPositions,
	})
	if err != nil {
		return err
	}

	return util.SafeWrite(rememberedPositionsPath(), buf.Bytes(), true)
}

// RecordRememberedPosition stores the current cursor position for a buffer.
func RecordRememberedPosition(b *Buffer) {
	if !rememberPositionEnabled() || b == nil || b.Type != BTDefault || b.AbsPath == "" || b.NumCursors() == 0 {
		return
	}

	rememberedPositionSeq++
	rememberedPositions[b.AbsPath] = rememberedPosition{
		Loc: b.GetActiveCursor().Loc.Clamp(b.Start(), b.End()),
		Seq: rememberedPositionSeq,
	}
	trimRememberedPositions()
}

// RecordOpenBufferPositions snapshots the current cursor position of all open file buffers.
func RecordOpenBufferPositions() {
	for _, b := range OpenBuffers {
		RecordRememberedPosition(b)
	}
}

func getRememberedPosition(path string) (Loc, bool) {
	pos, ok := rememberedPositions[path]
	if !ok {
		return Loc{}, false
	}
	return pos.Loc, true
}

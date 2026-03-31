package dict

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

// Matrix holds the bigram connection cost matrix loaded from matrix.bin.
// Costs are indexed by (right context ID of previous morpheme,
// left context ID of current morpheme).
type Matrix struct {
	lsize uint16
	rsize uint16
	data  []int16
}

// LoadMatrix reads the binary connection cost matrix from matrix.bin.
// Format: uint16 lsize, uint16 rsize, then lsize*rsize int16 values.
func LoadMatrix(dir string) (*Matrix, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "matrix.bin"))
	if err != nil {
		return nil, fmt.Errorf("read matrix.bin: %w", err)
	}

	if len(raw) < 4 {
		return nil, fmt.Errorf("matrix.bin too small: %d bytes", len(raw))
	}

	m := &Matrix{
		lsize: binary.LittleEndian.Uint16(raw[0:]),
		rsize: binary.LittleEndian.Uint16(raw[2:]),
	}

	count := int(m.lsize) * int(m.rsize)
	expected := 4 + count*2
	if len(raw) < expected {
		return nil, fmt.Errorf("matrix.bin truncated: want %d bytes, got %d", expected, len(raw))
	}

	m.data = make([]int16, count)
	for i := 0; i < count; i++ {
		off := 4 + i*2
		m.data[i] = int16(binary.LittleEndian.Uint16(raw[off:]))
	}

	return m, nil
}

// ConnCost returns the connection cost between two adjacent morphemes.
// prevRightID: right context ID of the preceding morpheme.
// currLeftID: left context ID of the current morpheme.
func (m *Matrix) ConnCost(prevRightID, currLeftID uint16) int32 {
	idx := int(prevRightID) + int(m.lsize)*int(currLeftID)
	if idx < 0 || idx >= len(m.data) {
		return 0
	}
	return int32(m.data[idx])
}

package dict

import "encoding/binary"

// DoubleArray implements the DARTS (Double-Array Trie System) used by MeCab
// for fast dictionary lookup. The structure is loaded directly from compiled
// MeCab dictionary files (sys.dic, unk.dic).
type DoubleArray struct {
	base  []int32
	check []uint32
}

// ResultPair holds a single result from CommonPrefixSearch.
// Value encodes (tokenOffset << 8) | tokenCount.
// Length is the matched key length in bytes.
type ResultPair struct {
	Value  int32
	Length int
}

// NewDoubleArray creates a DoubleArray from raw binary data.
// Each unit is 8 bytes: int32 base + uint32 check, little-endian.
func NewDoubleArray(data []byte) *DoubleArray {
	n := len(data) / 8
	da := &DoubleArray{
		base:  make([]int32, n),
		check: make([]uint32, n),
	}
	for i := 0; i < n; i++ {
		off := i * 8
		da.base[i] = int32(binary.LittleEndian.Uint32(data[off:]))
		da.check[i] = binary.LittleEndian.Uint32(data[off+4:])
	}
	return da
}

// CommonPrefixSearch finds all dictionary entries that are prefixes of key.
// Returns results ordered by increasing key length.
func (da *DoubleArray) CommonPrefixSearch(key []byte) []ResultPair {
	size := len(da.base)
	if size == 0 {
		return nil
	}

	var results []ResultPair
	b := uint32(da.base[0])

	for i := 0; i <= len(key); i++ {
		// Check for end-of-word marker at position b.
		// The leaf node at p=b uses the same check as traversal: check[p] must equal b.
		p := int(b)
		if p < size && b == da.check[p] {
			n := da.base[p]
			if n < 0 {
				results = append(results, ResultPair{
					Value:  -n - 1,
					Length: i,
				})
			}
		}

		if i >= len(key) {
			break
		}

		// Traverse to next byte: offset = base + byte_value + 1
		nextP := int(b) + int(key[i]) + 1
		if nextP >= 0 && nextP < size && b == da.check[nextP] {
			b = uint32(da.base[nextP])
		} else {
			break
		}
	}

	return results
}

// ExactMatchSearch finds the exact entry matching key.
// Returns the value and true if found, or 0 and false if not.
func (da *DoubleArray) ExactMatchSearch(key []byte) (int32, bool) {
	size := len(da.base)
	if size == 0 {
		return 0, false
	}

	b := uint32(da.base[0])

	for i := 0; i < len(key); i++ {
		p := int(b) + int(key[i]) + 1
		if p >= 0 && p < size && b == da.check[p] {
			b = uint32(da.base[p])
		} else {
			return 0, false
		}
	}

	p := int(b)
	if p >= 0 && p < size && b == da.check[p] {
		n := da.base[p]
		if n < 0 {
			return -n - 1, true
		}
	}
	return 0, false
}

package dict

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

const (
	dicMagicID = 0xef718f77
	headerSize = 72 // 10 * uint32 (40 bytes) + 32-byte charset
	tokenSize  = 16 // sizeof(Token) in MeCab's C code
)

// Token represents a single dictionary entry as stored in compiled MeCab dictionaries.
type Token struct {
	LcAttr   uint16 // left context attribute ID
	RcAttr   uint16 // right context attribute ID
	PosID    uint16 // part-of-speech ID
	WCost    int16  // word cost (lower = more likely)
	Feature  uint32 // byte offset into feature string pool
	Compound uint32 // compound word information
}

// Dictionary holds a loaded MeCab binary dictionary (sys.dic or unk.dic).
type Dictionary struct {
	Version uint32
	Type    uint32 // 0=user, 1=sys, 2=unk
	Charset string
	Trie    *DoubleArray
	Tokens  []Token
	features []byte
}

// LoadDictionary reads and parses a compiled MeCab dictionary file.
func LoadDictionary(path string) (*Dictionary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dictionary %s: %w", path, err)
	}

	if len(data) < headerSize {
		return nil, fmt.Errorf("dictionary too small: %d bytes", len(data))
	}

	// Parse header (10 x uint32 + 32-byte charset)
	magic := binary.LittleEndian.Uint32(data[0:])
	version := binary.LittleEndian.Uint32(data[4:])
	dicType := binary.LittleEndian.Uint32(data[8:])
	// lexSize := binary.LittleEndian.Uint32(data[12:])  // unused
	// lSize := binary.LittleEndian.Uint32(data[16:])     // unused
	// rSize := binary.LittleEndian.Uint32(data[20:])     // unused
	dSize := binary.LittleEndian.Uint32(data[24:])
	tSize := binary.LittleEndian.Uint32(data[28:])
	fSize := binary.LittleEndian.Uint32(data[32:])
	// dummy := binary.LittleEndian.Uint32(data[36:])     // unused

	// Charset is a null-terminated string in 32 bytes
	charsetBytes := data[40:72]
	charsetEnd := 0
	for charsetEnd < 32 && charsetBytes[charsetEnd] != 0 {
		charsetEnd++
	}
	charset := string(charsetBytes[:charsetEnd])

	// Validate magic: (magic ^ DIC_MAGIC_ID) == file_size
	fileSize := uint32(len(data))
	if (magic ^ dicMagicID) != fileSize {
		return nil, fmt.Errorf("invalid dictionary magic: expected file size %d, got %d",
			fileSize, magic^dicMagicID)
	}

	d := &Dictionary{
		Version: version,
		Type:    dicType,
		Charset: charset,
	}

	offset := uint32(headerSize)

	// Load double-array trie
	if offset+dSize > fileSize {
		return nil, fmt.Errorf("trie section exceeds file size")
	}
	d.Trie = NewDoubleArray(data[offset : offset+dSize])
	offset += dSize

	// Load token array
	if offset+tSize > fileSize {
		return nil, fmt.Errorf("token section exceeds file size")
	}
	tokenCount := tSize / tokenSize
	d.Tokens = make([]Token, tokenCount)
	for i := uint32(0); i < tokenCount; i++ {
		off := offset + i*tokenSize
		d.Tokens[i] = Token{
			LcAttr:   binary.LittleEndian.Uint16(data[off:]),
			RcAttr:   binary.LittleEndian.Uint16(data[off+2:]),
			PosID:    binary.LittleEndian.Uint16(data[off+4:]),
			WCost:    int16(binary.LittleEndian.Uint16(data[off+6:])),
			Feature:  binary.LittleEndian.Uint32(data[off+8:]),
			Compound: binary.LittleEndian.Uint32(data[off+12:]),
		}
	}
	offset += tSize

	// Load feature string pool (null-terminated C strings packed together)
	if offset+fSize > fileSize {
		return nil, fmt.Errorf("feature section exceeds file size")
	}
	d.features = make([]byte, fSize)
	copy(d.features, data[offset:offset+fSize])

	return d, nil
}

// GetFeature returns the feature string at the given byte offset.
func (d *Dictionary) GetFeature(offset uint32) string {
	if int(offset) >= len(d.features) {
		return ""
	}
	end := int(offset)
	for end < len(d.features) && d.features[end] != 0 {
		end++
	}
	return string(d.features[offset:end])
}

// LookupTokens decodes a trie result value into the corresponding token entries.
// The value encodes (tokenOffset << 8) | tokenCount.
func (d *Dictionary) LookupTokens(value int32) []Token {
	count := int(value & 0xFF)
	idx := int(value >> 8)
	if idx < 0 || idx+count > len(d.Tokens) {
		return nil
	}
	return d.Tokens[idx : idx+count]
}

// LoadSystemDict loads the system dictionary (sys.dic) from a directory.
func LoadSystemDict(dir string) (*Dictionary, error) {
	return LoadDictionary(filepath.Join(dir, "sys.dic"))
}

// LoadUnkDict loads the unknown word dictionary (unk.dic) from a directory.
func LoadUnkDict(dir string) (*Dictionary, error) {
	return LoadDictionary(filepath.Join(dir, "unk.dic"))
}

package dict

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CharInfo holds the character classification for a single Unicode codepoint.
type CharInfo struct {
	Type        uint32 // bitmask of category indices this character belongs to
	DefaultType uint8  // index of the primary category
	Length      uint8  // max unknown word length in characters (0 = single char only)
	Group       bool   // if true, group consecutive same-category chars into one unknown word
	Invoke      bool   // if true, always invoke unknown word processing even if dict matches exist
}

// CharCategory defines a character category with its unknown word generation rules.
type CharCategory struct {
	Name   string
	Invoke bool
	Group  bool
	Length int
}

// CharDef holds character classification data loaded from char.bin or char.def.
type CharDef struct {
	Categories []CharCategory
	charInfo   [0xFFFF]CharInfo // one entry per Unicode codepoint 0x0000–0xFFFE
}

// GetCharInfo returns the character classification for a rune.
// Runes outside the BMP (>= 0xFFFF) get the DEFAULT category.
func (cd *CharDef) GetCharInfo(r rune) CharInfo {
	if r >= 0 && int(r) < len(cd.charInfo) {
		return cd.charInfo[r]
	}
	// Out-of-range characters get DEFAULT (index 0)
	if len(cd.Categories) > 0 {
		return CharInfo{
			Type:        1, // bit 0 = DEFAULT
			DefaultType: 0,
			Group:       cd.Categories[0].Group,
			Length:      uint8(cd.Categories[0].Length),
			Invoke:      cd.Categories[0].Invoke,
		}
	}
	return CharInfo{}
}

// CategoryName returns the name of the category at the given index.
func (cd *CharDef) CategoryName(idx uint8) string {
	if int(idx) < len(cd.Categories) {
		return cd.Categories[idx].Name
	}
	return "DEFAULT"
}

// LoadCharDef loads character definitions, trying char.bin first, then char.def.
func LoadCharDef(dir string) (*CharDef, error) {
	cd, err := loadCharBin(filepath.Join(dir, "char.bin"))
	if err == nil {
		return cd, nil
	}
	return loadCharDefText(filepath.Join(dir, "char.def"))
}

// loadCharBin parses the compiled binary character property file.
// Format: uint32 categoryCount, then categoryCount*32 bytes of names,
// then 0xFFFF CharInfo entries (4 bytes each, bit-packed).
func loadCharBin(path string) (*CharDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if len(data) < 4 {
		return nil, fmt.Errorf("char.bin too small")
	}

	cd := &CharDef{}
	csize := binary.LittleEndian.Uint32(data[0:])
	offset := 4

	// Read category names (each padded to 32 bytes)
	cd.Categories = make([]CharCategory, csize)
	for i := uint32(0); i < csize; i++ {
		if offset+32 > len(data) {
			return nil, fmt.Errorf("char.bin truncated at category %d", i)
		}
		nameBytes := data[offset : offset+32]
		end := 0
		for end < 32 && nameBytes[end] != 0 {
			end++
		}
		cd.Categories[i] = CharCategory{Name: string(nameBytes[:end])}
		offset += 32
	}

	// Read character info table (0xFFFF entries, 4 bytes each)
	needed := offset + 0xFFFF*4
	if len(data) < needed {
		return nil, fmt.Errorf("char.bin truncated at character info table")
	}

	for i := 0; i < 0xFFFF; i++ {
		v := binary.LittleEndian.Uint32(data[offset:])
		ci := CharInfo{
			Type:        v & 0x3FFFF,         // bits 0–17
			DefaultType: uint8((v >> 18) & 0xFF), // bits 18–25
			Length:      uint8((v >> 26) & 0xF),   // bits 26–29
			Group:       (v>>30)&1 != 0,           // bit 30
			Invoke:      (v>>31)&1 != 0,           // bit 31
		}
		cd.charInfo[i] = ci

		// Backfill category invoke/group/length from the first character that uses it
		catIdx := ci.DefaultType
		if int(catIdx) < len(cd.Categories) && cd.Categories[catIdx].Name != "" {
			cat := &cd.Categories[catIdx]
			cat.Invoke = ci.Invoke
			cat.Group = ci.Group
			cat.Length = int(ci.Length)
		}

		offset += 4
	}

	return cd, nil
}

// loadCharDefText parses the text-format char.def file.
func loadCharDefText(path string) (*CharDef, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cd := &CharDef{}
	catIndex := map[string]int{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '#' {
			continue
		}

		// Character mapping: starts with 0x
		if strings.HasPrefix(line, "0x") || strings.HasPrefix(line, "0X") {
			cd.parseCharMapping(line, catIndex)
			continue
		}

		// Category definition: NAME INVOKE GROUP LENGTH
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			name := fields[0]
			invoke, _ := strconv.Atoi(fields[1])
			group, _ := strconv.Atoi(fields[2])
			length, _ := strconv.Atoi(fields[3])

			idx := len(cd.Categories)
			catIndex[name] = idx
			cd.Categories = append(cd.Categories, CharCategory{
				Name:   name,
				Invoke: invoke != 0,
				Group:  group != 0,
				Length: length,
			})
		}
	}

	return cd, scanner.Err()
}

func (cd *CharDef) parseCharMapping(line string, catIndex map[string]int) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return
	}

	// Parse range: "0xAAAA" or "0xAAAA..0xBBBB"
	var from, to int
	rangeParts := strings.SplitN(fields[0], "..", 2)
	from64, err := strconv.ParseInt(strings.TrimPrefix(rangeParts[0], "0x"), 16, 32)
	if err != nil {
		from64, err = strconv.ParseInt(strings.TrimPrefix(rangeParts[0], "0X"), 16, 32)
		if err != nil {
			return
		}
	}
	from = int(from64)
	to = from

	if len(rangeParts) == 2 {
		to64, err := strconv.ParseInt(strings.TrimPrefix(rangeParts[1], "0x"), 16, 32)
		if err != nil {
			to64, err = strconv.ParseInt(strings.TrimPrefix(rangeParts[1], "0X"), 16, 32)
			if err != nil {
				return
			}
		}
		to = int(to64)
	}

	// Assign categories
	for _, catName := range fields[1:] {
		if strings.HasPrefix(catName, "#") {
			break // rest is comment
		}
		idx, ok := catIndex[catName]
		if !ok {
			continue
		}
		cat := cd.Categories[idx]

		for c := from; c <= to && c < 0xFFFF; c++ {
			ci := &cd.charInfo[c]
			ci.Type |= 1 << uint(idx)
			// First category assigned becomes the default
			if ci.Type == (1 << uint(idx)) {
				ci.DefaultType = uint8(idx)
				ci.Invoke = cat.Invoke
				ci.Group = cat.Group
				ci.Length = uint8(cat.Length)
			}
		}
	}
}

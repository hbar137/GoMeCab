package gomecab

import (
	"fmt"
	"unicode/utf8"

	"github.com/hbar137/GoMeCab/dict"
	"github.com/hbar137/GoMeCab/lattice"
)

// Tokenizer performs Japanese morphological analysis using MeCab-compatible dictionaries.
type Tokenizer struct {
	sysDict *dict.Dictionary
	unkDict *dict.Dictionary
	matrix  *dict.Matrix
	charDef *dict.CharDef
}

// Token represents a single morpheme in the analysis result.
type Token struct {
	Surface string // surface form as it appears in the input
	Feature string // comma-separated feature string (POS, reading, etc.)
	Start   int    // byte offset in input (inclusive)
	End     int    // byte offset in input (exclusive)
}

// New creates a Tokenizer by loading all dictionary files from the given directory.
// The directory must contain: sys.dic, unk.dic, matrix.bin, and char.bin (or char.def).
func New(dictDir string) (*Tokenizer, error) {
	sysDict, err := dict.LoadSystemDict(dictDir)
	if err != nil {
		return nil, fmt.Errorf("load sys.dic: %w", err)
	}

	unkDict, err := dict.LoadUnkDict(dictDir)
	if err != nil {
		return nil, fmt.Errorf("load unk.dic: %w", err)
	}

	matrix, err := dict.LoadMatrix(dictDir)
	if err != nil {
		return nil, fmt.Errorf("load matrix.bin: %w", err)
	}

	charDef, err := dict.LoadCharDef(dictDir)
	if err != nil {
		return nil, fmt.Errorf("load char.def: %w", err)
	}

	return &Tokenizer{
		sysDict: sysDict,
		unkDict: unkDict,
		matrix:  matrix,
		charDef: charDef,
	}, nil
}

// Tokenize performs morphological analysis on the input string and returns
// the optimal sequence of tokens.
func (t *Tokenizer) Tokenize(input string) []Token {
	if len(input) == 0 {
		return nil
	}

	lat := t.buildLattice(input)
	lat.Solve(t.matrix.ConnCost)
	path := lat.BestPath()
	if path == nil {
		return nil
	}

	tokens := make([]Token, len(path))
	for i, node := range path {
		tokens[i] = Token{
			Surface: node.Surface,
			Feature: node.Feature,
			Start:   node.Start,
			End:     node.End,
		}
	}
	return tokens
}

func (t *Tokenizer) buildLattice(input string) *lattice.Lattice {
	lat := lattice.New(input)
	inputBytes := []byte(input)

	pos := 0
	for pos < len(inputBytes) {
		r, rsize := utf8.DecodeRune(inputBytes[pos:])
		if r == utf8.RuneError && rsize <= 1 {
			rsize = 1 // skip invalid byte
		}

		// System dictionary lookup: find all prefixes starting at pos
		results := t.sysDict.Trie.CommonPrefixSearch(inputBytes[pos:])

		hasMatch := false
		for _, res := range results {
			if res.Length == 0 {
				continue
			}
			tokens := t.sysDict.LookupTokens(res.Value)
			for _, tok := range tokens {
				node := &lattice.Node{
					Surface: string(inputBytes[pos : pos+res.Length]),
					Feature: t.sysDict.GetFeature(tok.Feature),
					Start:   pos,
					End:     pos + res.Length,
					LeftID:  tok.LcAttr,
					RightID: tok.RcAttr,
					WCost:   int32(tok.WCost),
					Type:    lattice.NormalNode,
				}
				lat.Add(node)
				hasMatch = true
			}
		}

		// Unknown word processing
		charInfo := t.charDef.GetCharInfo(r)
		if !hasMatch || charInfo.Invoke {
			t.addUnknownNodes(lat, inputBytes, pos, rsize, charInfo)
		}

		pos += rsize
	}

	return lat
}

// addUnknownNodes generates unknown word candidates at the given position.
func (t *Tokenizer) addUnknownNodes(lat *lattice.Lattice, input []byte, pos, rsize int, info dict.CharInfo) {
	added := false

	// GROUP: find longest run of consecutive chars with the same default category
	if info.Group {
		groupEnd := pos + rsize
		for groupEnd < len(input) {
			nextR, nextSize := utf8.DecodeRune(input[groupEnd:])
			if nextR == utf8.RuneError && nextSize <= 1 {
				break
			}
			nextInfo := t.charDef.GetCharInfo(nextR)
			if nextInfo.DefaultType != info.DefaultType {
				break
			}
			groupEnd += nextSize
		}
		t.addUnknownNodesForSurface(lat, input, pos, groupEnd-pos, info)
		added = true
	}

	// LENGTH: generate candidates of 1 to LENGTH characters
	if info.Length > 0 {
		end := pos
		for k := 0; k < int(info.Length) && end < len(input); k++ {
			r, sz := utf8.DecodeRune(input[end:])
			if r == utf8.RuneError && sz <= 1 {
				break
			}
			if k > 0 {
				nextInfo := t.charDef.GetCharInfo(r)
				if nextInfo.DefaultType != info.DefaultType {
					break
				}
			}
			end += sz
			t.addUnknownNodesForSurface(lat, input, pos, end-pos, info)
			added = true
		}
	}

	// Fallback: single character unknown word
	if !added {
		t.addUnknownNodesForSurface(lat, input, pos, rsize, info)
	}
}

// addUnknownNodesForSurface creates lattice nodes for an unknown word surface form
// by looking up all matching categories in the unknown word dictionary.
func (t *Tokenizer) addUnknownNodesForSurface(lat *lattice.Lattice, input []byte, pos, length int, info dict.CharInfo) {
	surface := string(input[pos : pos+length])

	for catIdx := 0; catIdx < len(t.charDef.Categories); catIdx++ {
		if info.Type&(1<<uint(catIdx)) == 0 {
			continue
		}

		catName := t.charDef.Categories[catIdx].Name
		value, found := t.unkDict.Trie.ExactMatchSearch([]byte(catName))
		if !found {
			continue
		}

		tokens := t.unkDict.LookupTokens(value)
		for _, tok := range tokens {
			node := &lattice.Node{
				Surface: surface,
				Feature: t.unkDict.GetFeature(tok.Feature),
				Start:   pos,
				End:     pos + length,
				LeftID:  tok.LcAttr,
				RightID: tok.RcAttr,
				WCost:   int32(tok.WCost),
				Type:    lattice.UnknownNode,
			}
			lat.Add(node)
		}
	}
}

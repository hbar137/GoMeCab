package lattice

// NodeType classifies the origin of a lattice node.
type NodeType int

const (
	NormalNode  NodeType = iota // from system dictionary
	UnknownNode                 // from unknown word processing
	BOSNode                     // beginning of sentence
	EOSNode                     // end of sentence
	UserNode                    // from user dictionary
)

// Node represents a morpheme candidate in the lattice.
type Node struct {
	Surface   string
	Feature   string
	Start     int    // byte offset in input (inclusive)
	End       int    // byte offset in input (exclusive)
	LeftID    uint16 // left context attribute ID
	RightID   uint16 // right context attribute ID
	WCost     int32  // word cost from dictionary
	TotalCost int32  // accumulated Viterbi cost
	Prev      *Node  // best predecessor (set by Viterbi)
	Type      NodeType
}

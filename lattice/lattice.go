package lattice

const maxCost int32 = 0x7FFFFFFF

// Lattice is a directed acyclic graph of morpheme candidates for an input string.
// Nodes are indexed by their byte positions in the input.
type Lattice struct {
	Input      string
	BeginNodes [][]*Node // nodes starting at each byte position
	EndNodes   [][]*Node // nodes ending at each byte position
	BOS        *Node
	EOS        *Node
}

// New creates a lattice for the given input string with BOS and EOS sentinels.
func New(input string) *Lattice {
	n := len(input) + 1
	l := &Lattice{
		Input:      input,
		BeginNodes: make([][]*Node, n),
		EndNodes:   make([][]*Node, n),
	}

	l.BOS = &Node{
		Start:     0,
		End:       0,
		RightID:   0,
		TotalCost: 0,
		Type:      BOSNode,
	}
	l.EndNodes[0] = append(l.EndNodes[0], l.BOS)

	l.EOS = &Node{
		Start:  len(input),
		End:    len(input),
		LeftID: 0,
		Type:   EOSNode,
	}
	l.BeginNodes[len(input)] = append(l.BeginNodes[len(input)], l.EOS)

	return l
}

// Add inserts a morpheme node into the lattice at the appropriate positions.
func (l *Lattice) Add(node *Node) {
	if node.Start >= 0 && node.Start < len(l.BeginNodes) {
		l.BeginNodes[node.Start] = append(l.BeginNodes[node.Start], node)
	}
	if node.End >= 0 && node.End < len(l.EndNodes) {
		l.EndNodes[node.End] = append(l.EndNodes[node.End], node)
	}
}

// Solve runs the Viterbi forward pass to find the minimum-cost path.
// costFunc returns the bigram connection cost between two adjacent morphemes,
// given (prevRightID, currLeftID).
func (l *Lattice) Solve(costFunc func(prevRightID, currLeftID uint16) int32) {
	for i := 0; i < len(l.BeginNodes); i++ {
		for _, node := range l.BeginNodes[i] {
			node.TotalCost = maxCost
			for _, prev := range l.EndNodes[i] {
				if prev.TotalCost == maxCost {
					continue
				}
				cost := prev.TotalCost + costFunc(prev.RightID, node.LeftID) + node.WCost
				if cost < node.TotalCost {
					node.TotalCost = cost
					node.Prev = prev
				}
			}
		}
	}
}

// BestPath backtracks from EOS to BOS and returns the optimal morpheme sequence.
// Returns nil if no valid path exists.
func (l *Lattice) BestPath() []*Node {
	if l.EOS.TotalCost == maxCost {
		return nil
	}

	var path []*Node
	for node := l.EOS.Prev; node != nil && node.Type != BOSNode; node = node.Prev {
		path = append(path, node)
	}

	// Reverse to get left-to-right order
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

package searcher

import (
	"math"
	"risk/game"
	"sync"
)

type decision struct {
	sync.RWMutex
	parent   Node
	player   string
	moves    []game.Move
	children []Node
	delta    map[string]any
	rewards  float64
	visits   int
}

func newDecision(parent Node, state game.State) *decision {
	// TODO: randomize moves
	moves := state.LegalMoves()

	var delta map[string]any
	if _, ok := parent.(*chance); ok {
		delta = state.Delta()
	}

	return &decision{
		parent:   parent,
		player:   state.Player(),
		moves:    moves,
		children: make([]Node, 0, len(moves)),
		delta:    delta,
		rewards:  0,
		visits:   0,
	}
}

func (d *decision) SelectOrExpand(state game.State) (Node, game.State, bool) {
	d.Lock()
	defer d.Unlock()

	if len(d.moves) == 0 { // Terminal node
		return d, state, false
	}

	if len(d.moves) > len(d.children) { // Expandable node
		child, state := d.addChild(state)
		child.ApplyLoss()
		return child, state, true
	}

	// Fully expanded node
	ith := d.pickChild()
	child := d.children[ith]
	move := d.moves[ith]
	child.ApplyLoss()
	return child, state.Play(move), false
}

func (d *decision) addChild(state game.State) (Node, game.State) {
	move := d.moves[len(d.children)]
	var child Node
	if move.IsDeterministic() {
		child = newDecision(d, state)
	} else {
		child = newChance(d, state)
	}
	d.children = append(d.children, child)
	return child, state.Play(move)
}

func (d *decision) pickChild() int {
	if d.visits == 0 {
		panic("node has children but no visits")
	}

	normalizer := C_SQUARED * math.Log(float64(d.visits))

	maxIndex := -1
	maxScore := math.Inf(-1)
	for i, child := range d.children {
		score := child.Score(normalizer)
		if score == math.Inf(1) {
			return i
		}
		if score > maxScore {
			maxScore = score
			maxIndex = i
		}
	}
	return maxIndex
}

func (d *decision) ApplyLoss() {
	d.Lock()
	defer d.Unlock()

	d.rewards += LOSS
	d.visits++
}

func (d *decision) Score(normalizer float64) float64 {
	d.RLock()
	defer d.RUnlock()

	return ucb1(d.rewards, d.visits, normalizer)
}

func (d *decision) Update(rewarder func(string) float64) Node {
	d.Lock()
	defer d.Unlock()

	// Reverse virtual loss except for root node
	if d.parent != nil {
		d.rewards -= LOSS
		d.visits--
	}

	d.rewards += rewarder(d.player)
	d.visits++

	return d.parent
}

func (d *decision) Visits() int {
	d.RLock()
	defer d.RUnlock()

	return d.visits
}

func (d *decision) findBestMove() game.Move {
	bestIndex := -1
	maxVisits := -1
	for i, child := range d.children {
		if child.Visits() > maxVisits {
			maxVisits = child.Visits()
			bestIndex = i
		}
	}
	return d.moves[bestIndex]
}

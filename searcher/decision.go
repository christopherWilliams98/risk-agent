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

func (d *decision) Backup(rewarder func(string) float64) Node {
	d.Lock()
	defer d.Unlock()

	if d.parent != nil { // Non-root node
		d.reverseLoss()
	}

	d.rewards += rewarder(d.player)
	d.visits++

	return d.parent
}

func (d *decision) reverseLoss() {
	d.rewards -= LOSS
	d.visits--
}

func (d *decision) Value() int {
	d.RLock()
	defer d.RUnlock()

	return d.visits
}

func (d *decision) findBestMove() game.Move {
	if len(d.children) == 0 {
		panic("node has no children")
	}

	bestIndex := 0
	maxValue := d.children[0].Value()
	for i, child := range d.children[1:] {
		if v := child.Value(); v > maxValue {
			maxValue = v
			bestIndex = i
		}
	}
	return d.moves[bestIndex]
}

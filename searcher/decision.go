package searcher

import (
	"math"
	"risk/game"
	"sync"
)

type decision struct {
	sync.RWMutex
	parent Node
	player string
	// TODO: make sure Move is comparable
	moves    []game.Move        // Unexplored
	children map[game.Move]Node // Explored
	hash     game.StateHash
	rewards  float64
	visits   int
}

func newDecision(state game.State, parent Node) *decision {
	// TODO: randomize moves
	// TODO: prioritize 'pass' moves (in attack and maneuver phases)
	moves := state.LegalMoves()

	// Lazily compute state ID
	var hash game.StateHash
	if _, ok := parent.(*chance); ok {
		hash = state.Hash()
	}

	return &decision{
		parent:   parent,
		player:   state.Player(),
		moves:    moves,
		children: make(map[game.Move]Node, len(moves)),
		hash:     hash,
		rewards:  0,
		visits:   0,
	}
}

func (d *decision) SelectOrExpand(state game.State) (Node, game.State, bool) {
	d.Lock()
	defer d.Unlock()

	if len(d.moves) == 0 && len(d.children) == 0 { // Terminal Node
		return d, state, false
	}

	var child Node
	selected := false
	if len(d.moves) > 0 { // Expand Node with an unexplored move
		child, state = d.expands(state)
	} else { // Select a child of fully expanded Node
		child, state = d.selects(state)
		selected = true
	}

	child.ApplyLoss()
	return child, state, selected
}

func (d *decision) expands(state game.State) (Node, game.State) {
	move := d.moves[0]
	// TODO: copy state or pass arg by value?
	state = state.Play(move)

	var child Node
	if move.IsDeterministic() {
		child = newDecision(state, d)
	} else {
		child = newChance(state, d)
	}

	d.children[move] = child
	d.moves = d.moves[1:]
	return child, state
}

func (d *decision) selects(state game.State) (Node, game.State) {
	if d.visits == 0 { // Prevent log(0)
		panic("cannot select child from Node with 0 visits")
	}

	normalizer := C_SQUARED * math.Log(float64(d.visits))

	var maxMove game.Move
	maxScore := math.Inf(-1)
	for move, child := range d.children {
		// TODO: flip rewards/score if child.player != d.player
		score := child.Score(normalizer)
		if score > maxScore {
			maxScore = score
			maxMove = move
		}
	}
	// TODO: copy state or pass arg by value?
	return d.children[maxMove], state.Play(maxMove)
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

	if d.visits == 0 {
		panic("cannot compute score for child with 0 visits")
	}

	return uct(d.rewards, d.visits, normalizer)
}

func (d *decision) Backup(winner string) Node {
	d.Lock()
	defer d.Unlock()

	if d.parent != nil { // Virtual loss is not applied on root Node
		d.reverseLoss()
	}

	d.rewards += computeReward(winner, d.player)
	d.visits++

	return d.parent
}

func (d *decision) reverseLoss() {
	d.rewards -= LOSS
	d.visits--
}

func (d *decision) Visits() int {
	d.RLock()
	defer d.RUnlock()

	return d.visits
}

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
	moves    []game.Move        // Unexplored
	children map[game.Move]Node // Explored
	hash     game.StateHash
	rewards  float64
	visits   int
}

func newDecision(parent Node, state game.State) *decision {
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

// SelectOrExpand
// - if fully expanded, select a child node based on the selection policy
// - if not fully expanded, expand the node by adding a child node for an unexplored move
// - in both cases, advance the state by playing the move to the child node
// - if terminal, simply return the node itself with the state unchanged
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

	child.applyLoss()
	return child, state, selected
}

func (d *decision) expands(state game.State) (Node, game.State) {
	move := d.moves[0]
	// TODO: copy state or pass arg by value?
	state = state.Play(move)

	var child Node
	if move.IsStochastic() {
		child = newChance(d)
	} else {
		child = newDecision(d, state)
	}

	d.children[move] = child
	d.moves = d.moves[1:]
	return child, state
}

func (d *decision) selects(state game.State) (Node, game.State) {
	if len(d.children) == 0 {
		panic("no children")
	}

	if d.visits == 0 {
		panic("unexplored parent node (0 visits)")
	}

	// TODO: potentially parallelize selection policy computation on child
	//  nodes (large branching factor, many children, O(N) complexity)
	policy := newUCT(C_SQUARED, d.visits)
	maxValue := math.Inf(-1)
	var maxMove game.Move
	for move, child := range d.children {
		player, rewards, visits := child.stats()
		if visits == 0 {
			panic("unexplored child node (0 visits)")
		}
		// Maximize current player's rewards or minimize opponent's rewards
		if player != d.player { // Turn changes
			rewards = -rewards // Negate opponent's rewards
		}
		value := policy.evaluate(rewards, visits)
		if value > maxValue {
			maxValue = value
			maxMove = move
		}
	}
	// TODO: copy state or pass arg by value?
	return d.children[maxMove], state.Play(maxMove)
}

func (d *decision) applyLoss() {
	d.Lock()
	defer d.Unlock()

	d.rewards += LOSS
	d.visits++
}

func (d *decision) stats() (player string, rewards float64, visits int) {
	d.RLock()
	defer d.RUnlock()

	return d.player, d.rewards, d.visits
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

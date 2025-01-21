package searcher

import (
	// "fmt"
	"math"
	"math/rand"
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
	visits   float64
}

func newDecision(parent Node, state game.State) *decision {
	moves := state.LegalMoves()
	movesCopy := make([]game.Move, len(moves))
	copy(movesCopy, moves)

	// TODO: remove
	// var hash game.StateHash
	// if _, ok := parent.(*chance); ok {
	// 	hash = state.Hash()
	// }

	return &decision{
		parent:   parent,
		player:   state.Player(),
		moves:    movesCopy,
		children: make(map[game.Move]Node, len(moves)),
		hash:     state.Hash(),
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

	if len(d.moves) == 0 && len(d.children) == 0 {
		// Terminal
		return d, state, false
	}

	var child Node
	selected := false
	if len(d.moves) > 0 { // Expand node with an unexplored move
		child, state = d.expands(state)
	} else { // Select a child of fully expanded node
		child, state = d.selects(state)
		selected = true
	}

	child.applyLoss()
	return child, state, selected
}

func (d *decision) expands(state game.State) (Node, game.State) {
	// Expand a random move
	index := rand.Intn(len(d.moves)) // move := d.moves[0]
	move := d.moves[index]

	newState := state.Play(move)

	var child Node
	if move.IsStochastic() {
		child = newChance(d)
	} else {
		child = newDecision(d, newState)
	}
	d.children[move] = child

	// Remove the move from unexplored moves
	d.moves[index] = d.moves[0]
	d.moves = d.moves[1:]

	return child, newState
}

func (d *decision) selects(state game.State) (Node, game.State) {
	if len(d.children) == 0 {
		panic("no children")
	}

	parentVisits := d.visits
	// When the concurrency level is high or the number of legal moves is low, selection could happen when the parent is fully expanded but results are not yet backpropagated. In this case, use the number of children as the parent visit count and the child's virtual loss or backedup result as the child visit count.
	if parentVisits == 0 {
		parentVisits = float64(len(d.children))
	}

	policy := newUCT(CSquared, parentVisits)
	maxValue := math.Inf(-1)
	var maxMove game.Move
	// Iterate over children in random order
	for move, child := range d.children {
		player, rewards, visits := child.stats()
		if visits == 0 {
			// Child should have virtual loss or backpropagated result
			panic("unexplored child node (0 visits)")
		}
		// Maximize my chance of winning or if turn changes, minimize the opponent's
		if player != d.player {
			rewards = -rewards // Negate opponent's rewards
		}
		value := policy.evaluate(rewards, visits)
		if value > maxValue {
			maxValue = value
			maxMove = move
		}
	}
	return d.children[maxMove], state.Play(maxMove)
}

func (d *decision) applyLoss() {
	d.Lock()
	defer d.Unlock()

	d.rewards += Loss
	d.visits++
}

func (d *decision) stats() (player string, rewards float64, visits float64) {
	d.RLock()
	defer d.RUnlock()

	return d.player, d.rewards, d.visits
}

func (d *decision) Backup(player string, score float64) Node {
	d.Lock()
	defer d.Unlock()

	if d.parent != nil { // Virtual loss not applied on root node
		d.reverseLoss()
	}

	d.rewards += computeReward(player, score, d.player)
	d.visits++

	return d.parent
}

func (d *decision) reverseLoss() {
	d.rewards -= Loss
	d.visits--
}

func (d *decision) Policy() map[game.Move]float64 {
	d.RLock()
	defer d.RUnlock()

	visits := make(map[game.Move]float64, len(d.children))
	for move, child := range d.children {
		_, _, visits[move] = child.stats()
	}

	return visits
}

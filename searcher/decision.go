package searcher

import (
	"fmt"
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
	phase    game.Phase
	rewards  float64
	visits   float64
}

func newDecision(parent Node, state game.State) *decision {
	gs := state.(*game.GameState)

	moves := gs.LegalMoves()
	var hash game.StateHash
	if _, ok := parent.(*chance); ok {
		hash = gs.Hash()
	}

	return &decision{
		parent:   parent,
		player:   gs.Player(),
		moves:    moves,
		children: make(map[game.Move]Node, len(moves)),
		hash:     hash,
		rewards:  0,
		visits:   1,
		phase:    gs.Phase,
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
	if len(d.moves) > 0 {
		child, state = d.expands(state)
	} else {
		child, state = d.selects(state)
		selected = true
	}

	if child == d {
		// Means we “skipped” the move => do NOT call child.applyLoss()
		// Because that would cause a double lock.
		return d, state, false
	}

	child.applyLoss()
	return child, state, selected
}

func (d *decision) expands(state game.State) (Node, game.State) {
	gs := state.(*game.GameState)
	move := d.moves[0]

	// If the move is obviously invalid for this phase, skip it:
	if !game.IsMoveValidForPhase(gs.Phase, move) {
		fmt.Printf("[MCTS expands] Skipping invalid move: Phase=%d, ActionType=%d\n",
			gs.Phase, move.(*game.GameMove).ActionType)

		// remove the invalid move from the unexplored moves
		d.moves = d.moves[1:]

		// Also increment visits to avoid re-selecting the same node
		d.visits++

		return d, state
	}
	newState := state.Play(move)

	var child Node
	if move.IsStochastic() {
		child = newChance(d)
	} else {
		child = newDecision(d, newState)
	}
	d.children[move] = child
	d.moves = d.moves[1:]
	return child, newState
}

func (d *decision) selects(state game.State) (Node, game.State) {
	if len(d.children) == 0 {
		panic("no children")
	}

	if d.visits == 0 {
		panic("unexplored parent node (0 visits)")
	}

	policy := newUCT(CSquared, d.visits)
	maxValue := math.Inf(-1)
	var maxMove game.Move
	for move, child := range d.children {
		player, rewards, visits := child.stats()
		if visits == 0 {
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

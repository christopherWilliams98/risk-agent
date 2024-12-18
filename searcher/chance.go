package searcher

import (
	"risk/game"
	"sync"
)

type chance struct {
	sync.RWMutex
	parent   Node
	player   string
	children []*decision
	rewards  float64
	visits   float64
}

func newChance(parent *decision) *chance {
	return &chance{
		parent:  parent,
		player:  parent.player,
		rewards: 0,
		visits:  0,
	}
}

func (c *chance) SelectOrExpand(state game.State) (Node, game.State, bool) {
	c.Lock()
	defer c.Unlock()

	// Select if explored outcome
	selected := true
	child := c.selects(state)
	// Expand if unexplored outcome
	if child == nil {
		child = c.expands(state)
		selected = false
	}

	child.applyLoss()
	return child, state, selected
}

func (c *chance) selects(state game.State) *decision {
	expected := state.Hash()
	for _, child := range c.children {
		if child.hash == expected {
			return child
		}
	}
	return nil
}

func (c *chance) expands(state game.State) *decision {
	child := newDecision(c, state)
	c.children = append(c.children, child)
	return child
}

func (c *chance) applyLoss() {
	c.Lock()
	defer c.Unlock()

	c.rewards += Loss
	c.visits++
}

func (c *chance) stats() (player string, rewards float64, visits float64) {
	c.RLock()
	defer c.RUnlock()

	return c.player, c.rewards, c.visits
}

func (c *chance) Backup(player string, score float64) Node {
	c.Lock()
	defer c.Unlock()

	c.reverseLoss()

	c.rewards += computeReward(player, score, c.player)
	c.visits++

	return c.parent
}

func (c *chance) reverseLoss() {
	c.rewards -= Loss
	c.visits--
}

func (c *chance) Policy() map[game.Move]float64 {
	// Chance nodes do not have a policy (stochastic outcomes)
	return nil
}

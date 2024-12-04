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
	visits   int
}

func newChance(state game.State, parent Node) *chance {
	return &chance{
		parent:  parent,
		player:  state.Player(),
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

	child.ApplyLoss()
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
	child := newDecision(state, c)
	c.children = append(c.children, child)
	return child
}

func (c *chance) ApplyLoss() {
	c.Lock()
	defer c.Unlock()

	c.rewards += LOSS
	c.visits++
}

func (c *chance) Score(normalizer float64) float64 {
	c.RLock()
	defer c.RUnlock()

	if c.visits == 0 {
		panic("cannot compute score for child with 0 visits")
	}

	return uct(c.rewards, c.visits, normalizer)
}

func (c *chance) Backup(winner string) Node {
	c.Lock()
	defer c.Unlock()

	c.reverseLoss()

	c.rewards += computeReward(winner, c.player)
	c.visits++

	return c.parent
}

func (c *chance) reverseLoss() {
	c.rewards -= LOSS
	c.visits--
}

func (c *chance) Visits() int {
	c.RLock()
	defer c.RUnlock()

	return c.visits
}

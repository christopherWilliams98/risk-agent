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

func newChance(parent Node, state game.State) *chance {
	return &chance{
		parent:  parent,
		player:  state.Player(),
		rewards: 0,
		visits:  0,
	}
}

func (c *chance) PickChild(state game.State) (Node, game.State, bool) {
	c.Lock()
	defer c.Unlock()

	child := c.findChild(state)
	if child != nil {
		child.applyLoss()
		return child, state, false
	}

	// Expand the chance node for the new state
	child = c.addChild(state)
	child.applyLoss()
	return child, state, true
}

func (c *chance) findChild(state game.State) *decision {
	equals := func(a, b map[string]any) bool {
		if len(a) != len(b) {
			return false
		}
		for k, v := range a {
			if b[k] != v {
				return false
			}
		}
		return true
	}

	expected := state.Delta()
	for _, child := range c.children {
		if equals(child.delta, expected) {
			return child
		}
	}
	return nil
}

func (c *chance) addChild(state game.State) *decision {
	child := newDecision(c, state)
	c.children = append(c.children, child)
	return child
}

func (c *chance) applyLoss() {
	c.Lock()
	defer c.Unlock()

	c.rewards += LOSS
	c.visits++
}

func (c *chance) score(normalizer float64) float64 {
	c.RLock()
	defer c.RUnlock()

	return ucb1(c.rewards, c.visits, normalizer)
}

func (c *chance) Update(rewarder func(string) float64) Node {
	c.Lock()
	defer c.Unlock()

	// Reverse virtual loss
	c.rewards -= LOSS
	c.visits--

	c.rewards += rewarder(c.player)
	c.visits++

	return c.parent
}

func (c *chance) Value() int {
	c.RLock()
	defer c.RUnlock()

	return c.visits
}

package searcher

import (
	"reflect"
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

func (c *chance) SelectOrExpand(state game.State) (Node, game.State, bool) {
	c.Lock()
	defer c.Unlock()

	child := c.findChild(state)
	if child != nil {
		child.ApplyLoss()
		return child, state, false
	}

	// Expand the chance node for the new state
	child = c.addChild(state)
	child.ApplyLoss()
	return child, state, true
}

func (c *chance) findChild(state game.State) *decision {
	expected := state.Delta()
	for _, child := range c.children {
		// TODO: custom/more efficient comparison??
		if reflect.DeepEqual(child.delta, expected) {
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

func (c *chance) ApplyLoss() {
	c.Lock()
	defer c.Unlock()

	c.rewards += LOSS
	c.visits++
}

func (c *chance) Score(normalizer float64) float64 {
	c.RLock()
	defer c.RUnlock()

	return ucb1(c.rewards, c.visits, normalizer)
}

func (c *chance) Backup(rewarder func(string) float64) Node {
	c.Lock()
	defer c.Unlock()

	c.reverseLoss()

	c.rewards += rewarder(c.player)
	c.visits++

	return c.parent
}

func (c *chance) reverseLoss() {
	c.rewards -= LOSS
	c.visits--
}

func (c *chance) Value() int {
	c.RLock()
	defer c.RUnlock()

	return c.visits
}

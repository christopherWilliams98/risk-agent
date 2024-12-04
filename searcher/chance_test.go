package searcher

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChanceSelectOrExpand(t *testing.T) {
	t.Run("expanding a new stochastic outcome", func(t *testing.T) {
		// Setup a node with a known outcome child and a different outcome state
		child := &decision{hash: 1}
		node := &chance{
			children: []*decision{child},
		}
		state := mockState{player: "player1", hash: 2}

		gotChild, gotState, gotSelected := node.SelectOrExpand(state)

		require.IsType(t, &decision{}, gotChild, "Child should be a decision node")
		require.Equal(t, LOSS, gotChild.(*decision).rewards, "Child should apply a temporary loss")
		require.Equal(t, 1, gotChild.(*decision).visits, "Child should apply a temporary loss")
		require.NotEqual(t, child, gotChild, "Node should expand with a new child")
		require.Equal(t, 2, len(node.children), "Node should expand with a new child")
		require.Equal(t, state, gotState, "State should not change")
		require.False(t, gotSelected, "Node should expand with a new child")
	})

	t.Run("selecting an existing stochastic outcome", func(t *testing.T) {
		// Setup a node with a known outcome child and a different outcome state
		child := &decision{hash: 1}
		node := &chance{
			children: []*decision{child},
		}

		// Expand first then select
		state := mockState{player: "player1", hash: 2}
		otherChild, _, _ := node.SelectOrExpand(state)
		state = mockState{player: "player1", hash: 2}
		gotChild, gotState, gotSelected := node.SelectOrExpand(state)

		require.IsType(t, &decision{}, gotChild, "Child should be a decision node")
		require.Equal(t, LOSS*2, gotChild.(*decision).rewards, "Child should apply 2 temporary losses")
		require.Equal(t, 2, gotChild.(*decision).visits, "Child should apply 2 temporary losses")
		require.Equal(t, otherChild, gotChild, "Node should select an existing child")
		require.Equal(t, 2, len(node.children), "Node should select an existing child")
		require.Equal(t, state, gotState, "State should not change")
		require.True(t, gotSelected, "Node should select an existing child")
	})
}

func TestChanceBackup(t *testing.T) {
	t.Run("recording win", func(t *testing.T) {
		// Setup a node with a virtual loss
		parent := &decision{}
		node := &chance{
			parent:  parent,
			player:  "player1",
			rewards: LOSS,
			visits:  1,
		}

		got := node.Backup("player1")

		require.Equal(t, parent, got, "Should return the parent node")
		require.Equal(t, WIN, node.rewards, "Should reverse virtual loss and add a win")
		require.Equal(t, 1, node.visits, "Should reverse virtual loss and add a visit")
	})

	t.Run("recording loss", func(t *testing.T) {
		// Setup a node with a virtual loss
		parent := &decision{}
		node := &chance{
			parent:  parent,
			player:  "player1",
			rewards: LOSS,
			visits:  1,
		}

		got := node.Backup("player2")

		require.Equal(t, parent, got, "Should return the parent node")
		require.Equal(t, LOSS, node.rewards, "Should reverse virtual loss and add a loss")
		require.Equal(t, 1, node.visits, "Should reverse virtual loss and add a visit")
	})
}

// func TestChanceScore(t *testing.T) {
// 	state := mockState{player: "player1"}
// 	c := newChance(state, nil)

// 	// Should panic with 0 visits
// 	func() {
// 		defer func() {
// 			if r := recover(); r == nil {
// 				t.Error("Expected panic for score calculation with 0 visits")
// 			}
// 		}()
// 		c.Score(1.0)
// 	}()

// 	// Test normal score calculation
// 	c.visits = 10
// 	c.rewards = 5.0
// 	normalizer := 2.0

// 	score := c.Score(normalizer)
// 	if score <= 0 {
// 		t.Error("Expected positive score for node with positive rewards")
// 	}
// }

// func TestChanceVisits(t *testing.T) {
// 	c := newChance(mockState{}, nil)

// 	if c.Visits() != 0 {
// 		t.Error("New node should have 0 visits")
// 	}

// 	c.visits = 5
// 	if c.Visits() != 5 {
// 		t.Error("Visits() not returning correct visit count")
// 	}
// }

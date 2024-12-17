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
		require.Equal(t, 1.0, gotChild.(*decision).visits, "Child should apply a temporary loss")
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
		require.Equal(t, 2.0, gotChild.(*decision).visits, "Child should apply 2 temporary losses")
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
		require.Equal(t, 1.0, node.visits, "Should reverse virtual loss and add a visit")
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
		require.Equal(t, 1.0, node.visits, "Should reverse virtual loss and add a visit")
	})
}

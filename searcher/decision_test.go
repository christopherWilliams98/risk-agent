package searcher

import (
	"risk/game"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

/**
Tests parallel MCTS (tree parallelization with virtual loss) on decision nodes (
deterministic moves only)
sequential:
- selection:
	- happy path: fully expanded node -> max UCB child + loss, child state
	- otherwise: skip
	- edge case: terminal node -> same node, same state
- expansion:
	- happy path: expandable node -> new added child + loss, child state
	- otherwise: skip
	- edge case: terminal node -> same node, same state
- rollout:
	- happy path: state -> terminal state, winner
		- mock State and State.GetMoves() and State.Play()
	- edge case: terminal state -> same state, winner
- backup:
	- happy path: winner -> [new added child, 1st selected child]: reverse loss, visits++, update rewards; [root] visits++, update rewards
concurrent: 3 race conditions
- shared expansion
- shared backup
- shared selection + backup
*/

func TestDecisionSelectOrExpand(t *testing.T) {
	t.Run("selecting fully expanded node (all deterministic moves explored)", func(t *testing.T) {
		maxMove := mockMove{id: 1}
		maxChild := &decision{rewards: 1, visits: 1}
		otherChild := &decision{rewards: 0, visits: 1}
		node := &decision{
			unexplored: []game.Move{},
			explored:   []game.Move{mockMove{id: 0}, maxMove},
			children:   []Node{otherChild, maxChild},
			rewards:    1,
			visits:     2,
		}
		state := mockState{}

		gotChild, gotState, gotSelected := node.SelectOrExpand(state)

		require.Equal(t, maxChild, gotChild, "Node should select child with max policy value")
		require.IsType(t, &decision{}, gotChild,
			"Child should be a decision node")
		require.Equal(t, 1+Loss, gotChild.(*decision).rewards, "Child should apply a temporary loss")
		require.Equal(t, 2.0, gotChild.(*decision).visits,
			"Child should apply a temporary loss")
		require.Equal(t, []game.Move{maxMove}, gotState.(mockState).played, "State should update by the move to the max policy child")
		require.True(t, gotSelected, "Node should perform selection")
		require.Equal(t, 1.0, node.rewards, "Node stats should not change")
		require.Equal(t, 2.0, node.visits, "Node stats should not change")
	})

	t.Run("selecting fully expanded node (all stochastic moves explored)", func(t *testing.T) {
		maxMove := mockMove{id: 1, stochastic: true}
		maxChild := &chance{rewards: 1, visits: 1}
		otherChild := &chance{rewards: 0, visits: 1}
		node := &decision{
			unexplored: []game.Move{},
			explored:   []game.Move{mockMove{id: 0, stochastic: true}, maxMove},
			children:   []Node{otherChild, maxChild},
			rewards:    1,
			visits:     2,
		}
		state := mockState{}

		gotChild, gotState, gotSelected := node.SelectOrExpand(state)

		require.Equal(t, maxChild, gotChild, "Node should select child with max policy value")
		require.IsType(t, &chance{}, gotChild,
			"Child should be a chance node")
		require.Equal(t, 1+Loss, gotChild.(*chance).rewards, "Child should apply a temporary loss")
		require.Equal(t, 2.0, gotChild.(*chance).visits,
			"Child should apply a temporary loss")
		require.Equal(t, []game.Move{maxMove}, gotState.(mockState).played, "State should update by the move to the max policy child")
		require.True(t, gotSelected, "Node should perform selection")
		require.Equal(t, 1.0, node.rewards, "Node stats should not change")
		require.Equal(t, 2.0, node.visits, "Node stats should not change")
	})

	t.Run("selecting fully expanded node (all deterministic moves explored) with turn change", func(t *testing.T) {
		minMove := mockMove{id: 1}
		minChild := &decision{player: "player2", rewards: 0, visits: 1}
		otherChild := &decision{player: "player2", rewards: 1, visits: 1}
		node := &decision{
			player:     "player1",
			unexplored: []game.Move{},
			explored:   []game.Move{mockMove{id: 0}, minMove},
			children:   []Node{otherChild, minChild},
			rewards:    1,
			visits:     2,
		}
		state := mockState{}

		gotChild, gotState, gotSelected := node.SelectOrExpand(state)

		require.Equal(t, minChild, gotChild, "Node should select child with max policy value that minimizes opponent rewards")
		require.IsType(t, &decision{}, gotChild, "Child should be a decision node")
		require.Equal(t, Loss, gotChild.(*decision).rewards, "Child should apply a temporary loss")
		require.Equal(t, 2.0, gotChild.(*decision).visits,
			"Child should apply a temporary loss")
		require.Equal(t, []game.Move{minMove}, gotState.(mockState).played, "State should update by the move to the max policy child")
		require.True(t, gotSelected, "Node should perform selection")
		require.Equal(t, 1.0, node.rewards, "Node stats should not change")
		require.Equal(t, 2.0, node.visits, "Node stats should not change")
	})

	t.Run("selecting fully expanded node (all stochastic moves explored) with turn change", func(t *testing.T) {
		minMove := mockMove{id: 1, stochastic: true}
		minChild := &chance{player: "player2", rewards: 0, visits: 1}
		otherChild := &chance{player: "player2", rewards: 1, visits: 1}
		node := &decision{
			player:     "player1",
			unexplored: []game.Move{},
			explored:   []game.Move{mockMove{id: 0, stochastic: true}, minMove},
			children:   []Node{otherChild, minChild},
			rewards:    1,
			visits:     2,
		}
		state := mockState{}

		gotChild, gotState, gotSelected := node.SelectOrExpand(state)

		require.Equal(t, minChild, gotChild, "Node should select child with max policy value that minimizes opponent rewards")
		require.IsType(t, &chance{}, gotChild, "Child should be a chance node")
		require.Equal(t, Loss, gotChild.(*chance).rewards, "Child should apply a temporary loss")
		require.Equal(t, 2.0, gotChild.(*chance).visits,
			"Child should apply a temporary loss")
		require.Equal(t, []game.Move{minMove}, gotState.(mockState).played, "State should update by the move to the max policy child")
		require.True(t, gotSelected, "Node should perform selection")
		require.Equal(t, 1.0, node.rewards, "Node stats should not change")
		require.Equal(t, 2.0, node.visits, "Node stats should not change")
	})

	t.Run("expanding node with unexplored deterministic moves", func(t *testing.T) {
		unexploredMove := mockMove{id: 1}
		node := &decision{
			unexplored: []game.Move{unexploredMove},
			explored:   []game.Move{mockMove{id: 0}},
			children:   []Node{&decision{rewards: 1, visits: 1}},
			visits:     1,
		}
		state := mockState{moves: []game.Move{}}

		gotChild, gotState, gotSelected := node.SelectOrExpand(state)

		require.IsType(t, &decision{}, gotChild,
			"Child should be a decision node")
		require.Equal(t, Loss, gotChild.(*decision).rewards, "Child should apply a temporary loss")
		require.Equal(t, 1.0, gotChild.(*decision).visits,
			"Child should apply a temporary loss")
		require.Equal(t, 2, len(node.children), "Node should add a new child")
		require.Equal(t, []game.Move{unexploredMove},
			gotState.(mockState).played, "State should update by the move to the unexplored child")
		require.False(t, gotSelected, "Node should perform expansion")
	})

	t.Run("expanding node with unexplored stochastic moves", func(t *testing.T) {
		unexploredMove := mockMove{id: 1, stochastic: true}
		node := &decision{
			unexplored: []game.Move{unexploredMove},
			explored:   []game.Move{mockMove{id: 0, stochastic: true}},
			children:   []Node{&chance{rewards: 1, visits: 1}},
			visits:     1,
		}
		state := mockState{moves: []game.Move{}}

		gotChild, gotState, gotSelected := node.SelectOrExpand(state)

		require.IsType(t, &chance{}, gotChild,
			"Child should be a chance node")
		require.Equal(t, Loss, gotChild.(*chance).rewards, "Child should apply a temporary loss")
		require.Equal(t, 1.0, gotChild.(*chance).visits,
			"Child should apply a temporary loss")
		require.Equal(t, []game.Move{unexploredMove},
			gotState.(mockState).played, "State should update by the move to the unexplored child")
		require.False(t, gotSelected, "Node should perform expansion")
		require.Equal(t, 2, len(node.children), "Node should add a new child")
	})

	t.Run("stagnating on terminal node", func(t *testing.T) {
		node := &decision{}
		state := mockState{}

		gotChild, gotState, gotSelected := node.SelectOrExpand(state)

		require.Equal(t, node, gotChild, "Should return the same node")
		require.Equal(t, mockState{}, gotState, "Should return the same state")
		require.False(t, gotSelected, "Should not select any child or expand")
	})
}

func TestDecisionBackup(t *testing.T) {
	t.Run("recording win on root node", func(t *testing.T) {
		// Setup a root node with no parent
		node := &decision{
			parent:  nil,
			player:  "player1",
			rewards: 0,
			visits:  0,
		}

		got := node.Backup("player1", Win)

		require.Nil(t, got, "Should return no parent")
		require.Equal(t, Win, node.rewards, "Should apply a win reward")
		require.Equal(t, 1.0, node.visits, "Should add a visit")
	})

	t.Run("recording win on deterministic outcome node", func(t *testing.T) {
		// Setup a node with decision parent and a virtual loss
		parent := &decision{}
		node := &decision{
			parent:  parent,
			player:  "player1",
			rewards: Loss,
			visits:  1,
		}

		got := node.Backup("player1", Win)

		require.Equal(t, parent, got, "Should return the parent node")
		require.Equal(t, Win, node.rewards, "Should reverse virtual loss and add a win")
		require.Equal(t, 1.0, node.visits,
			"Should reverse virtual loss and add a visit")
	})

	t.Run("recording win on stochastic outcome node", func(t *testing.T) {
		// Setup a node with chance parent and a virtual loss
		parent := &chance{}
		node := &decision{
			parent:  parent,
			player:  "player1",
			rewards: Loss,
			visits:  1,
		}

		got := node.Backup("player1", Win)

		require.Equal(t, parent, got, "Should return the parent node")
		require.Equal(t, Win, node.rewards, "Should reverse virtual loss and add a win")
		require.Equal(t, 1.0, node.visits,
			"Should reverse virtual loss and add a visit")
	})

	t.Run("recording loss on deterministic outcome node", func(t *testing.T) {
		// Setup a node with decision parent and a virtual loss
		parent := &decision{}
		node := &decision{
			parent:  parent,
			player:  "player1",
			rewards: Loss,
			visits:  1,
		}

		got := node.Backup("player2", Win)

		require.Equal(t, parent, got, "Should return the parent node")
		require.Equal(t, Loss, node.rewards, "Should reverse virtual loss and add a loss")
		require.Equal(t, 1.0, node.visits,
			"Should reverse virtual loss and add a visit")
	})

	t.Run("recording loss on stochastic outcome node", func(t *testing.T) {
		// Setup a node with chance parent and a virtual loss
		parent := &chance{}
		node := &decision{
			parent:  parent,
			player:  "player1",
			rewards: Loss,
			visits:  1,
		}

		got := node.Backup("player2", Win)

		require.Equal(t, parent, got, "Should return the parent node")
		require.Equal(t, Loss, node.rewards, "Should reverse virtual loss and add a loss")
		require.Equal(t, 1.0, node.visits, "Should reverse virtual loss and add a visit")
	})
}

func TestDecisionRaceConditions(t *testing.T) {
	t.Run("concurrent expansion", func(t *testing.T) {
		// Setup a node with 2 unexplored moves
		node := &decision{
			unexplored: []game.Move{mockMove{id: 0}, mockMove{id: 1}},
			explored:   []game.Move{},
			children:   []Node{},
			rewards:    0,
			visits:     0,
		}
		baseState := mockState{moves: []game.Move{}}

		// Launch two goroutines to expand simultaneously
		var wg sync.WaitGroup
		type result struct {
			child    Node
			state    mockState
			selected bool
		}
		var got [2]result

		for i := 0; i < 2; i++ {
			wg.Add(1)
			i := i
			go func() {
				defer wg.Done()
				// Each goroutine gets its own copy of state
				state := mockState{moves: baseState.moves}
				gotChild, gotState, gotSelected := node.SelectOrExpand(state)
				got[i] = result{gotChild, gotState.(mockState), gotSelected}
			}()
		}
		wg.Wait()

		// Verify results
		require.Equal(t, 2, len(node.children), "Node should have two children")

		// Each goroutine should have:
		// - Received a decision node as child
		// - Applied virtual loss to that child
		// - Marked the expansion as successful
		for i := 0; i < 2; i++ {
			require.IsType(t, &decision{}, got[i].child,
				"Child should be a decision node")
			require.Equal(t, Loss, got[i].child.(*decision).rewards,
				"Child should apply a temporary loss")
			require.Equal(t, 1.0, got[i].child.(*decision).visits,
				"Child should apply a temporary loss")
			require.False(t, got[i].selected, "Node should be expanded")
			require.Contains(t, []game.Move{mockMove{id: 0}, mockMove{id: 1}}, got[i].state.played[0],
				"Node should expand with a legal move")
		}

		// Both goroutines should have expanded different moves
		require.NotEqual(t, got[0].state.played[0], got[1].state.played[0],
			"Node should expand with different moves")
	})

	t.Run("concurrent backup", func(t *testing.T) {
		// Setup a node with 2 virtual losses
		parent := &decision{}
		node := &decision{
			parent:  parent, // Non-root
			player:  "player1",
			rewards: Loss * 2, // 2 virtual losses
			visits:  2,        // 2 virtual losses
		}

		// Launch multiple goroutines to backup simultaneously
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				got := node.Backup("player1", Win)
				require.Equal(t, parent, got,
					"Should return the parent node")
			}()
		}
		wg.Wait()

		// Verify node stats
		require.Equal(t, Win*2, node.rewards,
			"Node should reverse virtual losses and add two wins")
		require.Equal(t, 2.0, node.visits,
			"Node should reverse virtual losses and add two visits")
	})

	t.Run("concurrent selection and backup", func(t *testing.T) {
		// Setup a node with a child and a virtual loss
		parent := &decision{}
		node := &decision{
			parent:  parent, // Non-root
			player:  "player1",
			rewards: Loss, // Virtual loss
			visits:  3,    // Virtual loss
		}
		child := &decision{
			parent:  node,
			rewards: 0,
			visits:  1,
		}
		move := mockMove{id: 0}
		node.explored = []game.Move{move}
		node.children = []Node{child}
		state := mockState{moves: []game.Move{}}

		// Launch selection and backup simultaneously
		var wg sync.WaitGroup
		wg.Add(2)

		// Goroutine 1: Select the child
		go func() {
			defer wg.Done()
			gotChild, gotState, gotSelected := node.SelectOrExpand(state)
			require.Equal(t, child, gotChild,
				"Node should select the child")
			require.Equal(t, move, gotState.(mockState).played[0],
				"State should update by the move to the child")
			require.True(t, gotSelected, "Node should perform selection")
		}()

		// Goroutine 2: Backup through the node
		go func() {
			defer wg.Done()
			got := node.Backup("player1", Win)
			require.Equal(t, parent, got,
				"Node should return its parent")
		}()

		wg.Wait()

		// Verify final state reflects selection
		require.Equal(t, Loss, child.rewards,
			"Child should apply a temporary loss")
		require.Equal(t, 2.0, child.visits,
			"Child should apply a temporary loss")
		// Verify final state reflects backup
		require.Equal(t, Win, node.rewards,
			"Node should reverse virtual loss and add a win")
		require.Equal(t, 3.0, node.visits,
			"Node should reverse virtual loss and add a visit")
	})
}

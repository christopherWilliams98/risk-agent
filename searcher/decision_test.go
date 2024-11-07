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
Spec:
sequential:
- selection:
	- happy path: fully expanded node -> unvisited child or max UCB child
+ loss, child state
	- otherwise: skip
	- edge case: terminal node -> same node
- expansion:
	- happy path: expandable node -> new added child + loss, child state
	- otherwise: skip
	- edge case: terminal node -> same node
- rollout:
	- happy path: state -> terminal state, winner
		- mock State and State.GetMoves() and State.Play()
	- edge case: terminal state -> same state, winner
- backup:
	- happy path: new added child, winner, rewarder -> [new added child,
root] reverse loss, visits++ & add rewards
concurrent: 3 race conditions - shared expansion, shared backup,
shared selection + backup
*/

type MockMove struct {
	id int
}

func (m MockMove) IsDeterministic() bool {
	return true
}

type MockState struct {
	player string
	moves  []game.Move
	played []game.Move
}

func (m MockState) Player() string {
	return m.player
}

func (m MockState) LegalMoves() []game.Move {
	return m.moves
}

func (m MockState) Play(move game.Move) game.State {
	return MockState{played: append(m.played, move)}
}

func (m MockState) Delta() map[string]any {
	return nil
}

func (m MockState) Winner() string {
	return ""
}

func TestSelectOrExpand(t *testing.T) {
	t.Run("fully expanded node", func(t *testing.T) {
		node := &decision{
			moves:    []game.Move{MockMove{id: 0}, MockMove{id: 1}},
			children: []Node{&decision{rewards: 0, visits: 1}, &decision{rewards: 1, visits: 1}},
			rewards:  1,
			visits:   2,
		}
		state := MockState{}

		gotChild, gotState, gotAdded := node.SelectOrExpand(state)

		require.Equal(t, node.children[1], gotChild, "Node should select child with max policy value")
		require.IsType(t, &decision{}, gotChild, "Child should be a decision node")
		require.Equal(t, 1+LOSS, gotChild.(*decision).rewards, "Child should apply a temporary loss")
		require.Equal(t, 2, gotChild.(*decision).visits, "Child should apply a temporary loss")
		require.Equal(t, []game.Move{node.moves[1]}, gotState.(MockState).played, "State should update by the move to the max policy child")
		require.False(t, gotAdded, "Node should add no child")
		require.Equal(t, 1.0, node.rewards, "Node stats should not change")
		require.Equal(t, 2, node.visits, "Node stats should not change")
	})

	t.Run("expandable node", func(t *testing.T) {
		node := &decision{
			moves:    []game.Move{MockMove{id: 0}, MockMove{id: 1}},
			children: []Node{&decision{rewards: 1, visits: 1}},
			visits:   1,
		}
		state := MockState{moves: []game.Move{MockMove{id: 0}, MockMove{id: 1}}}

		gotChild, gotState, gotAdded := node.SelectOrExpand(state)

		require.IsType(t, &decision{}, gotChild, "Child should be a decision node")
		require.Equal(t, LOSS, gotChild.(*decision).rewards, "Child should apply a temporary loss")
		require.Equal(t, 1, gotChild.(*decision).visits, "Child should apply a temporary loss")
		require.Equal(t, []game.Move{node.moves[1]},
			gotState.(MockState).played, "State should update by the move to the child")
		require.True(t, gotAdded, "Node should add a new child")
		require.Equal(t, 2, len(node.children), "Node should add a new child")
	})

	t.Run("terminal node", func(t *testing.T) {
		node := &decision{}
		state := MockState{}

		child, childState, added := node.SelectOrExpand(state)

		require.Equal(t, node, child, "Should return the same node")
		require.Equal(t, MockState{}, childState, "Should return the same state")
		require.False(t, added, "Should not add a child")
	})
}

func TestUpdate(t *testing.T) {
	t.Run("winning root node", func(t *testing.T) {
		node := &decision{
			parent:  nil,
			player:  "player1",
			rewards: 0,
			visits:  0,
		}

		got := node.Backup(rewarder("player1"))

		require.Nil(t, got, "Should return no parent")
		require.Equal(t, WIN, node.rewards, "Should apply a win reward")
		require.Equal(t, 1, node.visits, "Should add a visit")
	})

	t.Run("winning non-root node", func(t *testing.T) {
		parent := &decision{}
		node := &decision{
			parent:  parent,
			player:  "player1",
			rewards: LOSS,
			visits:  1,
		}

		got := node.Backup(rewarder("player1"))

		require.Equal(t, parent, got, "Should return the parent node")
		require.Equal(t, WIN, node.rewards, "Should reverse virtual loss and add a win")
		require.Equal(t, 1, node.visits, "Should reverse virtual loss and add a visit")
	})

	t.Run("losing non-root node", func(t *testing.T) {
		parent := &decision{}
		node := &decision{
			parent:  parent,
			player:  "player1",
			rewards: LOSS,
			visits:  1,
		}

		got := node.Backup(rewarder("player2"))

		require.Equal(t, parent, got, "Should return the parent node")
		require.Equal(t, LOSS, node.rewards, "Should reverse virtual loss and add a loss")
		require.Equal(t, 1, node.visits, "Should reverse virtual loss and add a visit")
	})
}

// TODO: test FindBestMove()

func TestRaceConditions(t *testing.T) {
	t.Run("concurrent expansion", func(t *testing.T) {
		// Setup a node with 2 possible moves but no children yet
		node := &decision{
			moves:    []game.Move{MockMove{id: 0}, MockMove{id: 1}},
			children: []Node{},
			rewards:  0,
			visits:   0,
		}
		baseState := MockState{moves: []game.Move{MockMove{id: 0}, MockMove{id: 1}}}

		// Launch two goroutines to expand simultaneously
		var wg sync.WaitGroup
		type result struct {
			child Node
			state game.State
			added bool
		}
		var results [2]result
		var states [2]MockState

		for i := 0; i < 2; i++ {
			wg.Add(1)
			i := i
			go func() {
				defer wg.Done()
				// Each goroutine gets its own copy of state
				state := MockState{moves: baseState.moves}
				child, childState, added := node.SelectOrExpand(state)
				results[i] = result{child, childState, added}
				states[i] = childState.(MockState)
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
			require.IsType(t, &decision{}, results[i].child,
				"Child should be a decision node")
			require.Equal(t, LOSS, results[i].child.(*decision).rewards,
				"Child should apply a temporary loss")
			require.Equal(t, 1, results[i].child.(*decision).visits,
				"Child should apply a temporary loss")
			require.True(t, results[i].added,
				"Child should be added")
			require.Contains(t, node.moves, states[i].played[0],
				"Node should expand with a legal move")
		}

		// Both goroutines should have expanded different moves
		require.NotEqual(t, states[0].played[0], states[1].played[0],
			"Node should expand with different moves")
	})

	t.Run("concurrent backup", func(t *testing.T) {
		// Setup a node with virtual loss
		parent := &decision{}
		node := &decision{
			parent:  parent, // Not root
			player:  "player1",
			rewards: LOSS * 2, // Virtual loss applied
			visits:  2,        // Visit from virtual loss
		}

		// Launch multiple goroutines to backup simultaneously
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				got := node.Backup(rewarder("player1"))
				require.Equal(t, parent, got,
					"Should return the parent node")
			}()
		}
		wg.Wait()

		// Verify node stats
		require.Equal(t, WIN*2, node.rewards,
			"Node should reverse virtual loss and add two wins")
		require.Equal(t, 2, node.visits,
			"Node should reverse virtual loss and add two visits")
	})

	t.Run("concurrent selection and backup", func(t *testing.T) {
		// Setup a node with a child
		parent := &decision{}
		node := &decision{
			parent:  parent, // Not root
			player:  "player1",
			moves:   []game.Move{MockMove{id: 0}},
			rewards: LOSS, // Virtual loss applied
			visits:  3,    // Visit from virtual loss
		}
		child := &decision{
			parent:  node,
			rewards: 0,
			visits:  1,
		}
		node.children = []Node{child}
		state := MockState{moves: []game.Move{MockMove{id: 0}}}

		// Launch selection and backup simultaneously
		var wg sync.WaitGroup
		wg.Add(2)

		// Goroutine 1: Select the child
		go func() {
			defer wg.Done()
			gotChild, _, _ := node.SelectOrExpand(state)
			require.Equal(t, child, gotChild,
				"Should select the child node")
		}()

		// Goroutine 2: Backup through the node
		go func() {
			defer wg.Done()
			got := node.Backup(rewarder("player1"))
			require.Equal(t, parent, got,
				"Should return the parent node")
		}()

		wg.Wait()

		// Verify final state reflects selection
		require.Equal(t, LOSS, child.rewards,
			"Child should apply a temporary loss")
		require.Equal(t, 2, child.visits,
			"Child should apply a temporary loss")
		// Verify final state reflects backup
		require.Equal(t, WIN, node.rewards,
			"Node should reverse virtual loss and add a win")
		require.Equal(t, 3, node.visits,
			"Node should reverse virtual loss and add a visit")
	})
}

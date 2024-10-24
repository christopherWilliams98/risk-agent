package searcher

import (
	"risk/game"
	"testing"

	"github.com/stretchr/testify/require"
)

// TODO: test sequential mcts
/* spec:
- selection:
  - happy path: fully expanded node -> unvisited child or max UCB child, child state
  - otherwise: skip
  - edge case: terminal node -> same node
- expansion:
	- happy path: expandable node -> new added child (for a random move?), child state
	- otherwise: skip
	- edge case: terminal node -> same node
- simulation:
	- happy path: state -> terminal state, winner
		- mock State and State.GetMoves() and State.Play()
	- edge case: terminal state -> same state, winner
- backpropagation:
	- happy path: new added child, winner, rewarder -> [new added child, root] visits & rewards updated
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
	return nil // Implement if needed for tests
}

func TestSelectOrExpand(t *testing.T) {
	t.Run("fully expanded node + unvisited child", func(t *testing.T) {
		node := &decision{
			moves:    []game.Move{MockMove{id: 0}, MockMove{id: 1}},
			children: []Node{&decision{visits: 1}, &decision{visits: 0}},
			visits:   1,
		}
		state := MockState{}

		child, childState, isAdded := node.SelectOrExpand(state)

		require.Equal(t, node.children[1], child, "The unvisited child should be selected")
		require.Equal(t, []game.Move{node.moves[1]}, childState.(MockState).played, "State should be updated by a move to the unvisited child")
		require.False(t, isAdded, "No child should be added")
	})

	t.Run("fully expanded node + no unvisited child", func(t *testing.T) {
		node := &decision{
			moves:    []game.Move{MockMove{id: 0}, MockMove{id: 1}},
			children: []Node{&decision{rewards: 0, visits: 1}, &decision{rewards: 1, visits: 1}},
			visits:   1,
		}
		state := MockState{}

		child, childState, isAdded := node.SelectOrExpand(state)

		require.Equal(t, node.children[1], child, "The child with max policy value should be selected")
		require.Equal(t, []game.Move{node.moves[1]}, childState.(MockState).played, "State should be updated by a move to the max policy child")
		require.False(t, isAdded, "No child should be added")
	})

	t.Run("expandable node", func(t *testing.T) {
		node := &decision{
			moves:    []game.Move{MockMove{id: 0}, MockMove{id: 1}},
			children: []Node{&decision{rewards: 1, visits: 1}},
			visits:   1,
		}
		state := MockState{moves: []game.Move{MockMove{id: 0}, MockMove{id: 1}}}

		child, childState, isAdded := node.SelectOrExpand(state)

		require.Equal(t, node.children[1], child,
			"Node should be expanded with a new child")
		require.Equal(t, []game.Move{node.moves[1]},
			childState.(MockState).played, "State should be updated by a move to the new child")
		require.True(t, isAdded, "A child should be added")
	})

	t.Run("terminal node", func(t *testing.T) {
		node := &decision{}
		state := MockState{}

		child, childState, isAdded := node.SelectOrExpand(state)

		require.Equal(t, node, child, "The same node should be returned")
		require.Equal(t, MockState{}, childState, "The same state should be returned")
		require.False(t, isAdded, "No child should be added")
	})
}

package searcher

import (
	"risk/game"
	"testing"

	"github.com/stretchr/testify/require"
)

// TODO: test parallel MCTS with virtual loss
/* spec:
// sequential:
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
// concurrent:
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

func TestSelectOrExpand(t *testing.T) {
	t.Run("fully expanded node", func(t *testing.T) {
		node := &decision{
			moves:    []game.Move{MockMove{id: 0}, MockMove{id: 1}},
			children: []Node{&decision{rewards: 0, visits: 1}, &decision{rewards: 1, visits: 1}},
			rewards:  1,
			visits:   2,
		}
		state := MockState{}

		gotChild, gotState, gotAdded := node.PickChild(state)

		require.Equal(t, node.children[1], gotChild, "Node should select child with max policy value")
		require.IsType(t, &decision{}, gotChild, "Child should be a decision node")
		require.Equal(t, 1+LOSS, gotChild.(*decision).rewards, "Child should apply virtual loss")
		require.Equal(t, 2, gotChild.(*decision).visits, "Child should apply virtual loss")
		require.Equal(t, []game.Move{node.moves[1]}, gotState.(MockState).played, "State should update by the move to the max policy child")
		require.False(t, gotAdded, "Node should add no child")
		require.Equal(t, 1.0, node.rewards, "Node should not apply virtual loss")
		require.Equal(t, 2, node.visits, "Node should not apply virtual loss")
	})

	t.Run("expandable node", func(t *testing.T) {
		node := &decision{
			moves:    []game.Move{MockMove{id: 0}, MockMove{id: 1}},
			children: []Node{&decision{rewards: 1, visits: 1}},
			visits:   1,
		}
		state := MockState{moves: []game.Move{MockMove{id: 0}, MockMove{id: 1}}}

		gotChild, gotState, gotAdded := node.PickChild(state)

		require.IsType(t, &decision{}, gotChild, "Child should be a decision node")
		require.Equal(t, LOSS, gotChild.(*decision).rewards, "Child should apply virtual loss")
		require.Equal(t, 1, gotChild.(*decision).visits, "Child should apply virtual loss")
		require.Equal(t, []game.Move{node.moves[1]},
			gotState.(MockState).played, "State should update by the move to the child")
		require.True(t, gotAdded, "Node should add the child")
		require.Equal(t, 2, len(node.children), "Node should expand with the new child")
	})

	t.Run("terminal node", func(t *testing.T) {
		node := &decision{}
		state := MockState{}

		child, childState, added := node.PickChild(state)

		require.Equal(t, node, child, "The same node should be returned")
		require.Equal(t, MockState{}, childState, "The same state should be returned")
		require.False(t, added, "No child should be added")
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

		got := node.Update(rewarder("player1"))

		require.Nil(t, got, "Should return no parent")
		require.Equal(t, WIN, node.rewards, "Should apply win reward")
		require.Equal(t, 1, node.visits, "Should add a visit")
	})

	t.Run("winning non-root node", func(t *testing.T) {
		parent := &decision{}
		node := &decision{
			parent:  parent,
			player:  "player1",
			rewards: LOSS, // Virtual loss
			visits:  1,
		}

		got := node.Update(rewarder("player1"))

		require.Equal(t, parent, got, "Should return parent node")
		require.Equal(t, WIN, node.rewards, "Should apply win reward")
		require.Equal(t, 1, node.visits, "Should reverse virtual loss")
	})

	t.Run("losing non-root node", func(t *testing.T) {
		parent := &decision{}
		node := &decision{
			parent:  parent,
			player:  "player1",
			rewards: LOSS, // Virtual loss
			visits:  1,
		}

		got := node.Update(rewarder("player2"))

		require.Equal(t, parent, got, "Should return parent node")
		require.Equal(t, LOSS, node.rewards, "Should apply loss reward")
		require.Equal(t, 1, node.visits, "Should reverse virtual loss")
	})
}

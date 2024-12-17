package searcher

import (
	"risk/game"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockStateDeterministic mocks game state for testing MCTS behaviors with
// deterministic moves only
// - Always has exactly 2 moves available
// - move1 leads to player1's turn
// - move2 leads to player2's turn
// - After 3 moves, player1 always wins
type mockStateDeterministic struct {
	player string
	depth  int // Track move depth for terminal condition
}

func (m mockStateDeterministic) Player() string {
	return m.player
}

func (m mockStateDeterministic) LegalMoves() []game.Move {
	if m.depth >= 3 { // Terminal after 3 moves
		return nil
	}
	return []game.Move{
		mockMove{id: 1}, // Always leads to P1's turn
		mockMove{id: 2}, // Always leads to P2's turn
	}
}

func (m mockStateDeterministic) Play(move game.Move) game.State {
	// Terminal after 3 moves
	if m.depth >= 3 {
		panic("game is already over")
	}

	// Non-terminal: next player depends on move
	var nextPlayer string
	if move.(mockMove).id == 1 {
		nextPlayer = "player1"
	} else {
		nextPlayer = "player2"
	}

	return mockStateDeterministic{
		player: nextPlayer,
		depth:  m.depth + 1,
	}
}

func (m mockStateDeterministic) Hash() game.StateHash {
	return 0 // Not needed for deterministic games
}

func (m mockStateDeterministic) Winner() string {
	if m.depth >= 3 {
		return "player1" // P1 always wins after 3 moves
	}
	return ""
}

func TestSimulate(t *testing.T) {
	move1 := mockMove{id: 1}
	move2 := mockMove{id: 2}
	initialState := mockStateDeterministic{player: "player1", depth: 0}

	t.Run("one episode", func(t *testing.T) {
		mcts := NewMCTS(1, WithEpisodes(1))
		policy := mcts.Simulate(initialState, nil)

		// 1st episode expands root with M1 to C1
		expectedRoot := &decision{
			player:  "player1",
			rewards: WIN, // Backpropagate a win for P1
			visits:  1,
			children: map[game.Move]Node{
				move1: &decision{ // Expand with M1 to C1
					player:  "player1",
					rewards: WIN,
					visits:  1,
				},
			},
		}
		require.Equal(t, map[game.Move]int{move1: 1}, policy, "Should explore M1 once")
		requireTreeEqual(t, expectedRoot, mcts.root.(*decision))
	})

	t.Run("two episodes", func(t *testing.T) {
		mcts := NewMCTS(1, WithEpisodes(2))
		policy := mcts.Simulate(initialState, nil)

		// 2nd episode expands root with M2 to C2
		expectedRoot := &decision{
			player:  "player1",
			rewards: WIN * 2, // Backpropagate 2 wins for P1
			visits:  2,
			children: map[game.Move]Node{
				move1: &decision{
					player:  "player1",
					rewards: WIN,
					visits:  1,
				},
				move2: &decision{ // Expand with M2 to C2
					player:  "player2",
					rewards: LOSS,
					visits:  1,
				},
			},
		}
		require.Equal(t, map[game.Move]int{move1: 1, move2: 1}, policy, "Should explore M1 and M2 each once")
		requireTreeEqual(t, expectedRoot, mcts.root.(*decision))
	})

	t.Run("three episodes", func(t *testing.T) {
		mcts := NewMCTS(1, WithEpisodes(3))
		policy := mcts.Simulate(initialState, nil)

		// After 2 episodes, both moves have been tried once
		// 3rd episode selects either child at random since they have equal UCT scores:
		// C1: rewards=WIN, visits=1, parent_visits=2, score = 1 + sqrt(2ln(2))
		// C2: rewards=LOSS, visits=1, parent_visits=2, score = 1 + sqrt(2ln(2))
		expectedRoot1 := &decision{
			player:  "player1",
			rewards: WIN * 3, // Backpropagate 3 wins for P1
			visits:  3,
			children: map[game.Move]Node{
				move1: &decision{ // Select C1
					player:  "player1",
					rewards: WIN * 2,
					visits:  2,
					children: map[game.Move]Node{
						move1: &decision{ // Expand C1 with M1 again
							player:  "player1",
							rewards: WIN,
							visits:  1,
						},
					},
				},
				move2: &decision{
					player:  "player2",
					rewards: LOSS,
					visits:  1,
				},
			},
		}
		expectedRoot2 := &decision{
			player:  "player1",
			rewards: WIN * 3,
			visits:  3,
			children: map[game.Move]Node{
				move1: &decision{
					player:  "player1",
					rewards: WIN,
					visits:  1,
				},
				move2: &decision{ // Select C2
					player:  "player2",
					rewards: LOSS * 2,
					visits:  2,
					children: map[game.Move]Node{
						move1: &decision{ // Expand C2 with M1
							player:  "player1",
							rewards: WIN,
							visits:  1,
						},
					},
				},
			},
		}

		require.Contains(t, []map[game.Move]int{
			{move1: 2, move2: 1}, // If C1 selected
			{move1: 1, move2: 2}, // If C2 selected
		}, policy, "Should explore one move twice and the other once")

		actualRoot := mcts.root.(*decision)
		if policy[move1] == 2 {
			requireTreeEqual(t, expectedRoot1, actualRoot)
		} else {
			requireTreeEqual(t, expectedRoot2, actualRoot)
		}
	})

	t.Run("four episodes", func(t *testing.T) {
		mcts := NewMCTS(1, WithEpisodes(4))
		policy := mcts.Simulate(initialState, nil)

		// 4th episode selects the child not selected by E3 since it has equal exploitation but bigger exploration
		// C1: rewards=WIN*2, visits=2, parent_visits=3, score = 2/2 + sqrt(2ln(3)/2)
		// C2: rewards=LOSS, visits=1, parent_visits=3, score = 1/1 + sqrt(2ln(3))
		// or
		// C1: rewards=WIN, visits=1, parent_visits=3, score = 1/1 + sqrt(2ln(3))
		// C2: rewards=LOSS*2, visits=2, parent_visits=3, score = 2/2 + sqrt(2ln(3)/2)
		require.Equal(t, map[game.Move]int{move1: 2, move2: 2}, policy,
			"Should explore M1 twice and M2 twice")
		expectedRoot := &decision{
			player:  "player1",
			rewards: WIN * 4, // Backpropagate 4 wins for P1
			visits:  4,
			children: map[game.Move]Node{
				move1: &decision{
					player:  "player1",
					rewards: WIN * 2,
					visits:  2,
					children: map[game.Move]Node{
						move1: &decision{
							player:  "player1",
							rewards: WIN,
							visits:  1,
						},
					},
				},
				move2: &decision{
					player:  "player2",
					rewards: LOSS * 2,
					visits:  2,
					children: map[game.Move]Node{
						move1: &decision{
							player:  "player1",
							rewards: WIN,
							visits:  1,
						},
					},
				},
			},
		}
		requireTreeEqual(t, expectedRoot, mcts.root.(*decision))
	})
}

// mockStateTerminal mocks game state for testing MCTS behaviors with
// terminal states
// - Has exactly 1 move available
// - Playing that move leads to P2's turn and terminal state
// - Terminal state has no moves and P1 always wins
type mockStateTerminal struct {
	player   string
	terminal bool
}

func (m mockStateTerminal) Player() string {
	return m.player
}

func (m mockStateTerminal) LegalMoves() []game.Move {
	if m.terminal {
		return nil
	}
	return []game.Move{mockMove{id: 1}}
}

func (m mockStateTerminal) Play(move game.Move) game.State {
	return mockStateTerminal{
		player:   "player2",
		terminal: true,
	}
}

func (m mockStateTerminal) Hash() game.StateHash {
	return 0 // Not needed for these tests
}

func (m mockStateTerminal) Winner() string {
	if m.terminal {
		return "player1" // P1 always wins
	}
	return ""
}

func TestSimulateTerminal(t *testing.T) {
	initialState := mockStateTerminal{
		player:   "player1",
		terminal: false,
	}
	move := mockMove{id: 1}

	t.Run("first episode expands to terminal state", func(t *testing.T) {
		mcts := NewMCTS(1, WithEpisodes(1))
		policy := mcts.Simulate(initialState, nil)

		// Single episode expands to terminal state
		expectedRoot := &decision{
			player:  "player1",
			rewards: WIN,
			visits:  1,
			children: map[game.Move]Node{
				move: &decision{
					player:  "player2", // Turn changes
					rewards: LOSS,      // From P2's perspective
					visits:  1,
				},
			},
		}

		require.Equal(t, map[game.Move]int{move: 1}, policy,
			"Should explore move once")
		requireTreeEqual(t, expectedRoot, mcts.root.(*decision))
	})

	t.Run("second episode selects terminal state", func(t *testing.T) {
		mcts := NewMCTS(1, WithEpisodes(2))
		policy := mcts.Simulate(initialState, nil)

		// First episode expands to terminal state
		// Second episode selects terminal state and does not expand
		expectedRoot := &decision{
			player:  "player1",
			rewards: WIN * 2,
			visits:  2,
			children: map[game.Move]Node{
				move: &decision{
					player:  "player2", // Turn changes
					rewards: LOSS * 2,  // From P2's perspective
					visits:  2,
				},
			},
		}

		require.Equal(t, map[game.Move]int{move: 2}, policy,
			"Should explore same move twice")
		requireTreeEqual(t, expectedRoot, mcts.root.(*decision))
	})
}

func requireTreeEqual(t *testing.T, expected, actual *decision) {
	t.Helper()

	require.Equal(t, expected.player, actual.player, "Player should match")
	require.Equal(t, expected.rewards, actual.rewards, "Rewards should match")
	require.Equal(t, expected.visits, actual.visits, "Visits should match")
	require.Equal(t, len(expected.children), len(actual.children),
		"Should have same number of children")

	for move, expectedChild := range expected.children {
		actualChild, exists := actual.children[move]
		require.True(t, exists, "Move %v should exist", move)
		switch expected := expectedChild.(type) {
		case *decision:
			actual, ok := actualChild.(*decision)
			require.True(t, ok, "Child should be decision node")
			requireTreeEqual(t, expected, actual)
		case *chance:
			actual, ok := actualChild.(*chance)
			require.True(t, ok, "Child should be chance node")
			requireChanceEqual(t, expected, actual)
		}
	}
}

func requireChanceEqual(t *testing.T, expected, actual *chance) {
	t.Helper()

	require.Equal(t, expected.player, actual.player, "Player should match")
	require.Equal(t, expected.rewards, actual.rewards, "Rewards should match")
	require.Equal(t, expected.visits, actual.visits, "Visits should match")
	require.Equal(t, len(expected.children), len(actual.children),
		"Should have same number of outcomes")

	// Compare outcome nodes recursively
	for i, expectedOutcome := range expected.children {
		requireTreeEqual(t, expectedOutcome, actual.children[i])
	}
}

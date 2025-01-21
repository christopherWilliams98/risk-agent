package searcher

import (
	"fmt"
	"risk/game"
	"testing"

	"github.com/rs/zerolog/log"
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
	return 0 // Not used for these tests
}

func (m mockStateDeterministic) Winner() string {
	if m.depth >= 3 {
		return "player1" // P1 always wins after 3 moves
	}
	return ""
}

func (m mockStateDeterministic) Evaluate() float64 {
	return 0 // Not used for these tests
}

func TestSimulate(t *testing.T) {
	move1 := mockMove{id: 1}
	move2 := mockMove{id: 2}

	t.Run("first episode expands root with either child", func(t *testing.T) {
		initialState := mockStateDeterministic{player: "player1"}
		mcts := NewMCTS(1, WithEpisodes(1))
		got, _ := mcts.Simulate(initialState, nil)

		// 1st episode expands root with M1 to C1 or M2 to C2
		expectedRoot1 := &decision{
			player:  "player1",
			rewards: Win, // Backup a win for P1
			visits:  1,
			children: map[game.Move]Node{ // Expand with M1 to C1
				move1: &decision{player: "player1", rewards: Win, visits: 1},
			},
		}
		expectedRoot2 := &decision{
			player:  "player1",
			rewards: Win, // Backup a win for P1
			visits:  1,
			children: map[game.Move]Node{ // Expand with M2 to C2
				move2: &decision{player: "player2", rewards: Loss, visits: 1},
			},
		}
		require.Contains(t, []map[game.Move]float64{
			{move1: 1}, // If C1 selected
			{move2: 1}, // If C2 selected
		}, got, "Should explore M1 or M2 once")
		require.True(t, containsTree([]*decision{expectedRoot1, expectedRoot2}, mcts.root), "Tree should be constructed correctly")
	})

	t.Run("second episode expands root with the other child", func(t *testing.T) {
		initialState := mockStateDeterministic{player: "player1"}
		mcts := NewMCTS(1, WithEpisodes(2))
		got, _ := mcts.Simulate(initialState, nil)

		// 2nd episode expands root with M2 to C2
		expectedRoot := &decision{
			player:  "player1",
			rewards: Win * 2, // Backup 2 wins for P1
			visits:  2,
			children: map[game.Move]Node{ // Expand with M2 to C2
				move1: &decision{player: "player1", rewards: Win, visits: 1},
				move2: &decision{player: "player2", rewards: Loss, visits: 1},
			},
		}
		require.Equal(t, map[game.Move]float64{move1: 1, move2: 1}, got, "Should explore M1 and M2 each once")
		require.True(t, containsTree([]*decision{expectedRoot}, mcts.root), "Tree should be constructed correctly")
	})

	t.Run("third episode selects either child", func(t *testing.T) {
		initialState := mockStateDeterministic{player: "player1"}
		mcts := NewMCTS(1, WithEpisodes(3))
		got, _ := mcts.Simulate(initialState, nil)

		// After 2 episodes, both moves have been tried once
		// 3rd episode selects either child since they have equal UCT scores:
		// C1: rewards=WIN, visits=1, parent_visits=2, score = 1 + sqrt(2ln(2))
		// C2: rewards=LOSS, visits=1, parent_visits=2, score = 1 + sqrt(2ln(2))
		// and expands it with a random move
		expectedRoot11 := &decision{
			player:  "player1",
			rewards: Win * 3, // Backup 3 wins for P1
			visits:  3,
			children: map[game.Move]Node{
				move1: &decision{ // Select C1
					player:  "player1",
					rewards: Win * 2,
					visits:  2,
					children: map[game.Move]Node{ // Expand C1 with M1
						move1: &decision{player: "player1", rewards: Win, visits: 1},
					},
				},
				move2: &decision{player: "player2", rewards: Loss, visits: 1},
			},
		}
		expectedRoot12 := &decision{
			player:  "player1",
			rewards: Win * 3, // Backup 3 wins for P1
			visits:  3,
			children: map[game.Move]Node{
				move1: &decision{ // Select C1
					player:  "player1",
					rewards: Win * 2,
					visits:  2,
					children: map[game.Move]Node{ // Expand C1 with M2
						move2: &decision{player: "player2", rewards: Loss, visits: 1},
					},
				},
				move2: &decision{player: "player2", rewards: Loss, visits: 1},
			},
		}
		expectedRoot21 := &decision{
			player:  "player1",
			rewards: Win * 3,
			visits:  3,
			children: map[game.Move]Node{
				move1: &decision{player: "player1", rewards: Win, visits: 1},
				move2: &decision{ // Select C2
					player:  "player2",
					rewards: Loss * 2,
					visits:  2,
					children: map[game.Move]Node{ // Expand C2 with M1
						move1: &decision{player: "player1", rewards: Win, visits: 1},
					},
				},
			},
		}
		expectedRoot22 := &decision{
			player:  "player1",
			rewards: Win * 3,
			visits:  3,
			children: map[game.Move]Node{
				move1: &decision{player: "player1", rewards: Win, visits: 1},
				move2: &decision{ // Select C2
					player:  "player2",
					rewards: Loss * 2,
					visits:  2,
					children: map[game.Move]Node{ // Expand C2 with M2
						move2: &decision{player: "player2", rewards: Loss, visits: 1},
					},
				},
			},
		}

		require.Contains(t, []map[game.Move]float64{
			{move1: 2, move2: 1}, // If C1 selected
			{move1: 1, move2: 2}, // If C2 selected
		}, got, "Should explore one move twice and the other once")
		require.True(t, containsTree([]*decision{expectedRoot11, expectedRoot12, expectedRoot21, expectedRoot22}, mcts.root), "Tree should be constructed correctly")
	})

	t.Run("fourth episode balances exploration vs exploitation", func(t *testing.T) {
		initialState := mockStateDeterministic{player: "player1"}
		mcts := NewMCTS(1, WithEpisodes(4))
		got, _ := mcts.Simulate(initialState, nil)

		// 4th episode selects the child not selected by E3 since it has equal exploitation but bigger exploration
		// C1: rewards=WIN*2, visits=2, parent_visits=3, score = 2/2 + sqrt(2ln(3)/2)
		// C2: rewards=LOSS, visits=1, parent_visits=3, score = 1/1 + sqrt(2ln(3))
		// or
		// C1: rewards=WIN, visits=1, parent_visits=3, score = 1/1 + sqrt(2ln(3))
		// C2: rewards=LOSS*2, visits=2, parent_visits=3, score = 2/2 + sqrt(2ln(3)/2)
		expectedRoot11 := &decision{
			player:  "player1",
			rewards: Win * 4, // Backup 4 wins for P1
			visits:  4,
			children: map[game.Move]Node{
				move1: &decision{
					player:  "player1",
					rewards: Win * 2,
					visits:  2,
					children: map[game.Move]Node{
						move1: &decision{player: "player1", rewards: Win, visits: 1},
					},
				},
				move2: &decision{
					player:  "player2",
					rewards: Loss * 2,
					visits:  2,
					children: map[game.Move]Node{
						move1: &decision{player: "player1", rewards: Win, visits: 1},
					},
				},
			},
		}
		expectedRoot12 := &decision{
			player:  "player1",
			rewards: Win * 4, // Backup 4 wins for P1
			visits:  4,
			children: map[game.Move]Node{
				move1: &decision{
					player:  "player1",
					rewards: Win * 2,
					visits:  2,
					children: map[game.Move]Node{
						move1: &decision{player: "player1", rewards: Win, visits: 1},
					},
				},
				move2: &decision{
					player:  "player2",
					rewards: Loss * 2,
					visits:  2,
					children: map[game.Move]Node{
						move2: &decision{player: "player2", rewards: Loss, visits: 1},
					},
				},
			},
		}
		expectedRoot21 := &decision{
			player:  "player1",
			rewards: Win * 4, // Backup 4 wins for P1
			visits:  4,
			children: map[game.Move]Node{
				move1: &decision{
					player:  "player1",
					rewards: Win * 2,
					visits:  2,
					children: map[game.Move]Node{
						move2: &decision{player: "player2", rewards: Loss, visits: 1},
					},
				},
				move2: &decision{
					player:  "player2",
					rewards: Loss * 2,
					visits:  2,
					children: map[game.Move]Node{
						move1: &decision{player: "player1", rewards: Win, visits: 1},
					},
				},
			},
		}
		expectedRoot22 := &decision{
			player:  "player1",
			rewards: Win * 4, // Backup 4 wins for P1
			visits:  4,
			children: map[game.Move]Node{
				move1: &decision{
					player:  "player1",
					rewards: Win * 2,
					visits:  2,
					children: map[game.Move]Node{
						move2: &decision{player: "player2", rewards: Loss, visits: 1},
					},
				},
				move2: &decision{
					player:  "player2",
					rewards: Loss * 2,
					visits:  2,
					children: map[game.Move]Node{
						move2: &decision{player: "player2", rewards: Loss, visits: 1},
					},
				},
			},
		}
		require.Equal(t, map[game.Move]float64{move1: 2, move2: 2}, got,
			"Should explore M1 twice and M2 twice")
		require.True(t, containsTree([]*decision{expectedRoot11, expectedRoot12, expectedRoot21, expectedRoot22}, mcts.root), "Tree should be constructed correctly")
	})
}

func TestSimulateParallel(t *testing.T) {
	move1 := mockMove{id: 1}
	move2 := mockMove{id: 2}
	initialState := mockStateDeterministic{player: "player1"}

	mcts := NewMCTS(2, WithEpisodes(4)) // 2 goroutines, 4 episodes
	got, _ := mcts.Simulate(initialState, nil)

	// After 4 episodes with 2 goroutines:
	// - Root should have expanded both moves
	// - Each child should be selected and expanded once
	// - Visit counts and rewards should match sequential simulation's results
	expectedRoot11 := &decision{
		player:  "player1",
		rewards: Win * 4, // Backup 4 wins for P1
		visits:  4,
		children: map[game.Move]Node{
			move1: &decision{
				player:  "player1",
				rewards: Win * 2,
				visits:  2,
				children: map[game.Move]Node{
					move1: &decision{player: "player1", rewards: Win, visits: 1},
				},
			},
			move2: &decision{
				player:  "player2",
				rewards: Loss * 2,
				visits:  2,
				children: map[game.Move]Node{
					move1: &decision{player: "player1", rewards: Win, visits: 1},
				},
			},
		},
	}
	expectedRoot12 := &decision{
		player:  "player1",
		rewards: Win * 4, // Backup 4 wins for P1
		visits:  4,
		children: map[game.Move]Node{
			move1: &decision{
				player:  "player1",
				rewards: Win * 2,
				visits:  2,
				children: map[game.Move]Node{
					move1: &decision{player: "player1", rewards: Win, visits: 1},
				},
			},
			move2: &decision{
				player:  "player2",
				rewards: Loss * 2,
				visits:  2,
				children: map[game.Move]Node{
					move2: &decision{player: "player2", rewards: Loss, visits: 1},
				},
			},
		},
	}
	expectedRoot21 := &decision{
		player:  "player1",
		rewards: Win * 4, // Backup 4 wins for P1
		visits:  4,
		children: map[game.Move]Node{
			move1: &decision{
				player:  "player1",
				rewards: Win * 2,
				visits:  2,
				children: map[game.Move]Node{
					move2: &decision{player: "player2", rewards: Loss, visits: 1},
				},
			},
			move2: &decision{
				player:  "player2",
				rewards: Loss * 2,
				visits:  2,
				children: map[game.Move]Node{
					move1: &decision{player: "player1", rewards: Win, visits: 1},
				},
			},
		},
	}
	expectedRoot22 := &decision{
		player:  "player1",
		rewards: Win * 4, // Backup 4 wins for P1
		visits:  4,
		children: map[game.Move]Node{
			move1: &decision{
				player:  "player1",
				rewards: Win * 2,
				visits:  2,
				children: map[game.Move]Node{
					move2: &decision{player: "player2", rewards: Loss, visits: 1},
				},
			},
			move2: &decision{
				player:  "player2",
				rewards: Loss * 2,
				visits:  2,
				children: map[game.Move]Node{
					move2: &decision{player: "player2", rewards: Loss, visits: 1},
				},
			},
		},
	}
	require.Equal(t, map[game.Move]float64{move1: 2, move2: 2}, got,
		"Should explore M1 twice and M2 twice")
	require.True(t, containsTree([]*decision{expectedRoot11, expectedRoot12, expectedRoot21, expectedRoot22}, mcts.root), "Tree should be constructed correctly")
}

// mockStateTerminal mocks game state for testing MCTS behaviors with
// terminal states
// - Has exactly 1 move available that leads to terminal state
// - Terminal state has no moves
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
	if m.terminal {
		panic("game over")
	}

	return mockStateTerminal{
		player:   m.player,
		terminal: true,
	}
}

func (m mockStateTerminal) Hash() game.StateHash {
	return 0 // Not used for these tests
}

func (m mockStateTerminal) Winner() string {
	if m.terminal {
		return m.player
	}
	return ""
}

func (m mockStateTerminal) Evaluate() float64 {
	return 0 // Not used for these tests
}

func TestSimulateTerminal(t *testing.T) {
	move := mockMove{id: 1}

	t.Run("first episode expands to terminal state", func(t *testing.T) {
		initialState := mockStateTerminal{player: "player1"}

		mcts := NewMCTS(1, WithEpisodes(1))
		got, _ := mcts.Simulate(initialState, nil)

		// First episode expands to terminal state
		expectedRoot := &decision{
			player:  "player1",
			rewards: Win,
			visits:  1,
			children: map[game.Move]Node{
				move: &decision{
					player:  "player1",
					rewards: Win,
					visits:  1,
				},
			},
		}

		require.Equal(t, map[game.Move]float64{move: 1}, got,
			"Should explore move once")
		require.True(t, containsTree([]*decision{expectedRoot}, mcts.root), "Tree should be constructed correctly")
	})

	t.Run("second episode selects terminal state", func(t *testing.T) {
		initialState := mockStateTerminal{player: "player1"}

		mcts := NewMCTS(1, WithEpisodes(2))
		got, _ := mcts.Simulate(initialState, nil)

		// Second episode selects terminal state and does not expand
		expectedRoot := &decision{
			player:  "player1",
			rewards: Win * 2,
			visits:  2,
			children: map[game.Move]Node{
				move: &decision{
					player:  "player1",
					rewards: Win * 2,
					visits:  2,
				},
			},
		}

		require.Equal(t, map[game.Move]float64{move: 2}, got,
			"Should explore same move twice")
		require.True(t, containsTree([]*decision{expectedRoot}, mcts.root), "Tree should be constructed correctly")
	})

	t.Run("third episode selects terminal state again", func(t *testing.T) {
		initialState := mockStateTerminal{player: "player1"}

		mcts := NewMCTS(1, WithEpisodes(3))
		got, _ := mcts.Simulate(initialState, nil)

		// Third episode selects terminal state again and does not expand
		expectedRoot := &decision{
			player:  "player1",
			rewards: Win * 3,
			visits:  3,
			children: map[game.Move]Node{
				move: &decision{
					player:  "player1",
					rewards: Win * 3,
					visits:  3,
				},
			},
		}

		require.Equal(t, map[game.Move]float64{move: 3}, got, "Should explore same move three times")
		require.True(t, containsTree([]*decision{expectedRoot}, mcts.root), "Tree should be constructed correctly")
	})
}

func TestSimulateTerminalParallel(t *testing.T) {
	move := mockMove{id: 1}
	initialState := mockStateTerminal{player: "player1"}

	mcts := NewMCTS(2, WithEpisodes(3))
	got, _ := mcts.Simulate(initialState, nil)

	// After 3 episodes with 2 goroutines:
	// - Root should have expanded the move once
	// - The child is terminal and selected twice
	// - Visit counts should add up correctly
	// - Rewards should reflect wins/losses correctly
	expectedRoot := &decision{
		player:  "player1",
		rewards: Win * 3,
		visits:  3,
		children: map[game.Move]Node{
			move: &decision{
				player:  "player1",
				rewards: Win * 3,
				visits:  3,
			},
		},
	}

	require.Equal(t, map[game.Move]float64{move: 3}, got, "Should explore same move three times")
	require.True(t, containsTree([]*decision{expectedRoot}, mcts.root), "Tree should be constructed correctly")
}

// alternator returns a function that alternates between two outcomes
func alternator() func(mockStateStochastic, mockStateStochastic) mockStateStochastic {
	first := false
	return func(one, other mockStateStochastic) mockStateStochastic {
		first = !first
		if first {
			return one
		}
		return other
	}
}

// mockStateStochastic mocks game state for testing MCTS behaviors with
// mixed deterministic and stochastic moves
// - S0 has 2 moves available: M1 (stochastic) and M2 (deterministic)
// - Other states have only 1 move: M3 (deterministic)
// - M1 leads to 2 possible outcomes: S1 (P1's turn) or S3 (P2's turn)
// - M2 leads to S2 (P2's turn)
// - Terminal after 2 moves and whoever last played wins
type mockStateStochastic struct {
	player      string
	state       string // Track different states: S0, S1, S2, S3
	depth       int
	nextOutcome func(mockStateStochastic,
		mockStateStochastic) mockStateStochastic // Fetch stochastic outcome
}

func (m mockStateStochastic) Player() string {
	return m.player
}

func (m mockStateStochastic) LegalMoves() []game.Move {
	if m.depth >= 2 { // Terminal after 2 moves
		return nil
	}
	if m.state == "S0" { // Initial state
		return []game.Move{
			mockMove{id: 1, stochastic: true}, // M1: stochastic
			mockMove{id: 2},                   // M2: deterministic
		}
	}
	// S1, S2, S3 have the same move available
	return []game.Move{mockMove{id: 3}}
}

func (m mockStateStochastic) Play(move game.Move) game.State {
	if m.depth >= 2 {
		panic("game is already over")
	}

	if m.state == "S0" {
		if move.IsStochastic() { // stochastic M1: alternate between S1 and S3
			return m.nextOutcome(mockStateStochastic{ // S1: P1's turn
				player: "player1",
				state:  "S1",
				depth:  1,
			}, mockStateStochastic{ // S3: P2's turn
				player: "player2",
				state:  "S3",
				depth:  1,
			})
		}
		return mockStateStochastic{ // deterministic M2
			player: "player2", // S2: P2's turn
			state:  "S2",
			depth:  m.depth + 1,
		}
	}

	// Terminal state after 2 moves
	return mockStateStochastic{
		player: m.player,
		state:  m.state,
		depth:  m.depth + 1,
	}
}

func (m mockStateStochastic) Hash() game.StateHash {
	return game.StateHash(m.state[len(m.state)-1]) // Distinguish states
}

func (m mockStateStochastic) Winner() string {
	if m.depth >= 2 {
		return m.player // Whoever played last wins
	}
	return ""
}

func (m mockStateStochastic) Evaluate() float64 {
	return 0 // Not used for these tests
}

func TestSimulateStochastic(t *testing.T) {
	move1 := mockMove{id: 1, stochastic: true}
	move2 := mockMove{id: 2}
	move3 := mockMove{id: 3}

	t.Run("first episode expands either move", func(t *testing.T) {
		initialState := mockStateStochastic{
			player:      "player1",
			state:       "S0",
			nextOutcome: alternator(),
		}
		mcts := NewMCTS(1, WithEpisodes(1))
		got, _ := mcts.Simulate(initialState, nil)

		// One episode expands either move
		expectedRoot1 := &decision{
			player:  "player1",
			rewards: Win,
			visits:  1,
			children: map[game.Move]Node{ // Expand M1 to chance node
				move1: &chance{player: "player1", rewards: Win, visits: 1, children: []*decision{}}, // No outcomes yet
			},
		}
		expectedRoot2 := &decision{
			player:  "player1",
			rewards: Loss,
			visits:  1,
			children: map[game.Move]Node{ // Expand M2 to S2
				move2: &decision{player: "player2", rewards: Win, visits: 1}, // No outcomes yet
			},
		}

		require.Contains(t, []map[game.Move]float64{{move1: 1}, {move2: 1}}, got,
			"Should explore stochastic move once")
		require.True(t, containsTree([]*decision{expectedRoot1, expectedRoot2}, mcts.root), "Tree should be constructed correctly")
	})

	t.Run("second episode expands the other move", func(t *testing.T) {
		initialState := mockStateStochastic{
			player:      "player1",
			state:       "S0",
			nextOutcome: alternator(),
		}
		mcts := NewMCTS(1, WithEpisodes(2))
		got, _ := mcts.Simulate(initialState, nil)

		// Two episodes should expand both moves
		expectedRoot := &decision{
			player:  "player1",
			rewards: Win + Loss,
			visits:  2,
			children: map[game.Move]Node{
				move1: &chance{
					player:   "player1",
					rewards:  Win,
					visits:   1,
					children: []*decision{}, // No outcomes yet
				},
				move2: &decision{ // S2
					player:  "player2",
					rewards: Win,
					visits:  1,
				},
			},
		}

		require.Equal(t, map[game.Move]float64{
			move1: 1,
			move2: 1,
		}, got, "Should explore both moves once")
		require.True(t, containsTree([]*decision{expectedRoot}, mcts.root), "Tree should be constructed correctly")
	})

	t.Run("third episode expands one stochastic outcome", func(t *testing.T) {
		initialState := mockStateStochastic{
			player:      "player1",
			state:       "S0",
			nextOutcome: alternator(),
		}
		mcts := NewMCTS(1, WithEpisodes(3))
		got, _ := mcts.Simulate(initialState, nil)

		// First episode used outcome S1 for rollout
		// Third episode expands chance node with outcome S3
		expectedRoot := &decision{
			player:  "player1",
			rewards: Win + Loss*2,
			visits:  3,
			children: map[game.Move]Node{
				move1: &chance{
					player:  "player1",
					rewards: Win + Loss,
					visits:  2,
					children: []*decision{
						{ // Outcome S3
							player:  "player2",
							rewards: Win,
							visits:  1,
						},
					},
				},
				move2: &decision{ // S2
					player:  "player2",
					rewards: Win,
					visits:  1,
				},
			},
		}

		require.Equal(t, map[game.Move]float64{
			move1: 2,
			move2: 1,
		}, got, "Should explore stochastic move twice")
		require.True(t, containsTree([]*decision{expectedRoot}, mcts.root), "Tree should be constructed correctly")
	})

	t.Run("fourth episode expands the other stochastic outcome", func(t *testing.T) {
		initialState := mockStateStochastic{
			player:      "player1",
			state:       "S0",
			nextOutcome: alternator(),
		}
		mcts := NewMCTS(1, WithEpisodes(4))
		got, _ := mcts.Simulate(initialState, nil)

		// Fourth episode expands chance node with outcome S1
		expectedRoot := &decision{
			player:  "player1",
			rewards: Win*2 + Loss*2,
			visits:  4,
			children: map[game.Move]Node{
				move1: &chance{
					player:  "player1",
					rewards: Win*2 + Loss,
					visits:  3,
					children: []*decision{
						{ // Outcome S3
							player:  "player2",
							rewards: Win,
							visits:  1,
						},
						{ // Outcome S1
							player:  "player1",
							rewards: Win,
							visits:  1,
						},
					},
				},
				move2: &decision{ // S2
					player:  "player2",
					rewards: Win,
					visits:  1,
				},
			},
		}

		require.Equal(t, map[game.Move]float64{
			move1: 3,
			move2: 1,
		}, got, "Should explore stochastic move three times")
		require.True(t, containsTree([]*decision{expectedRoot}, mcts.root), "Tree should be constructed correctly")
	})

	t.Run("fifth episode selects existing outcome", func(t *testing.T) {
		initialState := mockStateStochastic{
			player:      "player1",
			state:       "S0",
			nextOutcome: alternator(),
		}
		mcts := NewMCTS(1, WithEpisodes(5))
		got, _ := mcts.Simulate(initialState, nil)

		// Fifth episode selects outcome S1 and expands it with M3
		expectedRoot := &decision{
			player:  "player1",
			rewards: Win*2 + Loss*3,
			visits:  5,
			children: map[game.Move]Node{
				move1: &chance{
					player:  "player1",
					rewards: Win*2 + Loss*2,
					visits:  4,
					children: []*decision{
						{ // Outcome S3
							player:  "player2",
							rewards: Win * 2,
							visits:  2,
							children: map[game.Move]Node{
								move3: &decision{ // Expanded by M3
									player:  "player2",
									rewards: Win,
									visits:  1,
								},
							},
						},
						{ // Outcome S1
							player:  "player1",
							rewards: Win,
							visits:  1,
						},
					},
				},
				move2: &decision{ // S2
					player:  "player2",
					rewards: Win,
					visits:  1,
				},
			},
		}

		require.Equal(t, map[game.Move]float64{
			move1: 4,
			move2: 1,
		}, got, "Should explore stochastic move four times")
		require.True(t, containsTree([]*decision{expectedRoot}, mcts.root), "Tree should be constructed correctly")
	})
}

func TestSimulateStochasticParallel(t *testing.T) {
	move1 := mockMove{id: 1, stochastic: true}
	move2 := mockMove{id: 2}
	move3 := mockMove{id: 3}
	initialState := mockStateStochastic{
		player:      "player1",
		state:       "S0",
		nextOutcome: alternator(),
	}

	mcts := NewMCTS(2, WithEpisodes(5)) // 2 goroutines, 5 episodes
	got, _ := mcts.Simulate(initialState, nil)

	// After 5 episodes with 2 goroutines:
	// - Root should have expanded both moves
	// - Chance node should have both outcomes expanded
	// - One outcome should be expanded deeper
	// - Visit counts should add up correctly
	// - Rewards should reflect wins/losses correctly
	expectedRoot := &decision{
		player:  "player1",
		rewards: Win*2 + Loss*3,
		visits:  5,
		children: map[game.Move]Node{
			move1: &chance{
				player:  "player1",
				rewards: Win*2 + Loss*2,
				visits:  4,
				children: []*decision{
					{ // Outcome S3
						player:  "player2",
						rewards: Win * 2,
						visits:  2,
						children: map[game.Move]Node{
							move3: &decision{
								player:  "player2",
								rewards: Win,
								visits:  1,
							},
						},
					},
					{ // Outcome S1
						player:  "player1",
						rewards: Win,
						visits:  1,
					},
				},
			},
			move2: &decision{ // S2
				player:  "player2",
				rewards: Win,
				visits:  1,
			},
		},
	}

	require.Equal(t, map[game.Move]float64{
		move1: 4,
		move2: 1,
	}, got, "Should explore stochastic move four times")
	require.True(t, containsTree([]*decision{expectedRoot}, mcts.root), "Tree should be constructed correctly")
}

func containsTree(expected []*decision, actual *decision) bool {
	for _, candidate := range expected {
		if decisionEqual(candidate, actual) {
			return true
		}
	}
	log.Debug().Str("actual", fmt.Sprintf("%+v", actual)).Msg("no match was found")
	return false
}

func decisionEqual(expected, actual *decision) bool {
	if expected.player != actual.player ||
		expected.rewards != actual.rewards ||
		expected.visits != actual.visits ||
		len(expected.children) != len(actual.children) {
		return false
	}

	for move, expectedChild := range expected.children {
		actualChild, exists := actual.children[move]
		if !exists {
			return false
		}

		switch expected := expectedChild.(type) {
		case *decision:
			actual, ok := actualChild.(*decision)
			if !ok || !decisionEqual(expected, actual) {
				return false
			}
		case *chance:
			actual, ok := actualChild.(*chance)
			if !ok || !chanceEqual(expected, actual) {
				return false
			}
		}
	}
	return true
}

func chanceEqual(expected, actual *chance) bool {
	if expected.player != actual.player ||
		expected.rewards != actual.rewards ||
		expected.visits != actual.visits ||
		len(expected.children) != len(actual.children) {
		return false
	}

	for i, expectedOutcome := range expected.children {
		if !decisionEqual(expectedOutcome, actual.children[i]) {
			return false
		}
	}
	return true
}

package engine

import (
	"fmt"
	"risk/game"
	"risk/searcher"
	"risk/searcher/agent"
)

type Engine struct {
	State  *game.GameState
	Agents []MCTSAdapter
}

type Update struct {
	Move  game.Move
	State game.State
	Hash  game.StateHash
}

func LocalEngine(players []string, agents []MCTSAdapter, m *game.Map, r game.Rules) *Engine {
	if len(players) != len(agents) {
		panic("number of players does not match number of agents")
	}
	if len(players) < 2 {
		panic("need at least two players")
	}

	state := game.NewGameState(m, r)

	state.CurrentPlayer = 1

	eng := &Engine{
		State:  state,
		Agents: agents,
	}
	return eng
}

// Run executes the entire game loop until a winner is found.
func (e *Engine) Run() {

	updates := make([][]Update, len(e.Agents))
	for i := range updates {
		updates[i] = []Update{}
	}

	turnCount := 0
	maxTurns := 500

	// Loop until there's a winner
	for e.State.Winner() == "" && turnCount < maxTurns {
		currentPlayerID := e.State.CurrentPlayer
		agentIndex := currentPlayerID - 1

		// Debug print
		fmt.Printf("===== TURN BEGIN: Player %d (agent index %d) =====\n", currentPlayerID, agentIndex)

		move := e.Agents[agentIndex].FindMove(e.State, updates[agentIndex])

		// Debug print
		fmt.Printf("[Engine.Run] Player %d chose move: %+v\n", currentPlayerID, move)

		newState := e.State.Play(move).(*game.GameState)

		u := Update{
			Move:  move,
			State: newState.Copy(),
			Hash:  newState.Hash(),
		}
		updates[agentIndex] = append(updates[agentIndex], u)

		e.State = newState
		fmt.Printf("turn %d\n", turnCount)
		turnCount++

	}

	if e.State.Winner() != "" {
		fmt.Printf("Game ended due to a winner: %s\n", e.State.Winner())
	} else {
		fmt.Printf("Stopped after %d turns (no winner yet)\n", maxTurns)
	}
}

type MCTSAdapter struct {
	InternalAgent agent.Agent
}

func (ma *MCTSAdapter) FindMove(gs *game.GameState, recentUpdates []Update) game.Move {
	segments := make([]searcher.Segment, len(recentUpdates))
	for i, upd := range recentUpdates {
		segments[i] = searcher.Segment{
			Move:  upd.Move,
			State: upd.State,
		}
	}
	candidate := ma.InternalAgent.FindMove(gs, segments)

	if !game.IsMoveValidForPhase(gs.Phase, candidate) {
		fmt.Printf("[MCTSAdapter] MCTS returned an invalid move for Phase=%d => forcing pass.\n",
			gs.Phase)
		fallbackMoves := gs.LegalMoves()
		if len(fallbackMoves) == 0 {
			panic("No legal moves at all!")
		}
		return fallbackMoves[0]
	}

	return candidate
}

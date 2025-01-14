package engine

import (
	// "fmt"
	"risk/game"
	"risk/searcher"
	"risk/searcher/agent"
	metric "risk/searcher/experiments"

	"github.com/rs/zerolog/log"
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

// TODO: remove players arg
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
func (e *Engine) Run() (string, metric.GameMetrics) {
	updates := make([][]Update, len(e.Agents))
	for i := range updates {
		updates[i] = []Update{}
	}

	log.Info().Msgf("player %d is starting", e.State.CurrentPlayer)

	// Loop until there's a winner
	turnCount := 1
	const MaxTurns = 500
	var gameMetrics []metric.MoveMetrics
	for e.State.Winner() == "" && turnCount <= MaxTurns {
		currentPlayerID := e.State.CurrentPlayer
		agentIndex := currentPlayerID - 1

		// Debug print
		// fmt.Printf("===== TURN BEGIN: Player %d (agent index %d) =====\n", currentPlayerID, agentIndex)

		move, metrics := e.Agents[agentIndex].FindMove(e.State, updates[agentIndex])
		gameMetrics = append(gameMetrics, metric.MoveMetrics{
			Step:          turnCount,
			Player:        e.State.CurrentPlayer,
			SearchMetrics: metrics,
		})

		// Debug print
		// fmt.Printf("[Engine.Run] Player %d chose move: %+v\n", currentPlayerID, move)

		newState := e.State.Play(move).(*game.GameState)

		u := Update{
			Move:  move,
			State: newState.Copy(),
			Hash:  newState.Hash(),
		}
		updates[agentIndex] = append(updates[agentIndex], u)

		e.State = newState
		// fmt.Printf("turn %d\n", turnCount)
		turnCount++

	}

	if e.State.Winner() != "" {
		// fmt.Printf("Game ended due to a winner: %s\n", e.State.Winner())
	} else {
		// fmt.Printf("Stopped after %d turns (no winner yet)\n", MaxTurns)
	}

	return e.State.Winner(), gameMetrics
}

// TODO: remove code smell
type MCTSAdapter struct {
	InternalAgent agent.Agent
}

func (ma *MCTSAdapter) FindMove(gs *game.GameState, recentUpdates []Update) (game.Move, metric.SearchMetrics) {
	segments := make([]searcher.Segment, len(recentUpdates))
	for i, upd := range recentUpdates {
		segments[i] = searcher.Segment{
			Move:  upd.Move,
			State: upd.State,
		}
	}
	candidate, metrics := ma.InternalAgent.FindMove(gs, segments)

	if !game.IsMoveValidForPhase(gs.Phase, candidate) {
		// fmt.Printf("[MCTSAdapter] MCTS returned an invalid move for Phase=%d => forcing pass.\n", gs.Phase)
		fallbackMoves := gs.LegalMoves()
		if len(fallbackMoves) == 0 {
			panic("No legal moves at all!")
		}
		return fallbackMoves[0], metrics
	}

	return candidate, metrics
}

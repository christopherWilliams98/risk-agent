package engine

import (
	"risk/experiments/metrics"
	"risk/game"
	"risk/searcher"
	"risk/searcher/agent"
	"time"

	"github.com/rs/zerolog/log"
)

// Evaluation engine runs a game locally and collects performance metrics
type localEngine struct {
	agents []agent.Agent
}

func NewLocalEngine(agents []agent.Agent) Engine {
	if len(agents) != 2 {
		panic("need two agents to play a game")
	}
	return &localEngine{agents: agents}
}

func (e *localEngine) Run() (string, metrics.GameMetric, []metrics.MoveMetric) {
	// Initialize a new game
	m := game.CreateMap()
	rules := game.NewStandardRules()
	state := game.NewGameState(m, rules)

	startingPlayer := state.CurrentPlayer
	log.Info().Msgf("player %d is starting", startingPlayer)

	// Ask agents to play moves till there's a winner or max moves is exceeded
	var moveMetrics []metrics.MoveMetric
	start := time.Now()
	var turnUpdates []searcher.Segment
	prevPlayer := startingPlayer

	for numMoves := 1; state.Winner() == "" && numMoves <= MaxMoves; numMoves++ {
		// Find the next move
		currPlayer := state.CurrentPlayer
		relevantUpdates := turnUpdates
		if relevantUpdates != nil && currPlayer == prevPlayer {
			relevantUpdates = turnUpdates[len(turnUpdates)-1:]
		}
		move, searchMetric := e.agents[currPlayer-1].FindMove(state, relevantUpdates...)
		// Collect move metrics
		moveMetrics = append(moveMetrics, metrics.MoveMetric{
			Step:         numMoves,
			Player:       currPlayer,
			SearchMetric: searchMetric,
		})
		// Play the next move
		nextState, ok := state.Play(move).(*game.GameState)
		if !ok {
			panic("unexpected state type")
		}
		// Collect moves played during this player's turn
		if currPlayer != prevPlayer { // Reset when turn changes
			turnUpdates = []searcher.Segment{}
		}
		turnUpdates = append(turnUpdates, searcher.Segment{
			Move:      move,
			StateHash: nextState.Hash(),
		})

		state = nextState
		prevPlayer = currPlayer
	}

	winner := state.Winner()
	if winner == "" {
		log.Warn().Msgf("game ended after max number of moves (%d) without a winner", MaxMoves)
	}

	end := time.Now()
	gameMetric := metrics.GameMetric{
		StartingPlayer: startingPlayer,
		Winner:         winner,
		StartTime:      start,
		EndTime:        end,
		Duration:       end.Sub(start),
	}

	return winner, gameMetric, moveMetrics
}

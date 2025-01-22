package agent

import (
	"risk/experiments/metrics"
	"risk/game"
	"risk/searcher"

	"github.com/rs/zerolog/log"
)

type evaluationAgent struct {
	mcts *searcher.MCTS
}

// NewEvaluationAgent returns a new agent for actual game play during evaluation.
func NewEvaluationAgent(mcts *searcher.MCTS) Agent {
	return evaluationAgent{mcts: mcts}
}

func (a evaluationAgent) FindMove(state game.State, updates ...searcher.Segment) (game.Move, metrics.SearchMetric) {
	policy, metric := a.mcts.Simulate(state, updates)
	move := findMax(policy)
	return move, metric
}

func findMax(policy map[game.Move]float64) game.Move {
	var maxMove game.Move
	maxVisit := -1.0
	var moves []game.Move
	var visits []float64
	for move, visit := range policy {
		if visit > maxVisit {
			maxVisit = visit
			maxMove = move
		}
		moves = append(moves, move)
		visits = append(visits, visit)
	}

	if maxMove == nil {
		log.Error().Msgf("maxMove is nil, policy %+v, moves %+v, visits %+v", policy, moves, visits)
	}

	return maxMove
}

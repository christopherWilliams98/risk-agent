package agent

import (
	"risk/game"
	"risk/searcher"
)

type evaluationAgent struct {
	mcts *searcher.MCTS
}

// NewEvaluationAgent returns a new agent for actual game play during evaluation.
func NewEvaluationAgent(mcts *searcher.MCTS) Agent {
	return evaluationAgent{mcts: mcts}
}

func (a evaluationAgent) FindMove(state game.State, updates []searcher.Segment) game.Move {
	policy := a.mcts.Simulate(state, updates)
	return findMax(policy)
}

func findMax(policy map[game.Move]float64) game.Move {
	var maxMove game.Move
	maxVisit := -1.0
	for move, visit := range policy {
		if visit > maxVisit {
			maxVisit = visit
			maxMove = move
		}
	}
	return maxMove
}

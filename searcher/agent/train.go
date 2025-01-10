package agent

import (
	"math"
	"math/rand/v2"
	"risk/game"
	"risk/searcher"
)

type trainingAgent struct {
	mcts *searcher.MCTS
}

// NewTrainingAgent returns a new agent for self-play during training.
func NewTrainingAgent(mcts *searcher.MCTS) Agent {
	return trainingAgent{mcts: mcts}
}

func (a trainingAgent) FindMove(state game.State, updates []searcher.Segment) game.Move {
	policy := a.mcts.Simulate(state, updates)
	// TODO: apply a temperature schedule as training progresses
	policy = adjustTemperature(policy, 1.0)
	return sample(policy)
}

func adjustTemperature(policy map[game.Move]float64, temperature float64) map[game.Move]float64 {
	// Compute temperature-adjusted move probabilities
	exponent := 1.0 / temperature
	sum := 0.0
	adjusted := make(map[game.Move]float64, len(policy))
	for move, visit := range policy {
		prob := math.Pow(float64(visit), exponent)
		sum += prob
		adjusted[move] = prob
	}
	// Normalize
	for move := range adjusted {
		adjusted[move] /= sum
	}
	return adjusted
}

func sample(policy map[game.Move]float64) game.Move {
	// TODO: seed randomizer globally for testing
	sampled := rand.Float64()
	cumulative := 0.0
	var lastMove game.Move
	for move, prob := range policy {
		lastMove = move
		cumulative += prob
		if sampled < cumulative {
			return move
		}
	}
	return lastMove // Fallback in case of rounding errors
}

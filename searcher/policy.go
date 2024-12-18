package searcher

import "math"

// Hyperparameters for MCTS

const CSquared = 2.0 // Exploration constant

const MaxCutoff = 10000 // Maximum rollout depth in moves

const Win = 1.0   // Reward for winning outcome
const Loss = -Win // Reward for loss outcome (negated from opponent perspective)

type uct struct {
	numerator float64
}

func newUCT(cSquared float64, N float64) *uct {
	if N == 0 {
		panic("N cannot be 0")
	}
	return &uct{numerator: cSquared * math.Log(N)}
}

func (u uct) evaluate(q float64, n float64) float64 {
	if n == 0 {
		panic("n cannot be 0")
	}
	// UCT = q/n + sqrt(c^2*ln(N)/n)
	return q/n + math.Sqrt(u.numerator/n)
}

func computeReward(player string, score float64, current string) float64 {
	if player == current {
		return score
	}
	return -score
}

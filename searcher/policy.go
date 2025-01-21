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

func newUCT(cSquared float64, parentVisits float64) *uct {
	if parentVisits == 0 {
		panic("parent visits cannot be 0")
	}
	return &uct{numerator: cSquared * math.Log(parentVisits)}
}

func (u uct) evaluate(rewards float64, childVisits float64) float64 {
	if childVisits == 0 {
		panic("child visits cannot be 0")
	}
	// UCT = q/n + sqrt(c^2*ln(N)/n)
	return rewards/childVisits + math.Sqrt(u.numerator/childVisits)
}

func computeReward(player string, score float64, current string) float64 {
	if player == current {
		return score
	}
	return -score
}

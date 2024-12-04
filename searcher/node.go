package searcher

import (
	"math"
	"risk/game"
)

type Node interface {
	SelectOrExpand(state game.State) (child Node, childState game.State, selected bool)
	Backup(winner string) Node
	Visits() int
	applyLoss()
	score(normalizer float64) float64
}

func uct(rewards float64, visits int, c2LnN float64) float64 {
	if visits == 0 { // Prevent division by zero
		panic("cannot compute UCT: 0 visits")
	}

	return rewards/float64(visits) + math.Sqrt(c2LnN/float64(visits))
}

func computeReward(winner string, player string) float64 {
	if player == winner {
		return WIN
	}
	return LOSS
}

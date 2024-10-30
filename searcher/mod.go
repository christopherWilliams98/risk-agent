package searcher

import (
	"math"
	"risk/game"
)

const C_SQUARED = 2.0

const WIN = 1.0
const LOSS = 0.0

type Node interface {
	PickChild(state game.State) (child Node, childState game.State, added bool)
	Update(reward func(string) float64) Node
	Value() int
	applyLoss()
	score(normalizer float64) float64
}

type MCTS interface {
	FindNextMove(state game.State) game.Move
}

func rewarder(winner string) func(player string) (reward float64) {
	return func(player string) float64 {
		if player == winner {
			return WIN
		}
		return LOSS
	}
}

func ucb1(rewards float64, visits int, c2LnN float64) float64 {
	// Prioritize unexplored nodes
	if visits == 0 {
		return math.Inf(1)
	}

	return rewards/float64(visits) + math.Sqrt(c2LnN/float64(visits))
}

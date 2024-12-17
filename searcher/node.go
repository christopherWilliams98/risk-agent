package searcher

import (
	"risk/game"
)

type Node interface {
	SelectOrExpand(state game.State) (child Node, childState game.State, selected bool)
	// Backup accumulates the reward from the simulation outcome to estimate the
	// expected game outcome from this node
	Backup(winner string) Node
	Policy() map[game.Move]float64
	stats() (player string, rewards float64, visits float64)
	applyLoss()
}

func computeReward(winner string, player string) float64 {
	if player == winner {
		return WIN
	}
	return LOSS
}

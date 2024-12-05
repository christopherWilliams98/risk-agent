package searcher

import (
	"risk/game"
)

type Node interface {
	SelectOrExpand(state game.State) (child Node, childState game.State, selected bool)
	Backup(winner string) Node
	stats() (player string, rewards float64, visits int)
	applyLoss()
}

func computeReward(winner string, player string) float64 {
	if player == winner {
		return WIN
	}
	return LOSS
}

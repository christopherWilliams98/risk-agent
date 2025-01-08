package searcher

import (
	"risk/game"
)

type Node interface {
	SelectOrExpand(state game.State) (child Node, childState game.State, selected bool)
	// Backup accumulates the reward from the simulation outcome to estimate the
	// expected game outcome from this node
	Backup(player string, score float64) Node
	Policy() map[game.Move]float64
	stats() (player string, rewards float64, visits float64)
	applyLoss()
}

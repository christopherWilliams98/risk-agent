package agent

import (
	"risk/game"
	"risk/searcher"
	"risk/searcher/metrics"
)

type Agent interface {
	// FindMove returns a move policy and performance metrics (if collected) from the simulation process
	FindMove(state game.State, updates []searcher.Segment) (game.Move, metrics.SearchMetrics)
}

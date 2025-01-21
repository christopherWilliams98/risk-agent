package engine

import "risk/experiments/metrics"

const MaxMoves = 10000

type Engine interface {
	// Run starts a game till there's a winner or a max number of moves is reached
	Run() (winner string, gameMetric metrics.GameMetric, moveMetrics []metrics.MoveMetric)
}

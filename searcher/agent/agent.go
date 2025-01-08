package agent

import (
	"risk/game"
	"risk/searcher"
)

type Agent interface {
	FindMove(state game.State, updates []searcher.Segment) game.Move
}

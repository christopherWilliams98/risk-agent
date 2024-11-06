package agent

import (
	"risk/game"
	"risk/searcher"
	"time"
)

func main() {
	uct := searcher.NewUCT(searcher.WithGoroutines(2), searcher.WithIterations(100), searcher.WithDuration(time.Second*10))
	// TODO: initialize game state from game master
	// TODO: wait till this player's turn
	var state game.State
	uct.FindNextMove(state)
	// TODO: send move to game master, then loop
}

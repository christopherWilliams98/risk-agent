package main

import (
	"risk/game"
	"risk/searcher"
	"time"
)

func main() {
	// TODO: parse player name from command flags

	// TODO: initialize game state from game master
	var state game.State

	uct := searcher.NewUCT(state, searcher.WithGoroutines(2), searcher.WithIterations(100), searcher.WithDuration(time.Second*10))

	// TODO: wait for update on the played move and the new state
	var move game.Move

	uct.UpdateRoot(move, state)

	uct.FindNextMove(state)

	// TODO: send move to game master, and wait for update
}

package main

import (
	"flag"
	"risk/gamemaster"
	agent "risk/player"
	"risk/searcher"
	"time"
)

func main2() {

	player := flag.String("player", "", "Player's name")
	numGoroutines := flag.Int("goroutines", 2, "Number of goroutines for parallel playouts")
	numIterations := flag.Int("iterations", 200, "Number of playouts per move")
	duration := flag.Duration("duration", time.Second*10, "Duration of playouts per move")
	flag.Parse()

	// TODO: either by iterations or by duration
	mcts := searcher.NewMCTS(searcher.WithGoroutines(*numGoroutines), searcher.WithIterations(*numIterations), searcher.WithDuration(*duration))
	engine := gamemaster.GetLocalEngine()

	controller := agent.NewTrainingController(*player, mcts, engine)
	controller.Run()
}

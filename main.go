package main

import (
	"fmt"
	"risk/engine"
	"risk/game"
	"risk/searcher"
	"risk/searcher/agent"
	"time"
)

type config struct {
	goroutines int
	episodes   int
	duration   time.Duration
	cutoff     int
}

func main() {
	runSpeedupExperiment()
}

func runSpeedupExperiment() {
	numGames := 10
	duration := 500 * time.Millisecond
	configs := []config{
		{goroutines: 1, duration: duration},
		// {goroutines: 2, duration: duration},
		// {goroutines: 4, duration: duration},
		{goroutines: 8, duration: duration},
		// {goroutines: 16, duration: duration},
		// {goroutines: 32, duration: duration},
		{goroutines: 64, duration: duration},
	}

	fmt.Printf("Running speedup experiment...\n")
	for _, cfg := range configs {
		fmt.Printf("Agent %+v vs Agent %+v:\n", cfg, cfg)
		for i := 0; i < numGames; i++ {
			fmt.Printf("Game %d started...\n", i+1)
			// Use the same config for both players for the same playing strength
			winner := runGame(cfg, cfg)
			fmt.Printf("Game %d over! Winner: %s\n", i+1, winner)
		}
	}
	fmt.Printf("Finished speedup experiment.\n")
}

// runGame executes a single game between two agents and returns the winner
func runGame(config1, config2 config) string {
	players := []string{"Player1", "Player2"}
	agents := []engine.MCTSAdapter{
		{InternalAgent: agent.NewEvaluationAgent(createMCTS(config1))},
		{InternalAgent: agent.NewEvaluationAgent(createMCTS(config2))},
	}
	m := game.CreateMap()
	rules := game.NewStandardRules()
	e := engine.LocalEngine(players, agents, m, rules)

	e.Run()

	return e.State.Winner()
}

func createMCTS(config config) *searcher.MCTS {
	options := []searcher.Option{}

	if config.episodes > 0 {
		options = append(options, searcher.WithEpisodes(config.episodes))
	}
	if config.duration > 0 {
		options = append(options, searcher.WithDuration(config.duration))
	}
	if config.cutoff > 0 {
		options = append(options, searcher.WithCutoff(config.cutoff))
	}

	return searcher.NewMCTS(config.goroutines, options...)
}

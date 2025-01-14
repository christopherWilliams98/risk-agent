package main

import (
	"fmt"
	"os"
	"time"

	"risk/engine"
	"risk/game"
	"risk/searcher"
	"risk/searcher/agent"
	"risk/searcher/metrics"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Caller().Logger()
}

func main() {
	runSpeedupExperiment()
}

func runSpeedupExperiment() {
	const NumGames = 10 // Per match up
	const Duration = 500 * time.Millisecond
	// Same config for both players in each game
	// for the same playing strength and similar game length
	matchups := [][]metrics.AgentConfig{
		{
			{Goroutines: 1, Duration: Duration},
			{Goroutines: 1, Duration: Duration},
		},
		{
			{Goroutines: 8, Duration: Duration},
			{Goroutines: 8, Duration: Duration},
		},
		{
			{Goroutines: 64, Duration: Duration},
			{Goroutines: 64, Duration: Duration},
		},
	}

	log.Info().Msg("starting speedup experiment...")

	writer, err := metrics.NewExperimentWriter(NumGames, matchups)
	if err != nil {
		panic(fmt.Sprintf("failed to create experiment writer: %v", err))
	}

	err = writer.WriteSetup()
	if err != nil {
		panic(fmt.Sprintf("failed to write experiment setup: %v", err))
	}
	log.Info().Msg("stored experiment setup")

	// Run NumGames for each matchup
	for _, matchup := range matchups {
		for i := 0; i < NumGames; i++ {
			log.Info().Msgf("starting game %d between agent1=%+v and agent2=%+v", i+1, matchup[0], matchup[1])

			winner, metrics := runGame(matchup[0], matchup[1])
			log.Info().Msgf("completed game %d with winner: %s", i+1, winner)

			err := writer.WriteMetrics(i+1, metrics)
			if err != nil {
				panic(fmt.Sprintf("failed to write metrics for game %d: %v", i+1, err))
			}
			log.Info().Msgf("stored metrics for game %d", i+1)
		}
	}

	log.Info().Msg("completed speedup experiment")
}

// runGame executes a single game between two agents and returns the winner
func runGame(config1, config2 metrics.AgentConfig) (string, metrics.GameMetrics) {
	players := []string{"Player1", "Player2"}
	agents := []engine.MCTSAdapter{
		{InternalAgent: agent.NewEvaluationAgent(createMCTS(config1))},
		{InternalAgent: agent.NewEvaluationAgent(createMCTS(config2))},
	}
	m := game.CreateMap()
	rules := game.NewStandardRules()
	e := engine.LocalEngine(players, agents, m, rules)

	winner, metrics := e.Run()

	return winner, metrics
}

func createMCTS(config metrics.AgentConfig) *searcher.MCTS {
	options := []searcher.Option{}

	if config.Episodes > 0 {
		options = append(options, searcher.WithEpisodes(config.Episodes))
	}
	if config.Duration > 0 {
		options = append(options, searcher.WithDuration(config.Duration))
	}
	if config.Cutoff > 0 {
		options = append(options, searcher.WithCutoff(config.Cutoff))
	}

	options = append(options, searcher.WithMetrics())
	return searcher.NewMCTS(config.Goroutines, options...)
}

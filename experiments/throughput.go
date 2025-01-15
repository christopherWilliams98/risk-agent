package experiments

import (
	"fmt"
	"risk/engine"
	"risk/experiments/metrics"
	"risk/game"
	"risk/searcher"
	"risk/searcher/agent"
	"time"

	"github.com/rs/zerolog/log"
)

func RunThroughputExperiment() {
	const NumGames = 1 // Per match up
	const Duration = 10 * time.Millisecond
	configs := []metrics.AgentConfig{
		{ID: 1, Goroutines: 1, Duration: Duration},
		{ID: 2, Goroutines: 2, Duration: Duration},
		{ID: 3, Goroutines: 4, Duration: Duration},
		{ID: 4, Goroutines: 8, Duration: Duration},
		{ID: 5, Goroutines: 16, Duration: Duration},
		{ID: 6, Goroutines: 32, Duration: Duration},
		{ID: 7, Goroutines: 64, Duration: Duration},
		{ID: 8, Goroutines: 128, Duration: Duration},
	}
	// Same config for both players in each game
	// for the same playing strength and similar game length
	matchUps := [][]metrics.AgentConfig{
		{configs[0], configs[0]},
		{configs[1], configs[1]},
	}

	// Store experiment metadata
	writer, err := metrics.NewWriter()
	if err != nil {
		panic(fmt.Sprintf("failed to create experiment writer: %v", err))
	}

	err = writer.WriteAgentConfigs(configs)
	if err != nil {
		panic(fmt.Sprintf("failed to store agent configs: %v", err))
	}

	// Run a number of games for each matchup
	count := 0
	gameRecords := []metrics.GameRecord{}
	moveRecords := []metrics.MoveRecord{}

	log.Info().Msg("starting speedup experiment...")

	for _, matchup := range matchUps {
		config1 := matchup[0]
		config2 := matchup[1]

		log.Info().Msgf("starting matchup between agent1=%+v and agent2=%+v...", config1, config2)

		for i := 0; i < NumGames; i++ {
			log.Info().Msgf("starting game %d of %d...", i+1, NumGames)

			winner, gameMetric, moveMetrics := runGame(config1, config2)
			count++
			gameRecords = append(gameRecords, metrics.GameRecord{
				ID:         count,
				Agent1:     config1.ID,
				Agent2:     config2.ID,
				GameMetric: gameMetric,
			})
			for _, mm := range moveMetrics {
				moveRecords = append(moveRecords, metrics.MoveRecord{
					Game:       count,
					MoveMetric: mm,
				})
			}

			log.Info().Msgf("completed game %d with winner: %s", i+1, winner)
		}
		log.Info().Msg("completed matchup")
	}

	log.Info().Msg("completed speedup experiment")

	// Store experiment results
	err = writer.WriteGameRecords(gameRecords)
	if err != nil {
		panic(fmt.Sprintf("failed to write game records: %v", err))
	}
	log.Info().Msg("stored game records")

	err = writer.WriteMoveRecords(moveRecords)
	if err != nil {
		panic(fmt.Sprintf("failed to write move records: %v", err))
	}
	log.Info().Msg("stored move records")
}

// runGame executes a single game between two agents and returns the winner
func runGame(config1, config2 metrics.AgentConfig) (string, metrics.GameMetric, []metrics.MoveMetric) {
	players := []string{"Player1", "Player2"}
	agents := []engine.MCTSAdapter{
		{InternalAgent: agent.NewEvaluationAgent(createMCTS(config1))},
		{InternalAgent: agent.NewEvaluationAgent(createMCTS(config2))},
	}
	m := game.CreateMap()
	rules := game.NewStandardRules()
	e := engine.LocalEngine(players, agents, m, rules)

	winner, gameMetric, moveMetrics := e.Run()

	return winner, gameMetric, moveMetrics
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

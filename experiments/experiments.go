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

const (
	NumGames   = 30 // Per match up
	TimeBudget = 10 * time.Millisecond
)

var parallelConfigs = []metrics.AgentConfig{
	{ID: 1, Goroutines: 1, Duration: TimeBudget},
	{ID: 2, Goroutines: 4, Duration: TimeBudget},
	{ID: 3, Goroutines: 8, Duration: TimeBudget},
	{ID: 4, Goroutines: 16, Duration: TimeBudget},
	{ID: 5, Goroutines: 32, Duration: TimeBudget},
	{ID: 6, Goroutines: 64, Duration: TimeBudget},
	{ID: 7, Goroutines: 128, Duration: TimeBudget},
}

// TODO: remove
func RunParallelizationToThroughput() {
	// Each matchup uses the same config for both players
	// for the same playing strength and similar game length
	matchUps := [][]metrics.AgentConfig{}
	for _, config := range parallelConfigs {
		matchUps = append(matchUps, []metrics.AgentConfig{config, config})
	}

	runExperiment("parallelization_to_throughput", parallelConfigs, matchUps)
}

// TODO: rename RunParallelizationExperiment()
func RunParallelizationToStrength() {
	// Each matchup pairs an agent against the baseline sequential agent
	baseline := metrics.AgentConfig{ID: 0, Goroutines: 1, Duration: TimeBudget}
	matchUps := [][]metrics.AgentConfig{}
	for _, config := range parallelConfigs {
		// TODO: alternate starting agent
		matchUps = append(matchUps, []metrics.AgentConfig{baseline, config})
	}

	runExperiment("parallelization_to_strength", append(parallelConfigs, baseline), matchUps)
}

func RunCutoffExperiment() {
	// TODO: get 8-goroutine game length distribution/quartiles
	baseline := metrics.AgentConfig{ID: 0, Goroutines: 8, Duration: TimeBudget} // Without cutoff (full playout)
	cutoffConfigs := []metrics.AgentConfig{
		{ID: 1, Goroutines: baseline.Goroutines, Duration: baseline.Duration}, // Baseline equivalent
		{ID: 2, Goroutines: baseline.Goroutines, Duration: baseline.Duration, Cutoff: 10},
		{ID: 3, Goroutines: baseline.Goroutines, Duration: baseline.Duration, Cutoff: 75},  // Lower quartile
		{ID: 4, Goroutines: baseline.Goroutines, Duration: baseline.Duration, Cutoff: 150}, // Median
		{ID: 5, Goroutines: baseline.Goroutines, Duration: baseline.Duration, Cutoff: 200}, // Upper quartile
	}

	// Each matchup pairs the baseline agent against a cutoff agent
	matchUps := [][]metrics.AgentConfig{}
	for _, config := range cutoffConfigs {
		matchUps = append(matchUps, []metrics.AgentConfig{baseline, config})
	}

	runExperiment("cutoff", cutoffConfigs, matchUps)
}

func runExperiment(name string, configs []metrics.AgentConfig, matchUps [][]metrics.AgentConfig) {
	// Run a number of games for each matchup
	count := 0
	gameRecords := []metrics.GameRecord{}
	moveRecords := []metrics.MoveRecord{}

	log.Info().Msgf("starting %s experiment...", name)

	for mi, matchup := range matchUps {
		config1 := matchup[0]
		config2 := matchup[1]

		log.Info().Msgf("starting matchup %d of %d between agent1=%+v and agent2=%+v...", mi+1, len(matchUps), config1, config2)

		for i := 0; i < NumGames; i++ {
			log.Info().Msgf("starting matchup %d of %d game %d of %d...", mi+1, len(matchUps), i+1, NumGames)

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

			log.Info().Msgf("completed matchup %d of %d game %d with winner: %s", mi+1, len(matchUps), i+1, winner)
		}
		log.Info().Msgf("completed matchup %d of %d", mi+1, len(matchUps))
	}

	log.Info().Msgf("completed %s experiment", name)

	// TODO: extract into function
	// Store experiment metadata
	writer, err := metrics.NewWriter(name)
	if err != nil {
		panic(fmt.Sprintf("failed to create experiment writer: %v", err))
	}

	err = writer.WriteAgentConfigs(configs)
	if err != nil {
		panic(fmt.Sprintf("failed to store agent configs: %v", err))
	}
	log.Info().Msg("stored agent configs")

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
	if config.Evaluate != nil {
		options = append(options, searcher.WithEvaluationFn(config.Evaluate))
	}

	options = append(options, searcher.WithMetrics())
	return searcher.NewMCTS(config.Goroutines, options...)
}

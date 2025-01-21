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
	NumGames   = 3 // Per match up
	TimeBudget = 10 * time.Millisecond
)

func RunParallelismExperiment() {
	// Pairs the baseline agent against each experiment agent
	baseline := metrics.AgentConfig{ID: 0, Goroutines: 1, Duration: TimeBudget}
	expConfigs := []metrics.AgentConfig{
		{ID: 1, Goroutines: baseline.Goroutines, Duration: baseline.Duration},
		{ID: 2, Goroutines: 4, Duration: baseline.Duration},
		{ID: 3, Goroutines: 8, Duration: baseline.Duration},
		{ID: 4, Goroutines: 16, Duration: baseline.Duration},
		{ID: 5, Goroutines: 32, Duration: baseline.Duration},
		{ID: 6, Goroutines: 64, Duration: baseline.Duration},
		{ID: 7, Goroutines: 128, Duration: baseline.Duration},
	}
	var matchUps [][]metrics.AgentConfig
	for _, config := range expConfigs {
		// TODO: alternate starting agent
		matchUps = append(matchUps, []metrics.AgentConfig{baseline, config})
	}

	runExperiment("parallelism", append(expConfigs, baseline), matchUps)
}

// TODO: get SelectedConcurrency-goroutine's game length distribution/quartiles
const (
	SelectedConcurrency = 8
	LowCutoff           = 10
	LowerQuartileLength = 75
	MedianLength        = 150
	UpperQuartileLength = 200
)

func RunCutoffExperiment() {
	baseline := metrics.AgentConfig{ID: 0, Goroutines: SelectedConcurrency, Duration: TimeBudget} // Without cutoff (full playout)
	expConfigs := []metrics.AgentConfig{
		{ID: 1, Goroutines: baseline.Goroutines, Duration: baseline.Duration}, // Baseline equivalent
		{ID: 2, Goroutines: baseline.Goroutines, Duration: baseline.Duration, Cutoff: LowCutoff, Evaluate: game.EvaluateResources},
		{ID: 3, Goroutines: baseline.Goroutines, Duration: baseline.Duration, Cutoff: LowerQuartileLength, Evaluate: game.EvaluateResources},
		{ID: 4, Goroutines: baseline.Goroutines, Duration: baseline.Duration, Cutoff: MedianLength, Evaluate: game.EvaluateResources},
		{ID: 5, Goroutines: baseline.Goroutines, Duration: baseline.Duration, Cutoff: UpperQuartileLength, Evaluate: game.EvaluateResources},
	}

	// Pairs the baseline agent against each experiment agent
	var matchUps [][]metrics.AgentConfig
	for _, config := range expConfigs {
		matchUps = append(matchUps, []metrics.AgentConfig{baseline, config})
	}

	runExperiment("cutoff", expConfigs, matchUps)
}

func RunEvaluationExperiment() {
	baseline := metrics.AgentConfig{ID: 0, Goroutines: SelectedConcurrency, Duration: TimeBudget} // Without cutoff (full playout)
	expConfigs := []metrics.AgentConfig{
		{ID: 1, Goroutines: baseline.Goroutines, Duration: baseline.Duration}, // Baseline equivalent
		{ID: 2, Goroutines: baseline.Goroutines, Duration: baseline.Duration, Cutoff: LowCutoff, Evaluate: game.EvaluateBorderStrength},
		{ID: 3, Goroutines: baseline.Goroutines, Duration: baseline.Duration, Cutoff: LowerQuartileLength, Evaluate: game.EvaluateBorderStrength},
		{ID: 4, Goroutines: baseline.Goroutines, Duration: baseline.Duration, Cutoff: MedianLength, Evaluate: game.EvaluateBorderStrength},
		{ID: 5, Goroutines: baseline.Goroutines, Duration: baseline.Duration, Cutoff: UpperQuartileLength, Evaluate: game.EvaluateBorderStrength},
	}

	// Pairs the baseline agent against each experiment agent
	var matchUps [][]metrics.AgentConfig
	for _, config := range expConfigs {
		matchUps = append(matchUps, []metrics.AgentConfig{baseline, config})
	}

	runExperiment("evaluation", expConfigs, matchUps)
}

const StrongestCutoff = LowCutoff // TODO: pick cutoff depth with the highest playing strength from cutoff experiment

func RunEloExperiment() {
	configs := []metrics.AgentConfig{
		{ID: 1, Goroutines: 1, Duration: TimeBudget},
		{ID: 2, Goroutines: SelectedConcurrency, Duration: TimeBudget},
		{ID: 3, Goroutines: SelectedConcurrency, Duration: TimeBudget, Cutoff: StrongestCutoff, Evaluate: game.EvaluateResources},
		{ID: 4, Goroutines: SelectedConcurrency, Duration: TimeBudget, Cutoff: StrongestCutoff, Evaluate: game.EvaluateBorderStrength},
	}

	// Run games between each pair of agents in round-robin
	var matchUps [][]metrics.AgentConfig
	for i, config1 := range configs {
		for _, config2 := range configs[i+1:] {
			matchUps = append(matchUps, []metrics.AgentConfig{config1, config2})
		}
	}

	runExperiment("elo", configs, matchUps)
}

func runExperiment(name string, configs []metrics.AgentConfig, matchUps [][]metrics.AgentConfig) {
	// Run a number of games for each matchup
	count := 0
	var gameRecords []metrics.GameRecord
	var moveRecords []metrics.MoveRecord

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
	agents := []agent.Agent{
		agent.NewEvaluationAgent(createMCTS(config1)),
		agent.NewEvaluationAgent(createMCTS(config2)),
	}
	e := engine.NewLocalEngine(agents)
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

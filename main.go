package main

import (
	"os"
	"risk/experiments"
	"runtime"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Caller().Logger()
}

func main() {
	log.Info().Msgf("number of CPUs: %d", runtime.NumCPU())

	experiments.RunParallelismExperiment()
	experiments.RunCutoffExperiment()
	experiments.RunEvaluationExperiment()
	experiments.RunEloExperiment()
}

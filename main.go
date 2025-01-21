package main

import (
	"os"
	"risk/experiments"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Caller().Logger()
}

func main() {
	experiments.RunParallelismExperiment()
	// experiments.RunCutoffExperiment()
	// experiments.RunEvaluationExperiment()
	// experiments.RunEloExperiment()
}

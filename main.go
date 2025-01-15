package main

import (
	"os"
	"time"

	"risk/experiments"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Caller().Logger()
}

func main() {
	// experiments.RunThroughputExperiment()
	experiments.RunVolumeExperiment()
}

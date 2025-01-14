package experiments

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	Goroutines int
	Duration   time.Duration
	Episodes   int
	Cutoff     int
}

type Setup struct {
	Matchups  [][]Config    `json:"matchups"`
	NumGames  int           `json:"numGames"` // per matchup
	StartTime time.Time     `json:"startTime"`
	EndTime   time.Time     `json:"endTime"`
	Duration  time.Duration `json:"duration"`
}

type Writer struct {
	baseDir string
}

func NewWriter() (*Writer, error) {
	// Create subfolder named by timestamp
	timestamp := time.Now().UTC().Format(time.RFC3339)
	baseDir := filepath.Join("experiments", "speedup", timestamp)

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	return &Writer{
		baseDir: baseDir,
	}, nil
}

func (w *Writer) WriteSetup(start, end time.Time, matchups [][]Config, numGames int) error {
	setup := Setup{
		Matchups:  matchups,
		NumGames:  numGames,
		StartTime: start,
		EndTime:   end,
		Duration:  end.Sub(start),
	}

	setupPath := filepath.Join(w.baseDir, "setup.json")
	f, err := os.Create(setupPath)
	if err != nil {
		return fmt.Errorf("failed to create setup file: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(setup); err != nil {
		return fmt.Errorf("failed to write setup: %w", err)
	}

	return nil
}

func (w *Writer) WriteMetrics(game int, metrics GameMetrics) error {
	filename := fmt.Sprintf("game%d.csv", game)
	filepath := filepath.Join(w.baseDir, filename)

	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create metrics file for game %d: %w", game, err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	header := []string{"step", "player", "duration", "episodes", "fullPlayouts", "isTreeReused"}
	err = writer.Write(header)
	if err != nil {
		return fmt.Errorf("failed to write metrics header: %w", err)
	}

	for _, moveMetric := range metrics {
		record := []string{
			strconv.Itoa(moveMetric.Step),
			strconv.Itoa(moveMetric.Player),
			moveMetric.Duration.String(),
			strconv.FormatInt(moveMetric.Episodes, 10),
			strconv.FormatInt(moveMetric.FullPlayouts, 10),
			strconv.FormatBool(moveMetric.IsTreeReused),
		}
		err := writer.Write(record)
		if err != nil {
			return fmt.Errorf("failed to write metric: %w", err)
		}
	}
	return nil
}

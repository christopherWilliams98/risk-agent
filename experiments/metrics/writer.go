package metrics

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type GameRecord struct {
	ID     int
	Agent1 int // AgentConfig.ID
	Agent2 int // AgentConfig.ID
	GameMetric
}

type MoveRecord struct {
	Game int // GameMetric.ID
	MoveMetric
}

type Writer struct {
	baseDir string
}

func NewWriter() (*Writer, error) {
	// Create a subfolder named by current timestamp
	timestamp := time.Now().UTC().Format(time.RFC3339)
	baseDir := filepath.Join("experiments", "speedup", timestamp)
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	return &Writer{
		baseDir: baseDir,
	}, nil
}

func (w *Writer) WriteAgentConfigs(configs []AgentConfig) error {
	// Create a file
	path := filepath.Join(w.baseDir, "agent_configs.csv")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create agent configs file: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Write header
	header := []string{"id", "goroutines", "duration", "episodes", "cutoff"}
	err = writer.Write(header)
	if err != nil {
		return fmt.Errorf("failed to write agent configs header: %w", err)
	}

	// Write each row
	for _, config := range configs {
		row := []string{
			strconv.Itoa(config.ID),
			strconv.Itoa(config.Goroutines),
			config.Duration.String(),
			strconv.Itoa(config.Episodes),
			strconv.Itoa(config.Cutoff),
		}
		err = writer.Write(row)
		if err != nil {
			return fmt.Errorf("failed to write agent config row: %w", err)
		}
	}

	return nil
}

func (w *Writer) WriteGameRecords(records []GameRecord) error {
	// Create a file
	path := filepath.Join(w.baseDir, "game_records.csv")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create game records file: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Write header
	header := []string{"id", "agent1", "agent2", "starting_player", "winner", "start_time", "end_time", "duration"}
	err = writer.Write(header)
	if err != nil {
		return fmt.Errorf("failed to write game records header: %w", err)
	}

	// Write each row
	for _, record := range records {
		row := []string{
			strconv.Itoa(record.ID),
			strconv.Itoa(record.Agent1),
			strconv.Itoa(record.Agent2),
			strconv.Itoa(record.StartingPlayer),
			record.Winner,
			record.StartTime.Format(time.RFC3339),
			record.EndTime.Format(time.RFC3339),
			record.Duration.String(),
		}
		err = writer.Write(row)
		if err != nil {
			return fmt.Errorf("failed to write game record row: %w", err)
		}
	}

	return nil
}

func (w *Writer) WriteMoveRecords(records []MoveRecord) error {
	// Create a file
	path := filepath.Join(w.baseDir, "move_records.csv")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create move records file: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Write header
	header := []string{"game", "step", "player", "duration", "episodes", "full_playouts", "is_tree_reused"}
	err = writer.Write(header)
	if err != nil {
		return fmt.Errorf("failed to write move records header: %w", err)
	}

	// Write each row
	for _, record := range records {
		row := []string{
			strconv.Itoa(record.Game),
			strconv.Itoa(record.Step),
			strconv.Itoa(record.Player),
			record.Duration.String(),
			strconv.Itoa(record.Episodes),
			strconv.Itoa(record.FullPlayouts),
			strconv.FormatBool(record.IsTreeReused),
		}
		err = writer.Write(row)
		if err != nil {
			return fmt.Errorf("failed to write move record row: %w", err)
		}
	}

	return nil
}

package metrics

import (
	"risk/game"
	"sync/atomic"
	"time"
)

type SearchMetric struct {
	Goroutines   int
	Duration     time.Duration
	Episodes     int
	Cutoff       int
	Evaluate     game.Evaluate
	FullPlayouts int
	IsTreeReset  bool
}

type MoveMetric struct {
	Step   int
	Player int // Player ID
	SearchMetric
}

type GameMetric struct {
	StartingPlayer int    // Player ID
	Winner         string // Player ID
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	TotalMoves     int
}

type Collector interface {
	Start(goroutines, cutoff int, evaluate game.Evaluate)
	SetTreeReset(value bool)
	AddFullPlayout()
	AddEpisode()
	Complete() SearchMetric
}

type collector struct {
	goroutines   int
	cutoff       int
	evaluate     game.Evaluate
	startTime    time.Time
	episodes     atomic.Int32
	fullPlayouts atomic.Int32
	isTreeReset  atomic.Bool
}

func NewCollector() Collector {
	return &collector{}
}

func (m *collector) SetTreeReset(value bool) {
	m.isTreeReset.Store(value)
}

func (m *collector) Start(goroutines, cutoff int, evaluate game.Evaluate) {
	m.startTime = time.Now()
	m.goroutines = goroutines
	m.cutoff = cutoff
	m.evaluate = evaluate
}

func (m *collector) AddFullPlayout() {
	m.fullPlayouts.Add(1)
}

func (m *collector) AddEpisode() {
	m.episodes.Add(1)
}

func (m *collector) Complete() SearchMetric {
	return SearchMetric{
		Goroutines:   m.goroutines,
		Duration:     time.Since(m.startTime),
		Episodes:     int(m.episodes.Load()),
		FullPlayouts: int(m.fullPlayouts.Load()),
		Cutoff:       m.cutoff,
		Evaluate:     m.evaluate,
		IsTreeReset:  m.isTreeReset.Load(),
	}
}

type dummyCollector struct{}

func NewDummyCollector() Collector {
	return &dummyCollector{}
}

func (m *dummyCollector) Start(goroutines, cutoff int, evaluate game.Evaluate) {}
func (m *dummyCollector) SetTreeReset(value bool)                              {}
func (m *dummyCollector) AddFullPlayout()                                      {}
func (m *dummyCollector) AddEpisode()                                          {}
func (m *dummyCollector) Complete() SearchMetric                               { return SearchMetric{} }

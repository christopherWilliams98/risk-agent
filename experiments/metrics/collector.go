package metrics

import (
	"sync/atomic"
	"time"
)

type AgentConfig struct {
	ID         int
	Goroutines int
	Duration   time.Duration
	Episodes   int
	Cutoff     int
}

type SearchMetric struct {
	Duration     time.Duration
	Episodes     int
	FullPlayouts int
	IsTreeReused bool
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
}

type Collector interface {
	Start()
	AddFullPlayout()
	AddEpisode()
	ReusedTree()
	Complete() SearchMetric
}

type collector struct {
	startTime    time.Time
	episodes     atomic.Int32
	fullPlayouts atomic.Int32
	isTreeReused bool
}

func NewCollector() Collector {
	return &collector{}
}

func (m *collector) Start() {
	m.startTime = time.Now()
}

func (m *collector) AddFullPlayout() {
	m.fullPlayouts.Add(1)
}

func (m *collector) AddEpisode() {
	m.episodes.Add(1)
}

func (m *collector) ReusedTree() {
	m.isTreeReused = true
}

func (m *collector) Complete() SearchMetric {
	return SearchMetric{
		Duration:     time.Since(m.startTime),
		Episodes:     int(m.episodes.Load()),
		FullPlayouts: int(m.fullPlayouts.Load()),
		IsTreeReused: m.isTreeReused,
	}
}

type dummyCollector struct{}

func NewDummyCollector() Collector {
	return &dummyCollector{}
}

func (m *dummyCollector) Start()                 {}
func (m *dummyCollector) AddFullPlayout()        {}
func (m *dummyCollector) AddEpisode()            {}
func (m *dummyCollector) ReusedTree()            {}
func (m *dummyCollector) Complete() SearchMetric { return SearchMetric{} }

package experiments

import (
	"sync/atomic"
	"time"
)

type SearchMetrics struct {
	Duration     time.Duration
	Episodes     int64
	FullPlayouts int64
	IsTreeReused bool
}

type MoveMetrics struct {
	Step   int
	Player int
	SearchMetrics
}

type GameMetrics []MoveMetrics

type MetricsCollector interface {
	Start()
	AddFullPlayout()
	AddEpisode()
	ReusedTree()
	Complete() SearchMetrics
}

type metricsCollector struct {
	startTime    time.Time
	episodes     atomic.Int64
	fullPlayouts atomic.Int64
	isTreeReused bool
}

func NewMetricsCollector() MetricsCollector {
	return &metricsCollector{}
}

func (m *metricsCollector) Start() {
	m.startTime = time.Now()
}

func (m *metricsCollector) AddFullPlayout() {
	m.fullPlayouts.Add(1)
}

func (m *metricsCollector) AddEpisode() {
	m.episodes.Add(1)
}

func (m *metricsCollector) ReusedTree() {
	m.isTreeReused = true
}

func (m *metricsCollector) Complete() SearchMetrics {
	return SearchMetrics{
		Duration:     time.Since(m.startTime),
		Episodes:     m.episodes.Load(),
		FullPlayouts: m.fullPlayouts.Load(),
		IsTreeReused: m.isTreeReused,
	}
}

type noMetricsCollector struct{}

func NewNoMetricsCollector() MetricsCollector {
	return &noMetricsCollector{}
}

func (m *noMetricsCollector) Start()                  {}
func (m *noMetricsCollector) AddFullPlayout()         {}
func (m *noMetricsCollector) AddEpisode()             {}
func (m *noMetricsCollector) ReusedTree()             {}
func (m *noMetricsCollector) Complete() SearchMetrics { return SearchMetrics{} }

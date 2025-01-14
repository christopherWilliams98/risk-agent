package searcher

import (
	"sync/atomic"
	"time"
)

type MoveMetrics struct {
	StartTime    time.Time
	Duration     time.Duration
	Episodes     int64
	FullPlayouts int64
	TreeReused   bool
}

type MetricsCollector interface {
	Start()
	AddFullPlayout()
	AddEpisode()
	ReusedTree()
	Complete() MoveMetrics
}

type metricsCollector struct {
	startTime    time.Time
	episodes     atomic.Int64
	fullPlayouts atomic.Int64
	treeReused   atomic.Bool
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
	m.treeReused.Store(true)
}

func (m *metricsCollector) Complete() MoveMetrics {
	return MoveMetrics{
		StartTime:    m.startTime,
		Duration:     time.Since(m.startTime),
		Episodes:     m.episodes.Load(),
		FullPlayouts: m.fullPlayouts.Load(),
		TreeReused:   m.treeReused.Load(),
	}
}

type noMetricsCollector struct{}

func NewNoMetricsCollector() MetricsCollector {
	return &noMetricsCollector{}
}

func (m *noMetricsCollector) Start()                {}
func (m *noMetricsCollector) AddFullPlayout()       {}
func (m *noMetricsCollector) AddEpisode()           {}
func (m *noMetricsCollector) ReusedTree()           {}
func (m *noMetricsCollector) Complete() MoveMetrics { return MoveMetrics{} }

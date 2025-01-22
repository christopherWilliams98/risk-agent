package searcher

import (
	"risk/experiments/metrics"
	"risk/game"
	"sync"
	"time"

	"math/rand"

	"github.com/rs/zerolog/log"
)

type Option func(mcts *MCTS)

type Segment struct {
	Move      game.Move
	StateHash game.StateHash
}

type MCTS struct {
	goroutines int
	duration   time.Duration
	episodes   int
	cutoff     int
	evaluate   game.Evaluate
	root       *decision
	metrics    metrics.Collector
}

func WithDuration(duration time.Duration) Option {
	return func(u *MCTS) {
		if duration > 0 {
			u.duration = duration
		}
	}
}

func WithEpisodes(episodes int) Option {
	return func(u *MCTS) {
		if episodes > 0 {
			u.episodes = episodes
		}
	}
}

func WithCutoff(depth int) Option {
	return func(u *MCTS) {
		if depth > 0 {
			u.cutoff = depth
		}
	}
}

func WithEvaluationFn(evaluate game.Evaluate) Option {
	return func(m *MCTS) {
		if evaluate != nil {
			m.evaluate = evaluate
		}
	}
}

func WithMetrics() Option {
	return func(m *MCTS) {
		m.metrics = metrics.NewCollector()
	}
}

func NewMCTS(goroutines int, options ...Option) *MCTS {
	m := &MCTS{ // Default values
		goroutines: goroutines,
		cutoff:     MaxCutoff,
		evaluate:   game.EvaluateResources,
		metrics:    metrics.NewDummyCollector(),
	}
	for _, option := range options {
		option(m)
	}
	if m.episodes <= 0 && m.duration <= 0 {
		panic("Must specify search episodes or duration")
	}
	return m
}

func (m *MCTS) Simulate(state game.State, lineage []Segment) (map[game.Move]float64, metrics.SearchMetric) {
	m.root = newDecision(nil, state)
	m.metrics.SetTreeReset(true)

	// Run simulations to collect statistics
	m.metrics.Start(m.goroutines, m.cutoff, m.evaluate)
	if m.episodes > 0 {
		m.iterate(state)
	} else if m.duration > 0 {
		m.countdown(state)
	} else {
		panic("Must specify search episodes or duration")
	}
	metric := m.metrics.Complete()

	// Output move policy and move finding metrics
	policy := m.root.Policy()
	return policy, metric
}

func (m *MCTS) iterate(state game.State) {
	task := make(chan any, m.episodes)
	for i := 0; i < m.episodes; i++ {
		task <- nil
	}
	close(task)

	var wg sync.WaitGroup
	for i := 0; i < m.goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for range task {
				m.simulate(state)
				m.metrics.AddEpisode()
			}
		}()
	}

	wg.Wait()
}

func (m *MCTS) countdown(state game.State) {
	done := make(chan any)

	for i := 0; i < m.goroutines; i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					m.simulate(state)
					m.metrics.AddEpisode()
				}
			}
		}()
	}

	<-time.After(m.duration)
	close(done)
}

func (m *MCTS) findRoot(path []Segment, state game.State) {
	root := traverse(m.root, path)
	if root == nil {
		m.root = newDecision(nil, state)
		m.metrics.SetTreeReset(true)
	} else {
		root.parent = nil
		m.root = root
		m.metrics.SetTreeReset(false)
	}
}

func traverse(root *decision, path []Segment) *decision {
	if root == nil {
		return nil
	}

	node := root
	for _, segment := range path {
		child, ok := node.children[segment.Move]
		if !ok { // Node has not expanded this move
			return nil
		}

		switch child := child.(type) {
		case *decision:
			if child.hash != segment.StateHash {
				log.Warn().Msgf("node's state hash %d does not match segment's state hash %d", child.hash, segment.StateHash)
				return nil
			}
			node = child
		case *chance:
			grandChild := child.selects(segment.StateHash)
			if grandChild == nil {
				return nil
			}
			node = grandChild
		default:
			panic("Unexpected node type")
		}
	}
	return node
}

func (m *MCTS) simulate(state game.State) {
	newNode, newState := selectThenExpand(m.root, state)
	player, score := rollout(newState, m.cutoff, m.evaluate, m.metrics)
	backup(newNode, player, score)
}

func selectThenExpand(root Node, state game.State) (Node, game.State) {
	parent := root
	child, state, selected := parent.SelectOrExpand(state)
	for selected && (child != parent) {
		parent = child
		child, state, selected = parent.SelectOrExpand(state)
	}
	return child, state
}

func rollout(state game.State, cutoff int, evaluate game.Evaluate, metrics metrics.Collector) (string, float64) {
	depth := 0
	moves := state.LegalMoves()
	// Rollout till game over or for cutoff number of moves
	for len(moves) > 0 && (depth < cutoff) {
		move := moves[rand.Intn(len(moves))] // Random rollout policy
		state = state.Play(move)
		moves = state.LegalMoves()
		depth++
	}

	if len(moves) == 0 { // Game over before cutoff
		metrics.AddFullPlayout()
		return state.Winner(), Win
	}

	// At cutoff state, return an evaluation score from current player's perspective
	return state.Player(), evaluate(state)
}

func backup(newNode Node, player string, score float64) {
	node := newNode
	for node != nil {
		parent := node.Backup(player, score)
		node = parent
	}
}

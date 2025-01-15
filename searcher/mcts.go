package searcher

import (
	"risk/experiments/metrics"
	"risk/game"
	"sync"
	"time"

	"golang.org/x/exp/rand"
)

type Option func(mcts *MCTS)

type Segment struct {
	Move  game.Move
	State game.State
}

type MCTS struct {
	goroutines int
	duration   time.Duration
	episodes   int
	cutoff     int
	evaluate   game.Evaluate
	root       Node
	metrics    metrics.Collector
}

func WithDuration(duration time.Duration) Option {
	return func(u *MCTS) {
		u.duration = duration
	}
}

func WithEpisodes(episodes int) Option {
	return func(u *MCTS) {
		u.episodes = episodes
	}
}

func WithCutoff(depth int) Option {
	return func(u *MCTS) {
		u.cutoff = depth
	}
}

func WithEvaluationFn(evaluate game.Evaluate) Option {
	return func(m *MCTS) {
		m.evaluate = evaluate
	}
}

func WithMetrics() Option {
	return func(m *MCTS) {
		m.metrics = metrics.NewCollector(m.goroutines)
	}
}

func NewMCTS(goroutines int, options ...Option) *MCTS {
	m := &MCTS{
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
	m.metrics.Start()

	// Reuse subtree if possible
	m.root = m.findSubtree(lineage, state, m.metrics)

	// Run simulations to collect statistics
	if m.episodes > 0 {
		iterate(m.goroutines, m.episodes, m.cutoff, m.root, state, m.evaluate, m.metrics)
	} else if m.duration > 0 {
		countdown(m.goroutines, m.duration, m.cutoff, m.root, state, m.evaluate, m.metrics)
	} else {
		panic("Must specify search episodes or duration")
	}

	// Output move policy and move finding metrics
	metrics := m.metrics.Complete()
	policy := m.root.Policy()
	return policy, metrics
}

// TODO: make method
func iterate(goroutines int, episodes int, cutoff int, root Node, state game.State, evaluate game.Evaluate, metrics metrics.Collector) {
	task := make(chan any, episodes)
	for i := 0; i < episodes; i++ {
		task <- nil
	}
	close(task)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			for range task {
				simulate(root, state, cutoff, evaluate, metrics)
				metrics.AddEpisode()
			}
		}()
	}

	wg.Wait()
}

// TODO: make method
func countdown(goroutines int, duration time.Duration, cutoff int, root Node, state game.State, evaluate game.Evaluate, metrics metrics.Collector) {
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for time.Since(start) < duration {
				simulate(root, state, cutoff, evaluate, metrics)
				metrics.AddEpisode()
			}
		}()
	}

	wg.Wait()
}

// TODO: make function rather than method
func (m *MCTS) findSubtree(path []Segment, state game.State, metrics metrics.Collector) Node {
	gs := state.(*game.GameState)

	if m.root == nil {
		return newDecision(nil, state)
	}
	if len(path) == 0 {
		// If root's phase doesn't match the current state's phase, discard it:
		rootDecision := m.root.(*decision)
		if rootDecision.phase != gs.Phase {
			return newDecision(nil, state)
		}
		metrics.ReusedTree()
		return m.root
	}

	node := m.root.(*decision)
	for _, segment := range path {
		// If mismatch in phase, discard
		if node.phase != gs.Phase {
			return newDecision(nil, state)
		}

		child := node.children[segment.Move]
		if child == nil {
			return newDecision(nil, state)
		}

		switch child := child.(type) {
		case *decision:
			node = child
		case *chance:
			grandChild := child.selects(segment.State)
			if grandChild == nil {
				return newDecision(nil, state)
			}
			node = grandChild
		default:
			panic("Unexpected node type")
		}
	}

	// final check
	if node.phase != gs.Phase {
		return newDecision(nil, state)
	}

	// Return the node at the end of the path as the new root
	node.parent = nil
	metrics.ReusedTree()
	return node
}

func simulate(root Node, state game.State, cutoff int, evaluate game.Evaluate, metrics metrics.Collector) {
	newNode, newState := selectThenExpand(root, state)
	player, score := rollout(newState, cutoff, evaluate, metrics)
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

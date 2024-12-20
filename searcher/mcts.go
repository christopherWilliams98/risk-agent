package searcher

import (
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
	episodes   int
	duration   time.Duration
	cutoff     int
	root       Node
}

func WithEpisodes(episodes int) Option {
	return func(u *MCTS) {
		u.episodes = episodes
	}
}

func WithDuration(duration time.Duration) Option {
	return func(u *MCTS) {
		u.duration = duration
	}
}

func WithCutoff(depth int) Option {
	return func(u *MCTS) {
		u.cutoff = depth
	}
}

func NewMCTS(goroutines int, options ...Option) *MCTS {
	u := &MCTS{goroutines: goroutines, cutoff: MaxCutoff}
	for _, option := range options {
		option(u)
	}
	if u.episodes <= 0 && u.duration <= 0 {
		panic("Must specify search episodes or duration")
	}
	return u
}

func (m *MCTS) Simulate(state game.State, lineage []Segment) map[game.Move]float64 {
	// Reuse subtree if possible
	m.root = m.findSubtree(lineage, state)
	// Run simulations to collect statistics
	if m.episodes > 0 {
		iterate(m.goroutines, m.episodes, m.cutoff, m.root, state)
	} else if m.duration > 0 {
		countdown(m.goroutines, m.duration, m.cutoff, m.root, state)
	} else {
		panic("Must specify search episodes or duration")
	}
	// Return visit counts at the root node
	return m.root.Policy()
}

func iterate(goroutines int, episodes int, cutoff int, root Node, state game.State) {
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
				simulate(root, state, cutoff)
			}
		}()
	}

	wg.Wait()
}

func countdown(goroutines int, duration time.Duration, cutoff int, root Node, state game.State) {
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for time.Since(start) < duration {
				simulate(root, state, cutoff)
			}
		}()
	}

	wg.Wait()
}

func (m *MCTS) findSubtree(path []Segment, state game.State) Node {
	if m.root == nil {
		return newDecision(nil, state)
	}
	if len(path) == 0 {
		return m.root
	}
	// Traverse the search tree by path
	// TODO: consider extracting to decision.go
	node := m.root.(*decision)
	for _, segment := range path {
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
	// Return the node at the end of the path as the new root
	node.parent = nil
	return node
}

func simulate(root Node, state game.State, cutoff int) {
	newNode, newState := selectThenExpand(root, state)
	player, score := rollout(newState, cutoff)
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

func rollout(state game.State, cutoff int) (string, float64) {
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
		return state.Winner(), Win
	}

	// At cutoff state, return an evaluation score from current player's perspective
	return state.Player(), state.Evaluate()
}

func backup(newNode Node, player string, score float64) {
	node := newNode
	for node != nil {
		parent := node.Backup(player, score)
		node = parent
	}
}

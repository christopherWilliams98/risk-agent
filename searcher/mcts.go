package searcher

import (
	"risk/game"
	"sync"
	"time"

	"golang.org/x/exp/rand"
)

// TODO: need to export?
type option func(mcts *MCTS)

type Segment struct {
	Move  game.Move
	State game.State
}

type MCTS struct {
	goroutines int
	episodes   int
	duration   time.Duration
	root       Node
}

func WithEpisodes(episodes int) option {
	return func(u *MCTS) {
		u.episodes = episodes
	}
}

func WithDuration(duration time.Duration) option {
	return func(u *MCTS) {
		u.duration = duration
	}
}

func NewMCTS(goroutines int, options ...option) *MCTS {
	u := &MCTS{goroutines: goroutines}
	for _, option := range options {
		option(u)
	}
	if u.episodes <= 0 && u.duration <= 0 {
		panic("Must specify search episodes or duration")
	}
	return u
}

func (m *MCTS) Simulate(state game.State, lineage []Segment) map[game.Move]int {
	// Reuse subtree if possible
	m.root = m.findSubtree(lineage, state)
	// Run simulations to collect statistics
	if m.episodes > 0 {
		iterate(m.goroutines, m.episodes, m.root, state)
	} else if m.duration > 0 {
		countdown(m.goroutines, m.duration, m.root, state)
	} else {
		panic("Must specify search episodes or duration")
	}
	// Return visit counts at the root node
	return m.root.Policy()
}

func iterate(goroutines int, episodes int, root Node, state game.State) {
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
				simulate(root, state)
			}
		}()
	}

	wg.Wait()
}

func countdown(goroutines int, duration time.Duration, root Node, state game.State) {
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for time.Since(start) < duration {
				simulate(root, state)
			}
		}()
	}

	wg.Wait()
}

func (m MCTS) findSubtree(path []Segment, state game.State) Node {
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

func simulate(root Node, state game.State) {
	newNode, newState := selectThenExpand(root, state)
	winner := rollout(newState)
	backup(newNode, winner)
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

func rollout(state game.State) string {
	// TODO: cutoff + manual evaluation function
	moves := state.LegalMoves()
	for len(moves) > 0 {
		// Follow a random rollout policy
		// TODO: 75% attack vs pass, 90% fortify vs pass
		move := moves[rand.Intn(len(moves))]
		state = state.Play(move)
		moves = state.LegalMoves()
	}
	return state.Winner()
}

func backup(newNode Node, winner string) {
	node := newNode
	for node != nil {
		parent := node.Backup(winner)
		node = parent
	}
}

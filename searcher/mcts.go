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
	root       *decision
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
	node := m.findSubtree(lineage)
	if node != nil {
		node.parent = nil
	} else {
		node = newDecision(nil, state)
	}
	m.root = node
	// Run simulations to collect statistics
	if m.episodes > 0 {
		iterate(m.goroutines, m.episodes, node, state)
	} else if m.duration > 0 {
		countdown(m.goroutines, m.duration, node, state)
	} else {
		panic("Must specify search episodes or duration")
	}
	// Return visit counts at the root node
	return m.root.Policy()
}

func iterate(goroutines int, episodes int, root Node, state game.State) {
	progress := make(chan any, episodes)
	done := make(chan any)

	driver := func() {
		for i := 0; i < episodes; i++ {
			<-progress
		}
		close(done)
	}
	driver()

	worker := func() {
		for {
			simulate(root, state)
			select {
			case progress <- nil:
			case <-done:
				return
			}
		}
	}
	for i := 0; i < goroutines; i++ {
		go worker()
	}
}

func countdown(goroutines int, duration time.Duration, root Node, state game.State) {
	start := time.Now()
	var wg sync.WaitGroup

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

func (m MCTS) findSubtree(path []Segment) *decision {
	if m.root == nil {
		return nil
	}
	// Traverse the search tree by path
	node := m.root
	for _, segment := range path {
		child := node.children[segment.Move]
		if child == nil {
			return nil
		}
		switch child := child.(type) {
		case *decision:
			node = child
		case *chance:
			grandChild := child.selects(segment.State)
			if grandChild == nil {
				return nil
			}
			node = grandChild
		default:
			panic("Unexpected node type")
		}
	}
	// Return the node at the end of the path
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
	for child != parent && selected {
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

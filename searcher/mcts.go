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
	iterations int
	duration   time.Duration
	root       *decision
}

func WithGoroutines(goroutines int) option {
	return func(u *MCTS) {
		u.goroutines = goroutines
	}
}

func WithIterations(iterations int) option {
	return func(u *MCTS) {
		u.iterations = iterations
	}
}

func WithDuration(duration time.Duration) option {
	return func(u *MCTS) {
		u.duration = duration
	}
}

func NewMCTS(options ...option) *MCTS {
	u := &MCTS{}
	for _, option := range options {
		option(u)
	}
	return u
}

func (m MCTS) findSubtree(path []Segment) *decision {
	if m.root == nil {
		return nil
	}
	// Traverse tree by path
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
			panic("Unexpected Node type")
		}
	}
	// Return Node at end of path
	return node
}

func (m *MCTS) RunSimulations(state game.State, lineage []Segment) map[game.Move]int {
	// Reuse subtree if possible
	node := m.findSubtree(lineage)
	if node == nil {
		node = newDecision(state, nil)
	} else {
		node.parent = nil
	}
	m.root = node

	// Run simulations to collect statistics
	// TODO: wait till parallel simulations complete
	if m.iterations > 0 {
		m.iterate(node, state)
	} else if m.duration > 0 {
		m.countdown(node, state)
	} else {
		panic("Must specify search iterations or duration")
	}

	// Return visit counts at root Node
	// TODO: extract to method on decision node
	visits := make(map[game.Move]int, len(m.root.children))
	for move, child := range m.root.children {
		visits[move] = child.Visits()
	}
	return visits
}

func (m *MCTS) iterate(root Node, state game.State) {
	progress := make(chan any, m.iterations)
	done := make(chan any)

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

	driver := func() {
		for i := 0; i < m.iterations; i++ {
			<-progress
		}
		close(done)
	}

	for i := 0; i < m.goroutines; i++ {
		go worker()
	}

	driver()
}

func (m *MCTS) countdown(root Node, state game.State) {
	start := time.Now()
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for time.Since(start) < m.duration {
			simulate(root, state)
		}
	}

	for i := 0; i < m.goroutines; i++ {
		wg.Add(1)
		go worker()
	}

	wg.Wait()
}

// TODO: startEpisode()
func simulate(root Node, state game.State) {
	// TODO: ensure each episode manipulates a diff state copy
	newNode, newState := doSelectionExpansion(root, state)
	winner := doRollout(newState)
	doBackup(newNode, winner)
}

func doSelectionExpansion(root Node, state game.State) (Node, game.State) {
	parent := root
	child, state, selected := parent.SelectOrExpand(state)
	for child != parent && selected {
		parent = child
		child, state, selected = parent.SelectOrExpand(state)
	}
	return child, state
}

func doRollout(state game.State) string {
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

func doBackup(newNode Node, winner string) {
	node := newNode
	for node != nil {
		parent := node.Backup(winner)
		node = parent
	}
}

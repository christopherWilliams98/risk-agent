package searcher

import (
	"risk/game"
	"sync"
	"time"

	"golang.org/x/exp/rand"
)

type option func(uct *uct)

type uct struct {
	goroutines int
	iterations int
	duration   time.Duration
	root       Node
}

func WithGoroutines(goroutines int) option {
	return func(u *uct) {
		u.goroutines = goroutines
	}
}

func WithIterations(iterations int) option {
	return func(u *uct) {
		u.iterations = iterations
	}
}

func WithDuration(duration time.Duration) option {
	return func(u *uct) {
		u.duration = duration
	}
}

func NewUCT(options ...option) *uct {
	u := &uct{}
	for _, option := range options {
		option(u)
	}
	return u
}

func (u *uct) FindNextMove(state game.State) game.Move {
	// TODO: find or create a node for state
	root := newDecision(nil, state) // TODO: set (new) root node parent to nil
	u.buildTree(root, state)
	u.root = root // TODO: save new root node for next search
	return root.findBestMove()
}

func (u *uct) buildTree(root Node, state game.State) {
	if u.iterations > 0 {
		u.iterate(root, state)
	} else if u.duration > 0 {
		u.countdown(root, state)
	} else {
		panic("Must specify search iterations or duration")
	}
}

func (u *uct) iterate(root Node, state game.State) {
	progress := make(chan any, u.iterations)
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
		for i := 0; i < u.iterations; i++ {
			<-progress
		}
		close(done)
	}

	for i := 0; i < u.goroutines; i++ {
		go worker()
	}

	driver()
}

func (u *uct) countdown(root Node, state game.State) {
	start := time.Now()
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for time.Since(start) < u.duration {
			simulate(root, state)
		}
	}

	for i := 0; i < u.goroutines; i++ {
		wg.Add(1)
		go worker()
	}

	wg.Wait()
}

func simulate(root Node, state game.State) {
	newNode, newState := doSelectionExpansion(root, state)
	winner := doRollout(newState)
	doBackup(newNode, winner)
}

func doSelectionExpansion(root Node, state game.State) (Node, game.State) {
	parent := root
	child, state, isAdded := parent.SelectOrExpand(state)
	for child != parent && !isAdded {
		parent = child
		child, state, isAdded = parent.SelectOrExpand(state)
	}
	return child, state
}

func doRollout(state game.State) string {
	// TODO: limit max depth vs rollout with cutoff vs value function
	moves := state.LegalMoves()
	for len(moves) > 0 {
		// Follow a random rollout policy
		// TODO: decision type: 75% attack, 90% fortify
		move := moves[rand.Intn(len(moves))]
		state = state.Play(move)
		moves = state.LegalMoves()
	}
	return state.Winner()
}

func doBackup(newNode Node, winner string) {
	node := newNode
	for node != nil {
		parent := node.Backup(rewarder(winner))
		node = parent
	}
}

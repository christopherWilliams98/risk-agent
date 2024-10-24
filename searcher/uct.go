package searcher

import (
	"risk/game"
	"time"

	"golang.org/x/exp/rand"
)

type uct struct {
	root  Node
	state game.State
}

func (u *uct) FindNextMove(state game.State) game.Move {
	// TODO: find or create a node for state
	root := newDecision(nil, state)

	return root.findBestMove()
}

func (u *uct) searchByIterations(goroutines int, iterations int) {
	progress := make(chan any, iterations)
	done := make(chan any)

	worker := func() {
		for {
			u.playout(u.root, u.state)
			select {
			case progress <- nil:
			case <-done:
				return
			}
		}
	}

	driver := func() {
		for i := 0; i < iterations; i++ {
			<-progress
		}
		close(done)
	}

	go driver()
	for i := 0; i < goroutines; i++ {
		go worker()
	}
}

func (u *uct) searchByDuration(goroutines int, duration time.Duration) {
	start := time.Now()
	worker := func() {
		for time.Since(start) < duration {
			u.playout(u.root, u.state)
		}
	}

	for i := 0; i < goroutines; i++ {
		go worker()
	}
}

func (u *uct) playout(root Node, state game.State) { // TODO: remove arguments??
	// Selection and expansion
	var parent Node = root
	child, nextState, isAdded := parent.SelectOrExpand(state)
	for child != parent && !isAdded {
		parent = child
		state = nextState
		child, nextState, isAdded = parent.SelectOrExpand(state)
	}

	// Simulation
	moves := nextState.GetMoves()
	for len(moves) > 0 {
		// Follow a random simulation policy
		move := moves[rand.Intn(len(moves))]
		state = nextState
		nextState = state.Play(move)
		moves = nextState.GetMoves()
	}
	// Last player to move wins the game
	winner := state.Player()

	// Backpropagation
	node := child
	for node != nil {
		parent := node.Update(rewarder(winner))
		node = parent
	}
}

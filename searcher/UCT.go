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
	// TODO: either iterate() or countdown() based on options argument

	return root.findBestMove()
}

func (u *uct) iterate(goroutines int, iterations int) {
	progress := make(chan any, iterations)
	done := make(chan any)

	worker := func() {
		for {
			simulate(u.root, u.state)
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

func (u *uct) countdown(goroutines int, duration time.Duration) {
	start := time.Now()

	worker := func() {
		for time.Since(start) < duration {
			simulate(u.root, u.state)
		}
	}

	for i := 0; i < goroutines; i++ {
		go worker()
	}
}

func simulate(root Node, state game.State) { // TODO: remove arguments??
	// Selection and expansion
	var parent Node = root
	child, nextState, isAdded := parent.PickChild(state)
	for child != parent && !isAdded {
		parent = child
		state = nextState
		child, nextState, isAdded = parent.PickChild(state)
	}

	// Rollout
	// TODO: limit max depth vs rollout with cutoff
	moves := nextState.LegalMoves()
	for len(moves) > 0 {
		// Follow a random rollout policy
		// TODO: 75% attack, 90% fortify
		move := moves[rand.Intn(len(moves))]
		state = nextState
		nextState = state.Play(move)
		moves = nextState.LegalMoves()
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

package gamemaster // TODO: rename to engine

import (
	"risk/game"
)

type UpdateGetter func() (game.Move, game.State)

type Engine interface {
	Init() (game.State, UpdateGetter)
	Play(game.Move) error
}

type update struct {
	move  game.Move
	state game.State
}

type localEngine struct {
	state    game.State
	updateCh chan update
}

// TODO: single instance for all players => use factory pattern?
func NewLocalEngine() *localEngine {
	return &localEngine{}
}

// TODO: remote engine should also satisfy this interface
func (e *localEngine) Init() (game.State, UpdateGetter) {
	// TODO: initialize the game state and board positions for 1 out of N total players
	// TODO: make sure a state value is returned, NOT a pointer (or return a copy)
	// e.state =

	e.updateCh = make(chan update, 1) // TODO: buffer size?
	// TODO: return a copy of the state or dereferenced value
	return e.state, func() (game.Move, game.State) {
		u, ok := <-e.updateCh
		if !ok { // Game over
			return nil, nil
		}
		// TODO: return a copy of the state or dereferenced value
		return u.move, u.state
	}
}

// TODO: remote engine should also satisfy this interface
func (e *localEngine) Play(move game.Move) error {
	// TODO: throw error if move is not legal or game is over, etc
	e.state = e.state.Play(move) // TODO: encompass move validation
	// TODO: propagate error

	e.updateCh <- update{move: move, state: e.state}
	return nil
}

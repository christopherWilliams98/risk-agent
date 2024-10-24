// Package game defines an agent that plays the game of Risk.
package game

type State interface {
	Player() string
	GetMoves() []Move
	Play(Move) State
	// State change from the last move
	// TODO: type should be easy to compare equality
	Delta() map[string]any
}

type Risk struct {
	// TODO: how to represent the board state? using a graph?
	board  [][]string
	player string // Player whose turn it is
	phase  string // Phase of the player's turn (reinforcement, attack or fortification)
	winner string // Winner if the game is over
}

func (r *Risk) Terminal() bool {
	return r.winner != ""
}

func (r *Risk) Player() string {
	return r.player
}

func (r *Risk) Winner() string {
	return r.winner
}

type Move interface {
	IsDeterministic() bool
}

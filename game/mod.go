// Package game defines an agent that plays the game of Risk.
package game

const (
	BONUS_TROOPS_REGION = 1
	//maybe could add number of players here? or in the rules. I'm not sure.
)

type Move interface {
	IsDeterministic() bool
}

type State interface {
	Player() string
	LegalMoves() []Move
	Play(Move) State
	// State change from the last move
	// TODO: type should be easy to compare equality
	Delta() map[string]any
}

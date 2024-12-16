package game

// TODO: move type definitions to corresponding files then delete this file

type Move interface {
	IsStochastic() bool
}

type StateHash uint64

// State should be immutable - operations on State always return a new copy
type State interface {
	Player() string
	LegalMoves() []Move
	Play(Move) State
	Hash() StateHash
	Winner() string
}

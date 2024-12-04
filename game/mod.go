package game

// TODO: move type definitions to corresponding files then delete this file

type Move interface {
	IsDeterministic() bool
}

type StateHash uint64

type State interface {
	Player() string
	LegalMoves() []Move
	Play(Move) State
	Hash() StateHash
	Winner() string
}

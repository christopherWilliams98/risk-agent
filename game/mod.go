package game

// TODO: move type definitions to corresponding files then delete this file

type Move interface {
	IsStochastic() bool
}

const (
	BONUS_TROOPS_REGION = 1
	//maybe could add number of players here? or in the rules. I'm not sure.
)

type StateHash uint64

// State should be immutable - operations on State always return a new copy
type State interface {
	Player() string
	LegalMoves() []Move
	Play(Move) State
	Hash() StateHash
	Winner() string
	Evaluate() float64
}

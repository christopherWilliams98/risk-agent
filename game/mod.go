package game

// TODO: interface should be defined in searcher package, any game that aims to be playable by an MCTS agent should implement this interface (i.e. searcher package is standalone, game package imports it, engine package imports both)

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
}

// Evaluates the game state to a score between -1 and 1 indicating how
// favorable the current player's position is to a winning (positive) outcome.
type Evaluate func(State) float64

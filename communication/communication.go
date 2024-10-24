package communication

import "risk/game"

// Communicator is an interface that abstracts the communication mechanism.
type Communicator interface {
	GetGameState() *game.GameState
	UpdateGameState(gs *game.GameState)
	SendAction(action Action)
	ReceiveAction() Action
}

// ActionType represents the type of action a player can perform.
type ActionType int

const (
	MoveAction ActionType = iota
	AttackAction
)

// Action represents an action taken by a player.
type Action struct {
	PlayerID     int
	Type         ActionType
	FromCantonID int
	ToCantonID   int
	NumTroops    int
}

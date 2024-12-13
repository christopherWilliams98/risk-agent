package communication

import "risk/game"

// Communicator is an interface that abstracts the communication mechanism.
type Communicator interface {
	GetGameState() *game.GameState
	UpdateGameState(gs *game.GameState)
	SendAction(action game.Action)
	ReceiveAction() game.Action
}

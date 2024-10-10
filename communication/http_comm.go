package communication

import (
	"risk/game"
	// TODO: imports, http stuff idk..
)

type HTTPCommunicator struct {
	//TODO: add fields for HTTP communication
}

// NewHTTPCommunicator initializes and returns a new HTTPCommunicator.
func NewHTTPCommunicator(initialState *game.GameState) *HTTPCommunicator {
	// TODO
	return &HTTPCommunicator{
		// TODO
	}
}

// GetGameState retrieves the current game state via HTTP.
func (hc *HTTPCommunicator) GetGameState() *game.GameState {
	// TODO
	return nil
}

// UpdateGameState updates the global game state via HTTP.
func (hc *HTTPCommunicator) UpdateGameState(gs *game.GameState) {
	// TODO
}

// SendAction sends an action to be processed via HTTP.
func (hc *HTTPCommunicator) SendAction(action Action) {
	// TODO
}

// ReceiveAction receives an action from players via HTTP.
func (hc *HTTPCommunicator) ReceiveAction() Action {
	// TODO
	return Action{}
}

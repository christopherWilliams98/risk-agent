package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"risk/game"
)

type ClientCommunicator struct {
	serverURL string
}

// NewClientCommunicator initializes and returns a new ClientCommunicator.
func NewClientCommunicator(serverURL string) *ClientCommunicator {
	return &ClientCommunicator{
		serverURL: serverURL,
	}
}

func (cc *ClientCommunicator) GetGameState() *game.GameState {
	resp, err := http.Get(cc.serverURL + "/getGameState")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var gs game.GameState
	json.NewDecoder(resp.Body).Decode(&gs)
	return &gs
}

func (cc *ClientCommunicator) UpdateGameState(gs *game.GameState) {
	data, _ := json.Marshal(gs)
	http.Post(cc.serverURL+"/updateGameState", "application/json", bytes.NewBuffer(data))
}

func (cc *ClientCommunicator) SendAction(action game.Action) {
	data, _ := json.Marshal(action)
	http.Post(cc.serverURL+"/sendAction", "application/json", bytes.NewBuffer(data))
}

func (cc *ClientCommunicator) ReceiveAction() game.Action {
	resp, err := http.Get(cc.serverURL + "/receiveAction")
	if err != nil || resp.StatusCode != http.StatusOK {
		return game.Action{}
	}
	defer resp.Body.Close()
	var action game.Action
	json.NewDecoder(resp.Body).Decode(&action)
	return action
}

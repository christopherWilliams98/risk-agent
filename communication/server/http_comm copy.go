package server

import (
	"encoding/json"
	"net/http"
	"risk/communication"
	"risk/game"
	"sync"
)

type ServerCommunicator struct {
	gameState *game.GameState
	actions   chan communication.Action
	mutex     sync.RWMutex
}

// NewServerCommunicator initializes and returns a new ServerCommunicator.
func NewServerCommunicator(initialState *game.GameState) *ServerCommunicator {
	sc := &ServerCommunicator{
		gameState: initialState,
		actions:   make(chan communication.Action, 100),
	}
	go sc.startServer()
	return sc
}

func (sc *ServerCommunicator) startServer() {
	http.HandleFunc("/getGameState", sc.handleGetGameState)
	http.HandleFunc("/updateGameState", sc.handleUpdateGameState)
	http.HandleFunc("/sendAction", sc.handleSendAction)
	http.HandleFunc("/receiveAction", sc.handleReceiveAction)
	http.ListenAndServe(":8080", nil)
}

func (sc *ServerCommunicator) handleGetGameState(w http.ResponseWriter, r *http.Request) {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	json.NewEncoder(w).Encode(sc.gameState)
}

func (sc *ServerCommunicator) handleUpdateGameState(w http.ResponseWriter, r *http.Request) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	var gs game.GameState
	json.NewDecoder(r.Body).Decode(&gs)
	sc.gameState = &gs
	w.WriteHeader(http.StatusOK)
}

func (sc *ServerCommunicator) handleSendAction(w http.ResponseWriter, r *http.Request) {
	var action communication.Action
	json.NewDecoder(r.Body).Decode(&action)
	sc.actions <- action
	w.WriteHeader(http.StatusOK)
}

func (sc *ServerCommunicator) handleReceiveAction(w http.ResponseWriter, r *http.Request) {
	select {
	case action := <-sc.actions:
		json.NewEncoder(w).Encode(action)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func (sc *ServerCommunicator) GetGameState() *game.GameState {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	return sc.gameState.Copy()
}

func (sc *ServerCommunicator) UpdateGameState(gs *game.GameState) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.gameState = gs
}

func (sc *ServerCommunicator) SendAction(action communication.Action) {
	sc.actions <- action
}

func (sc *ServerCommunicator) ReceiveAction() communication.Action {
	return <-sc.actions
}

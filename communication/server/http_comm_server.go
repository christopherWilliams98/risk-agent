package server

import (
	"encoding/json"
	"net/http"
	"risk/game"
	"sync"
)

type ServerCommunicator struct {
	gameState *game.GameState
	actions   chan game.Action
	mutex     sync.RWMutex
}

// NewServerCommunicator initializes and returns a new ServerCommunicator.
func NewServerCommunicator() *ServerCommunicator {
	sc := &ServerCommunicator{
		gameState: nil, // Initialize to nil asGameMaster will set it
		actions:   make(chan game.Action, 100),
	}
	return sc
}

// StartServer starts the HTTP server.
func (sc *ServerCommunicator) Start() {
	http.HandleFunc("/getGameState", sc.handleGetGameState)
	http.HandleFunc("/updateGameState", sc.handleUpdateGameState)
	http.HandleFunc("/sendAction", sc.handleSendAction)
	http.HandleFunc("/receiveAction", sc.handleReceiveAction)
	http.ListenAndServe(":8080", nil)
}

func (sc *ServerCommunicator) handleGetGameState(w http.ResponseWriter, r *http.Request) {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	if sc.gameState == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
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
	var action game.Action
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
	if sc.gameState == nil {
		return nil
	}
	gsCopy := sc.gameState.Copy()
	return &gsCopy
}

func (sc *ServerCommunicator) UpdateGameState(gs *game.GameState) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.gameState = gs
}

func (sc *ServerCommunicator) SendAction(action game.Action) {
	sc.actions <- action
}

func (sc *ServerCommunicator) ReceiveAction() game.Action {
	return <-sc.actions
}

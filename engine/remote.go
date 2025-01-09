package gamemaster

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"risk/game"
	"risk/meta"
)

type EngineHTTP struct {
	State *game.GameState

	AgentURLs []string
}

type Update struct {
	Move  game.GameMove  `json:"move"`
	State game.State     `json:"state"`
	Hash  game.StateHash `json:"hash"`
}

func LocalEngineHTTP(players []string, urls []string, m *game.Map, r game.Rules) *EngineHTTP {
	if len(players) != len(urls) {
		panic("number of players does not match number of agent URLs")
	}
	if len(players) < 2 {
		panic("need at least two players")
	}

	state := game.NewGameState(m, r)
	state.CurrentPlayer = 1

	return &EngineHTTP{
		State:     state,
		AgentURLs: urls,
	}
}

func (e *EngineHTTP) Run() {
	turnCount := 0

	updates := make([][]Update, len(e.AgentURLs))
	for i := range updates {
		updates[i] = []Update{}
	}

	for e.State.Winner() == "" && turnCount < meta.MAX_TURNS {
		currentPlayerID := e.State.CurrentPlayer
		agentIndex := currentPlayerID - 1

		fmt.Printf("===== TURN BEGIN: Player %d (agent index %d) =====\n", currentPlayerID, agentIndex)

		move := e.requestMoveFromAgent(agentIndex, updates[agentIndex])

		fmt.Printf("[EngineHTTP.Run] Player %d chose move: %+v\n", currentPlayerID, move)

		newState := e.State.Play(move).(*game.GameState)
		u := Update{
			Move:  *move,
			State: newState.Copy(),
			Hash:  newState.Hash(),
		}
		updates[agentIndex] = append(updates[agentIndex], u)

		e.State = newState

		// push to python visualizer web app:
		jsonPayload, _ := json.Marshal(e.State)
		http.Post("http://localhost:5000/visualhook", "application/json", bytes.NewReader(jsonPayload))

		turnCount++
	}

	if e.State.Winner() != "" {
		fmt.Printf("Game ended due to a winner: %s\n", e.State.Winner())
	} else {
		fmt.Printf("Stopped after %d turns (no winner yet)\n", meta.MAX_TURNS)
	}
}

// requestMoveFromAgent encodes the current state + updates in JSON, posts to /findmove on agent side
func (e *EngineHTTP) requestMoveFromAgent(agentIndex int, upd []Update) *game.GameMove {
	payload := struct {
		State   *game.GameState `json:"state"`
		Updates []Update        `json:"updates"`
	}{
		State:   e.State,
		Updates: upd,
	}

	// Marshal to JSON
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	url := e.AgentURLs[agentIndex] + "/findmove"
	resp, err := http.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out, _ := io.ReadAll(resp.Body)
		panic(fmt.Sprintf("Agent returned status %d: %s", resp.StatusCode, out))
	}

	var gm game.GameMove
	if err := json.NewDecoder(resp.Body).Decode(&gm); err != nil {
		panic(err)
	}
	// Possibly validate the move:
	if !game.IsMoveValidForPhase(e.State.Phase, &gm) {
		fmt.Printf("[EngineHTTP] Agent returned invalid move => forcing pass.\n")
		fallback := e.State.LegalMoves()
		if len(fallback) == 0 {
			panic("No legal moves at all")
		}
		return fallback[0].(*game.GameMove)
	}

	return &gm
}

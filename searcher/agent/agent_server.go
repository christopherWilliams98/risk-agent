package agent

import (
	"encoding/json"
	"log"
	"net/http"

	"risk/game"
	"risk/searcher"
)

var evalAgent Agent

// StartAgentServer starts an agent HTTP server on the given port.
func StartAgentServer(port string) {
	log.Printf("[AgentServer] Starting agent server on :%s ...", port)

	myMCTS := searcher.NewMCTS(
		8,
		searcher.WithEpisodes(50),
		searcher.WithCutoff(50),
	)

	evalAgent = NewEvaluationAgent(myMCTS)

	// Create a local mux rather than using the global DefaultServeMux
	mux := http.NewServeMux()
	mux.HandleFunc("/findmove", handleFindMove)

	// Then pass `mux` to ListenAndServe
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func handleFindMove(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		State   game.GameState     `json:"state"`
		Updates []searcher.Segment `json:"updates"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if payload.State.Rules == nil {
		payload.State.Rules = game.NewStandardRules()
	}

	chosenMove, _ := evalAgent.FindMove(&payload.State, payload.Updates...)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chosenMove); err != nil {
		http.Error(w, "failed to encode move: "+err.Error(), http.StatusInternalServerError)
	}
}

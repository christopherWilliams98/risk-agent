package main

import (
	"fmt"
	"log"
	gamemaster "risk/engine"
	"risk/game"
	"risk/searcher/agent"
	"time"
)

// run two agent servers on different ports (8080 & 8081).
func main() {
	// 1) Start the first agent server in a goroutine
	go agent.StartAgentServer("8080")

	// 2) Start the second agent server in another goroutine
	go agent.StartAgentServer("8081")

	// Sleep a tiny bit to ensure servers have started.
	time.Sleep(1 * time.Second)
	log.Println("[main] Both agent servers should be up now.")

	// 3) Now create the map, rules, and set up EngineHTTP with agent URLs
	gameMap := game.CreateMap()
	rules := game.NewStandardRules() // TODO I want more rules

	agentURLs := []string{
		"http://localhost:8080",
		"http://localhost:8081",
	}

	engine := gamemaster.LocalEngineHTTP(
		[]string{"Player1", "Player2"},
		agentURLs,
		gameMap,
		rules,
	)

	// 4) Run the game
	engine.Run()

	winner := engine.State.Winner()
	fmt.Printf("Game over! Winner: %s\n", winner)
}

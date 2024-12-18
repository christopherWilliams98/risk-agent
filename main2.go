package main

import (
	"fmt"
	"risk/engine"
	"risk/game"
)

func main() {

	// Create the map
	gameMap := game.CreateMap()

	// Create standard rules
	rules := game.NewStandardRules()

	// Create the agents
	agents := []gamemaster.Agent{
		gamemaster.NewMCTSAgent(1),
		gamemaster.NewMCTSAgent(2),
	}

	// Instantiate engine
	players := []string{"Player1", "Player2"}
	engine := gamemaster.LocalEngine(players, agents, gameMap, rules)

	// Run the game
	engine.Run()

	winner := engine.State.Winner()
	fmt.Printf("Game over! Winner: %s\n", winner)
}

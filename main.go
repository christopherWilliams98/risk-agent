package main

import (
	"fmt"
	gamemaster "risk/engine"
	"risk/game"
	"risk/searcher"
	"risk/searcher/agent"
)

func main() {

	// Create the map
	gameMap := game.CreateMap()

	// Create standard rules
	rules := game.NewStandardRules()

	// Construct MCTS objects
	myMCTS := searcher.NewMCTS(
		1,
		searcher.WithEpisodes(50),
		searcher.WithCutoff(50),
	)

	// agent1 := agent.NewTrainingAgent(myMCTS)
	// agent2 := agent.NewTrainingAgent(myMCTS)

	agent1 := agent.NewEvaluationAgent(myMCTS)
	agent2 := agent.NewEvaluationAgent(myMCTS)

	// Wrap them into gamemaster.Agent via MCTSAdapter
	adapter1 := &gamemaster.MCTSAdapter{InternalAgent: agent1}
	adapter2 := &gamemaster.MCTSAdapter{InternalAgent: agent2}

	players := []string{"Player1", "Player2"}
	agents := []gamemaster.MCTSAdapter{*adapter1, *adapter2}

	// Instantiate engine
	engine := gamemaster.LocalEngine(players, agents, gameMap, rules)

	// Run the game
	engine.Run()

	winner := engine.State.Winner()
	fmt.Printf("Game over! Winner: %s\n", winner)
}

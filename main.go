package main

import (
	"risk/communication"
	"risk/game"
	"risk/gamemaster"
	"risk/player"
	"sync"
)

func main() {
	// Initialize the game map and global game state
	gameMap := game.CreateMap()
	globalGameState := game.NewGameState(gameMap)

	// Initialize the HTTP communicator
	comm := communication.NewHTTPCommunicator(globalGameState)

	// Initialize the GameMaster
	gameMaster := gamemaster.NewGameMaster(comm)

	// Initialize Players
	player1 := player.NewPlayer(1, comm)
	player2 := player.NewPlayer(2, comm)

	// Initialize the game state
	gameMaster.InitializeGame()

	// wait for goroutines
	var wg sync.WaitGroup
	wg.Add(3)

	//  Start GameMaster in a goroutine
	go func() {
		defer wg.Done()
		gameMaster.RunGame()
	}()

	// Start players in goroutines
	go func() {
		defer wg.Done()
		player1.Play()
	}()

	go func() {
		defer wg.Done()
		player2.Play()
	}()

	// Wait for  goroutines to finish
	wg.Wait()
}

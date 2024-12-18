package main

import (
	"risk/communication/client"
	"risk/communication/server"
	"risk/engine"
	"risk/player"
	"sync"
)

func main2() {
	// Initialize the ServerCommunicator
	serverComm := server.NewServerCommunicator()
	go serverComm.Start()

	// Initialize the GameMaster's ClientCommunicator
	gmComm := client.NewClientCommunicator("http://localhost:8080")
	gameMaster := gamemaster.NewGameMaster(gmComm)

	// Initialize Players with ClientCommunicator
	player1Comm := client.NewClientCommunicator("http://localhost:8080")
	player2Comm := client.NewClientCommunicator("http://localhost:8080")

	player1 := player.NewPlayer(1, player1Comm)
	player2 := player.NewPlayer(2, player2Comm)

	// Initialize the game state
	gameMaster.InitializeGame()

	// WaitGroup for synchronization
	var wg sync.WaitGroup
	wg.Add(3)

	// Start GameMaster in a goroutine
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

	// Wait for goroutines to finish
	wg.Wait()
}

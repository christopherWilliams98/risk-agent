package gamemaster

import (
	"fmt"
	"risk/communication"
	"risk/game"
)

// GameMaster manages the game flow and resolves actions.
type GameMaster struct {
	Communicator communication.Communicator
}

// NewGameMaster initializes a new GameMaster.
func NewGameMaster(comm communication.Communicator) *GameMaster {
	return &GameMaster{
		Communicator: comm,
	}
}

// InitializeGame sets up the initial game state.
func (gm *GameMaster) InitializeGame() {
	// Create the initial game state
	gameMap := game.CreateMap()
	rules := game.NewStandardRules()
	gs := game.NewGameState(gameMap, rules)

	for id := 0; id < len(gs.Map.Cantons); id++ {
		gs.Ownership[id] = (id % 2) + 1 // Alternate ownership between Player 1 and 2
		gs.TroopCounts[id] = 3
	}
	gm.Communicator.UpdateGameState(gs)
}

// the game loop.
func (gm *GameMaster) RunGame() {
	gameOver := false
	for !gameOver {
		// Get the latest game state
		gs := gm.Communicator.GetGameState()

		// Check if it's the end of the game
		if gm.CheckGameOver(gs) {
			fmt.Printf("Player %d wins!\n", gs.CurrentPlayer)
			gameOver = true
			continue
		}

		// Receive action from the current player
		action := gm.Communicator.ReceiveAction()

		// Validate that the action is from the correct player
		if action.PlayerID != gs.CurrentPlayer {
			// Handle invalid action
			continue
		}

		// Apply the action using gs.Play()
		newGs := gs.Play(&game.GameMove{
			ActionType:   action.Type,
			FromCantonID: action.FromCantonID,
			ToCantonID:   action.ToCantonID,
			NumTroops:    action.NumTroops,
		}).(*game.GameState)

		// Update the global game state
		gm.Communicator.UpdateGameState(newGs)
	}
}

// CheckGameOver determines if the game has ended.
func (gm *GameMaster) CheckGameOver(gs *game.GameState) bool {
	// Check if any player has no territories
	playerTerritories := make(map[int]int)
	for _, owner := range gs.Ownership {
		if owner != -1 {
			playerTerritories[owner]++
		}
	}

	// If any player has no territories, game is over
	for playerID, territories := range playerTerritories {
		if territories == 0 {
			fmt.Printf("Player %d has been eliminated!\n", playerID)
		}
	}

	// Check if only one player remains
	if len(playerTerritories) == 1 {
		return true
	}
	return false
}

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

	// Assign territories to players
	totalTerritories := len(gs.Map.Cantons)
	//territoriesPerPlayer := totalTerritories / 2 // 13 territories per player

	// Initialize ownership and troop counts
	for id := 0; id < totalTerritories; id++ {
		playerID := (id % 2) + 1
		gs.Ownership[id] = playerID
		gs.TroopCounts[id] = 1 // Start with 1 troop per territory
	}

	// Players have remaining troops to place
	gs.PlayerTroops = map[int]int{
		1: 27, // 40 - 13
		2: 27,
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

		// Depending on the phase, handle actions
		switch gs.Phase {
		case game.InitialPlacementPhase:
			// Players take turns placing their initial troops
			gm.handleInitialPlacement(gs)
		default:
			// Regular gameplay
			action := gm.Communicator.ReceiveAction()

			// Validate that the action is from the correct player
			if action.PlayerID != gs.CurrentPlayer {
				// Handle invalid action
				// Security: how do we make sure that it really is the right player
				// websocket Look into it later! (low priority)
				continue
			}

			// Apply the action
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
}

func (gm *GameMaster) handleInitialPlacement(gs *game.GameState) {
	// Receive action from the current player
	action := gm.Communicator.ReceiveAction()

	// Validate action
	if action.Type != game.ReinforceAction || action.PlayerID != gs.CurrentPlayer {
		// Handle invalid action
		return
	}

	// Apply the action
	gs.TroopCounts[action.ToCantonID] += action.NumTroops
	gs.PlayerTroops[gs.CurrentPlayer] -= action.NumTroops

	// Check if the player has finished placing troops
	if gs.PlayerTroops[gs.CurrentPlayer] == 0 {
		gs.AdvancePhase()
	} else {
		// Switch to the next player
		gs.CurrentPlayer = gs.NextPlayer()
	}

	gm.Communicator.UpdateGameState(gs)
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
	return len(playerTerritories) == 1
}

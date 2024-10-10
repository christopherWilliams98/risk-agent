package gamemaster

import (
	"fmt"
	"risk/communication"
	"risk/game"
)

// managing the game flow and resolving actions.
type GameMaster struct {
	Communicator communication.Communicator
}

// NewGameMaster
func NewGameMaster(comm communication.Communicator) *GameMaster {
	return &GameMaster{
		Communicator: comm,
	}
}

// InitializeGame sets up the initial game state.
func (gm *GameMaster) InitializeGame() {
	gs := gm.Communicator.GetGameState()
	for id := 0; id < len(gs.Map.Cantons); id++ {
		gs.Ownership[id] = (id % 2) + 1 //TODO this, right now it just adds 3 troops to each canton with alternating ids for ownership. this should be handled by the player. (like a setup action)
		gs.TroopCounts[id] = 3
	}
	gm.Communicator.UpdateGameState(gs)
}

// the game loop.
func (gm *GameMaster) RunGame() {
	gameOver := false
	for !gameOver {
		// Receive action from a player
		action := gm.Communicator.ReceiveAction()

		// Get the latest game state from the game master
		gs := gm.Communicator.GetGameState()

		// Resolve the action
		switch action.Type {
		case communication.MoveAction:
			err := gs.MoveTroops(action.FromCantonID, action.ToCantonID, action.NumTroops)
			if err != nil {
				fmt.Printf("GameMaster: Move error: %v\n", err)
			}
		case communication.AttackAction:
			err := gs.Attack(action.FromCantonID, action.ToCantonID, action.NumTroops)
			if err != nil {
				fmt.Printf("GameMaster: Attack error: %v\n", err)
			}

			// TODO - add more action types like specific decision in seting up the game, assigning troops when you gain them, etc etc..

		}

		// Update the global game state
		gm.Communicator.UpdateGameState(gs)

		// Check for game over - TODO - rule constraints? - I am unsure how and where to implement the entire constraint thing, need to brainstorm and discuss.
		if gm.CheckGameOver(gs) {
			fmt.Printf("Player %d wins!\n", action.PlayerID)
			gameOver = true
		}
	}
}

// CheckGameOver determines if the game has ended.
func (gm *GameMaster) CheckGameOver(gs *game.GameState) bool {
	firstOwner := gs.Ownership[0]
	for _, owner := range gs.Ownership {
		if owner != firstOwner {
			return false
		}
	}
	return true
}

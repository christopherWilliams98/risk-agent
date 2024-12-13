package player

import (
	"fmt"
	"math/rand"
	"risk/communication"
	"risk/game"
)

// Player represents a game player.
type Player struct {
	ID             int
	Communicator   communication.Communicator
	LocalGameState *game.GameState
}

// NewPlayer creates a new Player instance.
func NewPlayer(id int, comm communication.Communicator) *Player {
	return &Player{
		ID:           id,
		Communicator: comm,
	}
}

// Play starts the player's turn loop.
func (p *Player) Play() {
	for {

		p.SyncGameState()

		action := p.TakeTurn()

		if action.Type == -1 {
			fmt.Printf("Player %d has no possible actions. Ending turn.\n", p.ID)
			return
		}

		action.PlayerID = p.ID
		p.Communicator.SendAction(action)
	}
}

// update the player's local game state.
func (p *Player) SyncGameState() {
	p.LocalGameState = p.Communicator.GetGameState()
}

// decides on an action to perform.
func (p *Player) TakeTurn() communication.Action {
	possibleActions := p.generatePossibleActions()
	if len(possibleActions) == 0 {
		return communication.Action{Type: -1} // No possible actions
	}

	// Placeholder TODO right now its just random.
	chosenAction := possibleActions[rand.Intn(len(possibleActions))]
	return chosenAction
}

// TODO implement the changes with rules.
// generatePossibleActions generates all possible actions for the player.
func (p *Player) generatePossibleActions() []communication.Action {
	var actions []communication.Action
	gs := p.LocalGameState

	for cantonID, owner := range gs.Ownership {
		if owner == p.ID {
			// Move actions
			for _, adjID := range gs.Map.Cantons[cantonID].AdjacentIDs {
				if gs.Ownership[adjID] == p.ID && gs.TroopCounts[cantonID] > 1 {
					actions = append(actions, communication.Action{
						Type:         communication.MoveAction,
						FromCantonID: cantonID,
						ToCantonID:   adjID,
						NumTroops:    gs.TroopCounts[cantonID] - 1,
					})
				}
			}
			// Attack actions
			for _, adjID := range gs.Map.Cantons[cantonID].AdjacentIDs {
				if gs.Ownership[adjID] != p.ID && gs.TroopCounts[cantonID] > 1 {
					actions = append(actions, communication.Action{
						Type:         communication.AttackAction,
						FromCantonID: cantonID,
						ToCantonID:   adjID,
						NumTroops:    gs.TroopCounts[cantonID] - 1,
					})
				}
			}
		}
	}
	return actions
}

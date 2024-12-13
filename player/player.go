package player

import (
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

func (p *Player) SyncGameState() {
	p.LocalGameState = p.Communicator.GetGameState()
}

// Play starts the player's turn loop.
func (p *Player) Play() {
	for {
		p.SyncGameState()

		// action := p.TakeTurn()

		// if action.Type == -1 {
		// 	fmt.Printf("Player %d has no possible actions. Ending turn.\n", p.ID)
		// 	return
		// }

		// action.PlayerID = p.ID
		// p.Communicator.SendAction(action)
	}
}

// update the player's local game state.
// func (p *Player) TakeTurn() game.Action {
// 	p.SyncGameState()
// 	gs := p.LocalGameState

// 	// Use MCTS to find the best move
// 	// mcts := &searcher.UCT{}
// 	// bestMove := mcts.FindNextMove(gs)

// 	if bestMove == nil {
// 		possibleMoves := gs.LegalMoves()
// 		if len(possibleMoves) == 0 {
// 			return game.Action{Type: PassAction}
// 		}
// 		bestMove = possibleMoves[rand.Intn(len(possibleMoves))]
// 	}

// 	// Convert GameMove to communication.Action
// 	gameMove := bestMove.(*game.GameMove)
// 	action := game.Action{
// 		PlayerID:     p.ID,
// 		Type:         gameMove.ActionType,
// 		FromCantonID: gameMove.FromCantonID,
// 		ToCantonID:   gameMove.ToCantonID,
// 		NumTroops:    gameMove.NumTroops,
// 	}
// 	return action
// }

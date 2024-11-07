package game

import "risk/communication"

// GameMove represents a move in the game.
type GameMove struct {
	ActionType   communication.ActionType
	FromCantonID int
	ToCantonID   int
	NumTroops    int
}

func (gm *GameMove) IsDeterministic() bool {

	return gm.ActionType != communication.AttackAction
}

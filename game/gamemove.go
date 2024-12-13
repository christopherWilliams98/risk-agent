package game

// GameMove represents a move in the game.
type GameMove struct {
	ActionType   ActionType
	FromCantonID int
	ToCantonID   int
	NumTroops    int
}

func (gm *GameMove) IsDeterministic() bool {

	return gm.ActionType != AttackAction
}

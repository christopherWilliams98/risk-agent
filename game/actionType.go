package game

// ActionType represents the type of action a player can perform.
type ActionType int

const (
	MoveAction ActionType = iota
	AttackAction
	ReinforceAction
	ManeuverAction
	PassAction
)

// Action represents an action taken by a player.
type Action struct {
	PlayerID     int
	Type         ActionType
	FromCantonID int
	ToCantonID   int
	NumTroops    int
}

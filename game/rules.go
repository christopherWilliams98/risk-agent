package game

type Rules interface {
	MaxAttackTroops() int
	MaxDefendTroops() int
	DetermineAttackOutcome(attackerRolls, defenderRolls []int) (attackerLosses, defenderLosses int)
	IsAttackSuccessful(attackerRolls, defenderRolls []int) bool
	// TODO add more ruels
}

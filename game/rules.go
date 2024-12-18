package game

type Rules interface {
	MaxAttackTroops() int
	MaxDefendTroops() int
	DetermineAttackOutcome(attackerRolls, defenderRolls []int) (attackerLosses, defenderLosses int)
	GetRegionBonus(regionID int) int
	// TODO add more ruels
}

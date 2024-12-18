package game

type StandardRules struct {
	MaxAttackDice int
	MaxDefendDice int
}

func NewStandardRules() *StandardRules {
	return &StandardRules{
		MaxAttackDice: 3,
		MaxDefendDice: 2,
	}
}

func (sr *StandardRules) MaxAttackTroops() int {
	return sr.MaxAttackDice
}

func (sr *StandardRules) DetermineAttackOutcome(attackerRolls, defenderRolls []int) (attackerLosses, defenderLosses int) {
	// Standard Risk attack outcome
	battles := min(len(attackerRolls), len(defenderRolls))
	for i := 0; i < battles; i++ {
		if attackerRolls[i] > defenderRolls[i] {
			defenderLosses++
		} else {
			attackerLosses++
		}
	}
	return
}

func (sr *StandardRules) MaxDefendTroops() int {
	return sr.MaxDefendDice
}

func (sr *StandardRules) IsAttackSuccessful(attackerRolls, defenderRolls []int) bool {
	// Todo

	return false
}

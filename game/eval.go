package game

// evaluateBasic simply tallies each player's controlled resources (territories, troops, and regions) to produce a relative score between -1 and 1 from the current player's perspective
func (gs *GameState) evaluateBasic() float64 {
	territoryScore, troopScore := gs.calculateResourceScores()
	bonusScore := gs.calculateBonusScore()

	return (territoryScore + troopScore + bonusScore) / 3.0
}

// evaluateWithBorders considers each player's border strength and connectedness, in addition to controlled resources, to produce a score between -1 and 1 from the current player's perspective
func (gs *GameState) evaluateWithBorders() float64 {
	territoryScore, troopScore := gs.calculateResourceScores()
	bonusScore := gs.calculateBonusScore()
	borderScore := gs.calculateBorderScore()

	return (territoryScore + troopScore + bonusScore + borderScore) / 4
}

func (gs *GameState) calculateResourceScores() (territoryScore, troopScore float64) {
	territories := make(map[int]float64)
	troops := make(map[int]float64)

	// Tally territories and troops by player
	for cantonID, owner := range gs.Ownership {
		if owner > 0 { // Skip unowned
			territories[owner]++
			troops[owner] += float64(gs.TroopCounts[cantonID])
		}
	}

	current := gs.CurrentPlayer
	opponent := gs.NextPlayer()
	territoryScore = normalize(territories[current], territories[opponent])
	troopScore = normalize(troops[current], troops[opponent])
	return territoryScore, troopScore
}

func (gs *GameState) calculateBonusScore() float64 {
	regionBonus := make(map[int]float64)

	// Tally fully controlled regions weighted by bonus values
	for _, region := range gs.Map.Regions {
		owner := getRegionOwner(*region, gs.Ownership)
		if owner > 0 {
			regionBonus[owner] += float64(region.Bonus)
		}
	}

	return normalize(regionBonus[gs.CurrentPlayer], regionBonus[gs.NextPlayer()])
}

func (gs *GameState) calculateBorderScore() float64 {
	current := gs.CurrentPlayer
	opponent := gs.NextPlayer()
	borderStrength := make(map[int]float64)

	// Calculate border strength for each player
	for canton, owner := range gs.Ownership {
		if owner != current && owner != opponent {
			continue
		}

		myTroops := float64(gs.TroopCounts[canton])
		// Tally all enemy neighbors factors in connectedness: higher connections, higher strategic value
		// (e.g. more opportunities to attack, potentially a chokepoint, etc)
		for _, neighbor := range gs.Map.Cantons[canton].AdjacentIDs {
			if gs.Ownership[neighbor] != owner {
				enemyTroops := float64(gs.TroopCounts[neighbor])
				// Troop difference mimics line of attack
				// Not capped because an arbitray number of attacks can be launched till only 1 troop left
				borderStrength[owner] += (myTroops - 1) - enemyTroops
			}
		}
	}

	return normalize(borderStrength[current], borderStrength[opponent])
}

func getRegionOwner(region Region, ownerByCanton []int) int {
	if len(region.CantonIDs) == 0 {
		return -1
	}
	owner := ownerByCanton[region.CantonIDs[0]]
	for _, cantonID := range region.CantonIDs[1:] {
		if ownerByCanton[cantonID] != owner {
			return -1
		}
	}
	return owner
}

// normalize normalizes value relative to otherValue to a score between -1 and 1
func normalize(value float64, otherValue float64) float64 {
	total := value + otherValue
	if total == 0 {
		return 0
	}
	return (value - otherValue) / total
}

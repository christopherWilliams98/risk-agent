package game

import "math"

// evaluateBasic simply tallies each player's controlled resources (territories, troops, and regions) to produce a relative score between -1 and 1 from the current player's perspective
func EvaluateResources(s State) float64 {
	gs, ok := s.(*GameState)
	if !ok {
		panic("unexpected state type")
	}
	territoryScore, troopScore := gs.calculateResourceScores()
	bonusScore := gs.calculateBonusScore()

	return (territoryScore + troopScore + bonusScore) / 3.0
}

// evaluateBorderStrength considers each player's border strength, in addition to controlled resources, to produce a score between -1 and 1 from the current player's perspective
func EvaluateBorderStrength(s State) float64 {
	gs, ok := s.(*GameState)
	if !ok {
		panic("unexpected state type")
	}
	territoryScore, troopScore := gs.calculateResourceScores()
	bonusScore := gs.calculateBonusScore()
	borderScore := gs.calculateBorderScore()

	return (territoryScore + troopScore + bonusScore + borderScore) / 4
}

// evaluateConnectivity considers each player's connectedness, in addition to controlled resources, to produce a score between -1 and 1 from the current player's perspective
func EvaluateConnectivity(s State) float64 {
	gs, ok := s.(*GameState)
	if !ok {
		panic("unexpected state type")
	}
	territoryScore, troopScore := gs.calculateResourceScores()
	bonusScore := gs.calculateBonusScore()
	connectivityScore := gs.calculateConnectivityScore()

	return (territoryScore + troopScore + bonusScore + connectivityScore) / 4
}

func EvaluateBorderConnectivity(s State) float64 {
	gs, ok := s.(*GameState)
	if !ok {
		panic("unexpected state type")
	}
	territoryScore, troopScore := gs.calculateResourceScores()
	bonusScore := gs.calculateBonusScore()
	borderScore := gs.calculateBorderScore()
	connectivityScore := gs.calculateConnectivityScore()

	return (territoryScore + troopScore + bonusScore + borderScore + connectivityScore) / 5
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

func (gs *GameState) calculateConnectivityScore() float64 {
	current := gs.CurrentPlayer
	opponent := gs.NextPlayer()
	connectivity := make(map[int]float64)

	// Build graph of controlled territories
	for player := range []int{current, opponent} {
		graph := make(map[int][]int)
		for canton, owner := range gs.Ownership {
			if owner == player {
				neighbors := []int{}
				for _, n := range gs.Map.Cantons[canton].AdjacentIDs {
					if gs.Ownership[n] == player {
						neighbors = append(neighbors, n)
					}
				}
				graph[canton] = neighbors
			}
		}

		// Calculate largest connected component
		visited := make(map[int]bool)
		maxComponent := 0

		for canton := range graph {
			if !visited[canton] {
				size := gs.dfs(canton, graph, visited)
				if size > maxComponent {
					maxComponent = size
				}
			}
		}

		connectivity[player] = float64(maxComponent)
	}

	return normalize(connectivity[current], connectivity[opponent])
}

func (gs *GameState) calculateBorderScore() float64 {
	current := gs.CurrentPlayer
	opponent := gs.NextPlayer()
	borderStrength := make(map[int]float64) // By player

	// Calculate border strength for each player
	for canton, owner := range gs.Ownership {
		if owner != current && owner != opponent {
			continue
		}

		myTroops := float64(gs.TroopCounts[canton])
		enemyBorders := 0
		troopDiff := 0.0
		// Tally troop difference with all enemy neighbors
		// factors in connectedness: higher connections, higher strategic value
		// (e.g. more opportunities to attack, potentially a chokepoint, etc)
		for _, neighbor := range gs.Map.Cantons[canton].AdjacentIDs {
			if gs.Ownership[neighbor] != owner {
				enemyBorders++
				enemyTroops := float64(gs.TroopCounts[neighbor])
				// Troop difference mimics line of attack
				// Not capped because an arbitray number of attacks can be launched till only 1 troop left
				troopDiff += myTroops - enemyTroops
			}
		}
		// Scale by square root to favor but not overly favor having multiple lines of attack (conversely, if troop diff is negative, penalize under multiple lines of attack)
		if enemyBorders > 0 {
			borderStrength[owner] += troopDiff / math.Sqrt(float64(enemyBorders))
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

// dfs performs depth-first search starting from a territory to find size of connected component
// Parameters:
// - start: ID of starting territory
// - graph: map of territory ID to slice of connected friendly territory IDs
// - visited: map tracking which territories have been visited
// Returns size of connected component containing start territory
func (gs *GameState) dfs(start int, graph map[int][]int, visited map[int]bool) int {
	// Skip if already visited
	if visited[start] {
		return 0
	}

	// Mark current territory as visited
	visited[start] = true

	// Initialize size with current territory
	size := 1

	// Recursively visit all unvisited neighbors
	for _, neighbor := range graph[start] {
		size += gs.dfs(neighbor, graph, visited)
	}

	return size
}

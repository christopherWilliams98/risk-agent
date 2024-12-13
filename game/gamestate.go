package game

import (
	"fmt"
	"math/rand"
	"sort"
)

// GameState represents the dynamic state of the game at any point. stuff that will change during the game, (everything except the map - which is static), and at some point even the rules.
type GameState struct {
	Map         *Map  // Reference to the static game map
	TroopCounts []int // Troop counts per canton, indexed by canton ID
	Ownership   []int // Owner IDs per canton, indexed by canton ID (-1 indicates unowned)
	Rules       Rules // The set of game rules to apply
}

// NewGameState initializes and returns a new GameState.
func NewGameState(m *Map, rules Rules) *GameState {
	numCantons := len(m.Cantons)
	gs := &GameState{
		Map:         m,
		TroopCounts: make([]int, numCantons),
		Ownership:   make([]int, numCantons),
		Rules:       rules,
	}
	// Initialize all cantons as unowned (-1)
	for i := range gs.Ownership {
		gs.Ownership[i] = -1
	}
	return gs
}

// copy of the GameState.
func (gs *GameState) Copy() *GameState {
	// Copy troop counts
	troopCountsCopy := make([]int, len(gs.TroopCounts))
	copy(troopCountsCopy, gs.TroopCounts)

	// Copy ownership
	ownershipCopy := make([]int, len(gs.Ownership))
	copy(ownershipCopy, gs.Ownership)

	return &GameState{
		Map:         gs.Map,
		TroopCounts: troopCountsCopy,
		Ownership:   ownershipCopy,
	}
}

// MoveTroops transfers troops between two cantons owned by the same player.
func (gs *GameState) MoveTroops(fromCantonID, toCantonID, numTroops int) error {
	// Check ownership
	if gs.Ownership[fromCantonID] != gs.Ownership[toCantonID] {
		return fmt.Errorf("cannot move troops: cantons are not owned by the same player")
	}
	// Check adjacency
	if !gs.AreAdjacent(fromCantonID, toCantonID) {
		return fmt.Errorf("cannot move troops: cantons are not adjacent")
	}
	// Check troop availability
	if gs.TroopCounts[fromCantonID] <= numTroops {
		return fmt.Errorf("cannot move troops: not enough troops in the source canton")
	}
	// Move troops
	gs.TroopCounts[fromCantonID] -= numTroops
	gs.TroopCounts[toCantonID] += numTroops
	return nil
}

func (gs *GameState) Attack(attackerID, defenderID, numTroops int) error {
	// Check ownership
	if gs.Ownership[attackerID] == gs.Ownership[defenderID] {
		return fmt.Errorf("cannot attack: target canton is owned by the same player")
	}
	// Check adjacency
	if !gs.AreAdjacent(attackerID, defenderID) {
		return fmt.Errorf("cannot attack: cantons are not adjacent")
	}
	// Check troop availability
	if gs.TroopCounts[attackerID] <= numTroops {
		return fmt.Errorf("cannot attack: not enough troops to attack")
	}
	// Limit the number of troops based on the rules
	maxAttackTroops := gs.Rules.MaxAttackTroops()
	if numTroops > maxAttackTroops {
		numTroops = maxAttackTroops
	}

	// Roll dice for attacker and defender
	attackerDice := min(numTroops, gs.Rules.MaxAttackTroops())
	defenderDice := min(gs.TroopCounts[defenderID], gs.Rules.MaxDefendTroops())

	attackerRolls := rollDice(attackerDice)
	defenderRolls := rollDice(defenderDice)

	// Sort dice rolls in descending order
	sort.Sort(sort.Reverse(sort.IntSlice(attackerRolls)))
	sort.Sort(sort.Reverse(sort.IntSlice(defenderRolls)))

	// Determine the outcome using the rules
	attackerLosses, defenderLosses := gs.Rules.DetermineAttackOutcome(attackerRolls, defenderRolls)

	// Apply losses
	gs.TroopCounts[attackerID] -= attackerLosses
	gs.TroopCounts[defenderID] -= defenderLosses

	// If defender has no troops left, attacker captures the canton
	if gs.TroopCounts[defenderID] <= 0 {
		gs.Ownership[defenderID] = gs.Ownership[attackerID]
		// Move at least one troop into the captured canton
		troopsToMove := numTroops - attackerLosses
		if troopsToMove < 1 {
			troopsToMove = 1
		}
		if troopsToMove > gs.TroopCounts[attackerID] {
			troopsToMove = gs.TroopCounts[attackerID]
		}
		gs.TroopCounts[attackerID] -= troopsToMove
		gs.TroopCounts[defenderID] = troopsToMove
	}

	return nil
}

func rollDice(num int) []int {
	rolls := make([]int, num)
	for i := 0; i < num; i++ {
		rolls[i] = rand.Intn(6) + 1
	}
	sort.Sort(sort.Reverse(sort.IntSlice(rolls)))
	return rolls
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// AreAdjacent checks if two cantons are adjacent on the map.
func (gs *GameState) AreAdjacent(cantonID1, cantonID2 int) bool {
	for _, adjID := range gs.Map.Cantons[cantonID1].AdjacentIDs {
		if adjID == cantonID2 {
			return true
		}
	}
	return false
}

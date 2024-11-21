package game

import (
	"fmt"
	"math/rand"
	"sort"
)

type Phase int

const (
	InitialPlacementPhase Phase = iota
	ReinforcementPhase
	AttackPhase
	ManeuverPhase
	EndPhase
)

// GameState represents the dynamic state of the game at any point. stuff that will change during the game, (everything except the map - which is static), and at some point even the rules.
type GameState struct {
	Map           *Map        // Reference to the static game map
	TroopCounts   []int       // Troop counts per canton, indexed by canton ID
	Ownership     []int       // Owner IDs per canton, indexed by canton ID (-1 indicates unowned)
	Rules         Rules       // The set of game rules to apply
	CurrentPlayer int         // The current player
	LastMove      Move        // The last move made (for delta)
	Phase         Phase       // The current phase of the game
	PlayerTroops  map[int]int // Troops remaining for initial placement per player
	TroopsToPlace int         // Troops to place during reinforcement phase
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

	// Deep copy PlayerTroops
	playerTroopsCopy := make(map[int]int)
	for key, value := range gs.PlayerTroops {
		playerTroopsCopy[key] = value
	}

	return &GameState{
		Map:           gs.Map,
		TroopCounts:   troopCountsCopy,
		Ownership:     ownershipCopy,
		Rules:         gs.Rules,
		CurrentPlayer: gs.CurrentPlayer,
		LastMove:      gs.LastMove,
		Phase:         gs.Phase,
		PlayerTroops:  playerTroopsCopy,
		TroopsToPlace: gs.TroopsToPlace,
	}
}

// LegalMoves returns all legal moves for the current player.
func (gs *GameState) LegalMoves() []Move {
	switch gs.Phase {
	case ReinforcementPhase:
		return gs.reinforcementMoves()
	case AttackPhase:
		return gs.attackMoves()
	case ManeuverPhase:
		return gs.maneuverMoves()
	default:
		return nil
	}
}

// reinforcementMoves generates all possible reinforcement moves for the current player.
func (gs *GameState) reinforcementMoves() []Move {
	var moves []Move
	remainingTroops := gs.TroopsToPlace

	// Get enemy-adjacent territories
	enemyAdjacentTerritories := gs.getEnemyAdjacentTerritories()

	// Possible troop amounts: one, half, all
	troopAmounts := []int{1, remainingTroops / 2, remainingTroops}

	for _, cantonID := range enemyAdjacentTerritories {
		for _, amount := range troopAmounts {
			if amount > 0 && amount <= remainingTroops {
				moves = append(moves, &GameMove{
					ActionType: ReinforceAction,
					ToCantonID: cantonID,
					NumTroops:  amount,
				})
			}
		}
	}
	return moves
}

func (gs *GameState) attackMoves() []Move {
	var moves []Move

	for cantonID, owner := range gs.Ownership {
		if owner == gs.CurrentPlayer && gs.TroopCounts[cantonID] > 1 {
			maxTroops := min(gs.TroopCounts[cantonID]-1, gs.Rules.MaxAttackTroops())
			for numTroops := 1; numTroops <= maxTroops; numTroops++ {
				for _, adjID := range gs.Map.Cantons[cantonID].AdjacentIDs {
					if gs.Ownership[adjID] != gs.CurrentPlayer {
						moves = append(moves, &GameMove{
							ActionType:   AttackAction,
							FromCantonID: cantonID,
							ToCantonID:   adjID,
							NumTroops:    numTroops,
						})
					}
				}
			}
		}
	}

	moves = append(moves, &GameMove{
		ActionType: PassAction,
	})
	return moves
}

func (gs *GameState) getPlayerTerritoriesWithTroops() []int {
	var territories []int
	for cantonID, owner := range gs.Ownership {
		if owner == gs.CurrentPlayer && gs.TroopCounts[cantonID] > 1 {
			territories = append(territories, cantonID)
		}
	}
	return territories
}

// maneuverMoves generates all possible maneuver moves for the current player.
func (gs *GameState) getEnemyAdjacentOwnedTerritories() []int {
	territoriesSet := make(map[int]struct{})
	for cantonID, owner := range gs.Ownership {
		if owner == gs.CurrentPlayer {
			for _, adjID := range gs.Map.Cantons[cantonID].AdjacentIDs {
				if gs.Ownership[adjID] != gs.CurrentPlayer {
					territoriesSet[cantonID] = struct{}{}
					break
				}
			}
		}
	}
	var territories []int
	for t := range territoriesSet {
		territories = append(territories, t)
	}
	return territories
}

func (gs *GameState) maneuverMoves() []Move {
	var moves []Move

	fromTerritories := gs.getPlayerTerritoriesWithTroops()
	toTerritories := gs.getEnemyAdjacentOwnedTerritories()

	for _, fromID := range fromTerritories {
		for _, toID := range toTerritories {
			if fromID != toID && gs.AreAdjacent(fromID, toID) {
				maxTroops := gs.TroopCounts[fromID] - 1
				halfTroops := maxTroops / 2
				troopAmounts := []int{1, halfTroops, maxTroops}

				for _, numTroops := range troopAmounts {
					if numTroops > 0 {
						moves = append(moves, &GameMove{
							ActionType:   ManeuverAction,
							FromCantonID: fromID,
							ToCantonID:   toID,
							NumTroops:    numTroops,
						})
					}
				}
			}
		}
	}

	moves = append(moves, &GameMove{
		ActionType: PassAction,
	})
	return moves
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

// Player returns the identifier of the current player.
func (gs *GameState) Player() string {
	return fmt.Sprintf("Player%d", gs.CurrentPlayer)
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

func (gs *GameState) Delta() map[string]any {
	if gs.LastMove == nil {
		return nil
	}
	gameMove := gs.LastMove.(*GameMove)
	return map[string]any{
		"ActionType":   gameMove.ActionType,
		"FromCantonID": gameMove.FromCantonID,
		"ToCantonID":   gameMove.ToCantonID,
		"NumTroops":    gameMove.NumTroops,
	}
}

func (gs *GameState) Play(move Move) State {
	newGs := gs.Copy()
	gameMove := move.(*GameMove)

	switch gs.Phase {
	case ReinforcementPhase:
		if gameMove.ActionType == ReinforceAction {
			// Apply reinforcement move
			newGs.TroopCounts[gameMove.ToCantonID] += gameMove.NumTroops
			// Subtract placed troops from troops to place
			newGs.TroopsToPlace -= gameMove.NumTroops
			if newGs.TroopsToPlace <= 0 {
				newGs.AdvancePhase()
			}
		} else {
			// Invalid action for this phase
			return newGs
		}
	case AttackPhase:
		if gameMove.ActionType == AttackAction {
			// Apply attack move
			err := newGs.Attack(gameMove.FromCantonID, gameMove.ToCantonID, gameMove.NumTroops)
			if err != nil {
				// Handle error
				return newGs
			}
		} else if gameMove.ActionType == PassAction {
			// End attack phase
			newGs.AdvancePhase()
		} else {
			// Invalid action for this phase
			return newGs
		}
	case ManeuverPhase:
		if gameMove.ActionType == ManeuverAction {
			// Apply maneuver move
			err := newGs.MoveTroops(gameMove.FromCantonID, gameMove.ToCantonID, gameMove.NumTroops)
			if err != nil {
				// Handle error
				return newGs
			}
			// After one maneuver, end the phase
			newGs.AdvancePhase()
		} else if gameMove.ActionType == PassAction {
			// End maneuver phase
			newGs.AdvancePhase()
		} else {
			// Invalid action for this phase
			return newGs
		}
	}

	// Update the last move
	newGs.LastMove = move
	return newGs
}

// advancePhase moves the game to the next phase or next player's turn.
func (gs *GameState) AdvancePhase() {
	switch gs.Phase {
	case InitialPlacementPhase:
		gs.CurrentPlayer = gs.NextPlayer()
		if gs.PlayerTroops[1] == 0 && gs.PlayerTroops[2] == 0 {
			gs.Phase = ReinforcementPhase
			gs.calculateTroopsToPlace()
		}
	case ReinforcementPhase:
		gs.Phase = AttackPhase
	case AttackPhase:
		gs.Phase = ManeuverPhase
	case ManeuverPhase:
		gs.Phase = ReinforcementPhase
		gs.CurrentPlayer = gs.NextPlayer()
		gs.calculateTroopsToPlace()
	}
}

func (gs *GameState) getEnemyAdjacentTerritories() []int {
	territoriesSet := make(map[int]struct{})
	for cantonID, owner := range gs.Ownership {
		if owner == gs.CurrentPlayer {
			for _, adjID := range gs.Map.Cantons[cantonID].AdjacentIDs {
				if gs.Ownership[adjID] != gs.CurrentPlayer {
					territoriesSet[cantonID] = struct{}{}
					break
				}
			}
		}
	}
	var territories []int
	for t := range territoriesSet {
		territories = append(territories, t)
	}
	return territories
}

// calculateTroopsToPlace calculates the number of troops the current player should place.
func (gs *GameState) calculateTroopsToPlace() {
	numTerritories := 0
	troops := 0

	// Count owned territories
	for _, owner := range gs.Ownership {
		if owner == gs.CurrentPlayer {
			numTerritories++
		}
	}
	troops += max(3, numTerritories/3)

	// Check for region bonuses
	for _, region := range gs.Map.Regions {
		ownsAll := true
		for _, cantonID := range region.CantonIDs {
			if gs.Ownership[cantonID] != gs.CurrentPlayer {
				ownsAll = false
				break
			}
		}
		if ownsAll {
			troops += BONUS_TROOPS_REGION
		}
	}
	gs.TroopsToPlace = troops
}

// we could add more players to the game, 2 players now for simplicity
func (gs *GameState) NextPlayer() int {
	if gs.CurrentPlayer == 1 {
		return 2
	}
	return 1
}

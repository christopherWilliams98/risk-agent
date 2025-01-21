package game

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sort"
)

type Phase int

const (
	ReinforcementPhase Phase = iota
	AttackPhase
	ManeuverPhase
	EndPhase
)

// GameState represents the dynamic state of the game at any point. stuff that will change during the game, (everything except the map - which is static), and at some point even the rules.
type GameState struct {
	Map               *Map         // Reference to the static game map
	TroopCounts       []int        // Troop counts per canton, indexed by canton ID
	Ownership         []int        // Owner IDs per canton, indexed by canton ID (-1 indicates unowned)
	Rules             Rules        // The set of game rules to apply
	CurrentPlayer     int          // The current player
	LastMove          Move         // The last move made (for delta)
	Phase             Phase        // The current phase of the game
	PlayerTroops      map[int]int  // Troops remaining for initial placement per player
	TroopsToPlace     int          // Troops to place during reinforcement phase
	Cards             []RiskCard   // Card deck
	DiscardedCards    []RiskCard   // Discarded cards
	PlayerHands       [][]RiskCard // Player hands
	Exchanges         int          // Number of exchanges
	ConqueredThisTurn bool         // Whether a territory was conquered this turn
	Won               string       // The player winner of the game, "" if no winner yet
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

	// Initialize all cantons to unowned
	for i := range gs.Ownership {
		gs.Ownership[i] = -1
	}

	numPlayers := 2
	STARTING_UNITS := 3 // or any other number

	cantonIDs := make([]int, numCantons)
	for i := 0; i < numCantons; i++ {
		cantonIDs[i] = i
	}
	rand.Shuffle(numCantons, func(i, j int) {
		cantonIDs[i], cantonIDs[j] = cantonIDs[j], cantonIDs[i]
	})

	// Assign territories in round-robin fashion to players as per Jean's code
	for i, cid := range cantonIDs {
		ownerID := (i % numPlayers) + 1
		gs.Ownership[cid] = ownerID
		gs.TroopCounts[cid] = STARTING_UNITS
	}

	// for i, owner := range gs.Ownership {
	// fmt.Printf("[NewGameState] Canton %d -> Player %d, TroopCounts = %d\n",
	// 	i, owner, gs.TroopCounts[i])
	// }
	// fmt.Printf("[NewGameState] Done assigning. Now calling calculateTroopsToPlace() for Player %d\n",
	// 	gs.CurrentPlayer)

	gs.InitCards()
	// TODO: Randomize starting player
	gs.CurrentPlayer = rand.Intn(numPlayers) + 1
	gs.PlayerHands = make([][]RiskCard, 3)
	gs.PlayerHands[1] = []RiskCard{}
	gs.PlayerHands[2] = []RiskCard{}

	gs.Phase = ReinforcementPhase
	gs = gs.calculateTroopsToPlace()
	return gs
}

func (gs GameState) Copy() GameState {
	troopCountsCopy := make([]int, len(gs.TroopCounts))
	copy(troopCountsCopy, gs.TroopCounts)

	ownershipCopy := make([]int, len(gs.Ownership))
	copy(ownershipCopy, gs.Ownership)

	playerTroopsCopy := make(map[int]int)
	for k, v := range gs.PlayerTroops {
		playerTroopsCopy[k] = v
	}

	playerHandsCopy := make([][]RiskCard, len(gs.PlayerHands))
	for i, hand := range gs.PlayerHands {
		handCopy := make([]RiskCard, len(hand))
		copy(handCopy, hand)
		playerHandsCopy[i] = handCopy
	}

	cardsCopy := make([]RiskCard, len(gs.Cards))
	copy(cardsCopy, gs.Cards)

	discardedCardsCopy := make([]RiskCard, len(gs.DiscardedCards))
	copy(discardedCardsCopy, gs.DiscardedCards)

	return GameState{
		Map:               gs.Map,
		TroopCounts:       troopCountsCopy,
		Ownership:         ownershipCopy,
		Rules:             gs.Rules,
		CurrentPlayer:     gs.CurrentPlayer,
		LastMove:          gs.LastMove,
		Phase:             gs.Phase,
		PlayerTroops:      playerTroopsCopy,
		TroopsToPlace:     gs.TroopsToPlace,
		Cards:             cardsCopy,
		DiscardedCards:    discardedCardsCopy,
		PlayerHands:       playerHandsCopy,
		Exchanges:         gs.Exchanges,
		ConqueredThisTurn: gs.ConqueredThisTurn,
		Won:               gs.Won,
	}
}

func (gs GameState) InitCards() {
	types := []CardType{Infantry, Cavalry, Artillery}
	tIndex := 0
	for i := 0; i < 24; i++ {
		gs.Cards = append(gs.Cards, RiskCard{Type: types[i%3], TerritoryID: i})
		tIndex++
	}
	// 2 Wild cards
	gs.Cards = append(gs.Cards, RiskCard{Type: Wild, TerritoryID: -1})
	gs.Cards = append(gs.Cards, RiskCard{Type: Wild, TerritoryID: -1})

	// Shuffle the deck
	rand.Shuffle(len(gs.Cards), func(i, j int) {
		gs.Cards[i], gs.Cards[j] = gs.Cards[j], gs.Cards[i]
	})
}

func (gs GameState) DrawCard() (RiskCard, bool) {
	if len(gs.Cards) == 0 {
		// If no cards left, reshuffle discarded into deck
		if len(gs.DiscardedCards) == 0 {
			// No cards at all
			return RiskCard{}, false
		}
		gs.Cards = append(gs.Cards, gs.DiscardedCards...)
		gs.DiscardedCards = nil
		rand.Shuffle(len(gs.Cards), func(i, j int) {
			gs.Cards[i], gs.Cards[j] = gs.Cards[j], gs.Cards[i]
		})
	}
	card := gs.Cards[0]
	gs.Cards = gs.Cards[1:]
	return card, true
}

func (gs GameState) AwardCardIfEligible() GameState {
	newGs := gs.Copy()
	if newGs.ConqueredThisTurn {
		card, ok := newGs.DrawCard()
		if ok {
			newGs.PlayerHands[newGs.CurrentPlayer] = append(newGs.PlayerHands[newGs.CurrentPlayer], card)
		}
	}
	newGs.ConqueredThisTurn = false // Reset for next turn
	return newGs
}

func (gs *GameState) HandleCardTrading() {
	playerID := gs.CurrentPlayer
	hand := gs.PlayerHands[playerID]

	// If player has 5 or more cards, they MUST trade in.
	// Otherwise, they may (but we are implementing a must trade sets if available logic)
	for {
		if len(hand) < 3 {
			break
		}
		set := gs.FindSet(hand)
		if set == nil {
			break
		}
		hand = gs.TradeInSet(hand, set)
	}
	gs.PlayerHands[playerID] = hand
}

// Find a set of cards in the player's hand. We return the indices of the chosen set.
// Sets:
// 1) Three of a kind (Infantry, Infantry, Infantry OR Cavalry, Cavalry, Cavalry OR Artillery, Artillery, Artillery)
// 2) One of each (Infantry, Cavalry, Artillery)
// 3) Any two plus a Wild
func (gs GameState) FindSet(hand []RiskCard) []int {
	n := len(hand)
	// Count types
	countType := map[CardType][]int{} // type -> indices
	for i, c := range hand {
		countType[c.Type] = append(countType[c.Type], i)
	}

	// Check three of a kind
	for t, indices := range countType {
		if t != Wild && len(indices) >= 3 {
			return indices[:3]
		}
	}

	// Check one of each (Inf, Cav, Art)
	inf, cav, art := countType[Infantry], countType[Cavalry], countType[Artillery]
	if len(inf) > 0 && len(cav) > 0 && len(art) > 0 {
		// Take the first one of each
		return []int{inf[0], cav[0], art[0]}
	}

	// Check two plus a wild
	wilds := countType[Wild]
	if len(wilds) > 0 {
		// Try to find any two of the other types
		otherTypes := []CardType{Infantry, Cavalry, Artillery}
		for _, t := range otherTypes {
			if len(countType[t]) >= 2 {
				return []int{countType[t][0], countType[t][1], wilds[0]}
			}
		}
		// Or one of one type and one of another type plus wild
		var nonWildIndices []int
		for i := 0; i < n; i++ {
			if hand[i].Type != Wild {
				nonWildIndices = append(nonWildIndices, i)
			}
		}
		if len(nonWildIndices) >= 2 {
			return []int{nonWildIndices[0], nonWildIndices[1], wilds[0]}
		}
	}

	// No set found
	return nil
}

// Trade in a given set of cards, remove from hand, put into discard, increment gs.Exchanges, and give armies.
func (gs GameState) TradeInSet(hand []RiskCard, setIndices []int) []RiskCard {
	playerID := gs.CurrentPlayer
	// Extract the cards
	var set []RiskCard
	sort.Sort(sort.Reverse(sort.IntSlice(setIndices)))
	for _, idx := range setIndices {
		set = append(set, hand[idx])
		hand = append(hand[:idx], hand[idx+1:]...)
	}

	// Move set to discarded
	gs.DiscardedCards = append(gs.DiscardedCards, set...)

	// Increment exchanges count (global)
	gs.Exchanges++

	// Calculate how many armies
	armiesFromSet := gs.ArmiesForThisExchange(gs.Exchanges)
	gs.TroopsToPlace += armiesFromSet

	// Check territory bonus
	extraArmiesGranted := 0
	for _, card := range set {
		if card.TerritoryID >= 0 && gs.Ownership[card.TerritoryID] == playerID && extraArmiesGranted < 2 {
			// Place 2 extra armies on this territory
			gs.TroopCounts[card.TerritoryID] += 2
			extraArmiesGranted += 2
		}
		if extraArmiesGranted == 2 {
			break
		}
	}

	return hand
}

// ArmiesForThisExchange calculates how many armies to award given the nth exchange. //TODO REFACTOR IN METADATA / RULES
// The first set traded in - 4 armies
// The second set - 6 armies
// The third set - 8 armies
// The fourth set - 10 armies
// The fifth set - 12 armies
// The sixth set - 15 armies
// After the sixth set, each additional set is worth 5 more than the previous.
func (gs *GameState) ArmiesForThisExchange(exchangeNumber int) int {

	switch exchangeNumber {
	case 1:
		return 4
	case 2:
		return 6
	case 3:
		return 8
	case 4:
		return 10
	case 5:
		return 12
	case 6:
		return 15
	default:
		// After sixth, it's 15 + 5*(exchangeNumber-6)
		return 15 + 5*(exchangeNumber-6)
	}
}

// LegalMoves returns all legal moves for the current player.
func (gs GameState) LegalMoves() []Move {
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
func (gs GameState) reinforcementMoves() []Move {
	var moves []Move
	remainingTroops := gs.TroopsToPlace
	enemyAdjacentTerritories := gs.getEnemyAdjacentTerritories()

	//fmt.Printf("[reinforcementMoves] Player %d, TroopsToPlace=%d\n", gs.CurrentPlayer, gs.TroopsToPlace)
	//fmt.Printf("[reinforcementMoves] EnemyAdjacentTerritories=%v\n", enemyAdjacentTerritories)

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
	//fmt.Printf("[reinforcementMoves] Generated %d moves\n", len(moves))
	return moves
}

func (gs GameState) attackMoves() []Move {
	var moves []Move
	// fmt.Printf("[attackMoves] Player %d attacking...\n", gs.CurrentPlayer)

	for cantonID, owner := range gs.Ownership {
		if owner == gs.CurrentPlayer && gs.TroopCounts[cantonID] > 1 {
			numTroops := gs.TroopCounts[cantonID] - 1
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

	// Pass always allowed
	moves = append(moves, &GameMove{
		ActionType: PassAction,
	})
	//fmt.Printf("[attackMoves] Generated %d moves\n", len(moves))
	return moves
}

func (gs GameState) getPlayerTerritoriesWithTroops() []int {
	var territories []int
	for cantonID, owner := range gs.Ownership {
		if owner == gs.CurrentPlayer && gs.TroopCounts[cantonID] > 1 {
			territories = append(territories, cantonID)
		}
	}
	return territories
}

func (gs GameState) getPlayerOwnedTerritories() []int {
	var territories []int
	for cantonID, owner := range gs.Ownership {
		if owner == gs.CurrentPlayer {
			territories = append(territories, cantonID)
		}
	}
	return territories
}

// maneuverMoves generates all possible maneuver moves for the current player.
func (gs GameState) maneuverMoves() []Move {
	var moves []Move
	// fmt.Printf("[maneuverMoves] Player %d maneuvering...\n", gs.CurrentPlayer)

	fromTerritories := gs.getPlayerTerritoriesWithTroops()
	toTerritories := gs.getPlayerOwnedTerritories()

	for _, fromID := range fromTerritories {
		for _, toID := range toTerritories {
			if fromID != toID && gs.AreConnected(fromID, toID, gs.CurrentPlayer) {
				maxTroops := gs.TroopCounts[fromID] - 1
				if maxTroops <= 0 {
					continue
				}
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

	// Pass always allowed
	moves = append(moves, &GameMove{
		ActionType: PassAction,
	})
	// fmt.Printf("[maneuverMoves] Generated %d moves\n", len(moves))
	return moves
}

// MoveTroops transfers troops between two cantons owned by the same player.
func (gs GameState) MoveTroops(fromCantonID, toCantonID, numTroops int) error {
	playerID := gs.Ownership[fromCantonID]

	// Check ownership
	if playerID != gs.Ownership[toCantonID] {
		return fmt.Errorf("cannot move troops: cantons are not owned by the same player")
	}
	// Check connectivity
	if !gs.AreConnected(fromCantonID, toCantonID, playerID) {
		return fmt.Errorf("cannot move troops: cantons are not connected")
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

func (gs GameState) Attack(attackerID, defenderID int) (GameState, error) {
	newGs := gs.Copy()
	// Check ownership and adjacency
	if newGs.Ownership[attackerID] == gs.Ownership[defenderID] {
		return newGs, fmt.Errorf("cannot attack: target canton is owned by the same player")
	}
	if !newGs.AreAdjacent(attackerID, defenderID) {
		return newGs, fmt.Errorf("cannot attack: cantons are not adjacent")
	}
	if newGs.TroopCounts[attackerID] <= 1 {
		return newGs, fmt.Errorf("cannot attack: not enough troops to attack")
	}

	// Initialize troop counts
	attackerTroops := gs.TroopCounts[attackerID] - 1 // Must leave at least one troop behind
	defenderTroops := gs.TroopCounts[defenderID]

	// Simulate attack rounds
	for attackerTroops > 0 && defenderTroops > 0 {
		// Determine dice count
		attackerDice := min(attackerTroops, newGs.Rules.MaxAttackTroops())
		defenderDice := min(defenderTroops, newGs.Rules.MaxDefendTroops())

		attackerRolls := rollDice(attackerDice)
		defenderRolls := rollDice(defenderDice)

		// Sort dice rolls
		sort.Sort(sort.Reverse(sort.IntSlice(attackerRolls)))
		sort.Sort(sort.Reverse(sort.IntSlice(defenderRolls)))

		// Determine outcome
		attackerLosses, defenderLosses := newGs.Rules.DetermineAttackOutcome(attackerRolls, defenderRolls)

		// Apply losses
		attackerTroops -= attackerLosses
		defenderTroops -= defenderLosses

		// Break if either side is defeated
		if attackerTroops <= 0 || defenderTroops <= 0 {
			break
		}
	}

	// Update troop counts and ownership
	newGs.TroopCounts[attackerID] = attackerTroops + 1 // Add back the troop left behind

	if defenderTroops <= 0 {
		// Capture the canton
		newGs.Ownership[defenderID] = newGs.Ownership[attackerID]
		moveTroops := newGs.TroopCounts[attackerID] - 1 // Move all but one troop
		newGs.TroopCounts[attackerID] -= moveTroops
		newGs.TroopCounts[defenderID] = moveTroops
		newGs.ConqueredThisTurn = true
	} else {
		// Defender survives
		newGs.TroopCounts[defenderID] = defenderTroops
		newGs.ConqueredThisTurn = false
	}

	return newGs, nil
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
func (gs GameState) Player() string {
	return fmt.Sprintf("Player%d", gs.CurrentPlayer)
}

// AreAdjacent checks if two cantons are adjacent on the map.
func (gs GameState) AreAdjacent(cantonID1, cantonID2 int) bool {
	for _, adjID := range gs.Map.Cantons[cantonID1].AdjacentIDs {
		if adjID == cantonID2 {
			return true
		}
	}
	return false
}

func (gs GameState) Hash() StateHash {
	hasher := fnv.New64a()

	// Hash current player
	binary.Write(hasher, binary.LittleEndian, int64(gs.CurrentPlayer))

	// Hash phase
	binary.Write(hasher, binary.LittleEndian, int64(gs.Phase))

	// Hash troop counts
	for _, count := range gs.TroopCounts {
		binary.Write(hasher, binary.LittleEndian, int64(count))
	}

	// Hash ownerships
	for _, owner := range gs.Ownership {
		binary.Write(hasher, binary.LittleEndian, int64(owner))
	}

	// Hash TroopsToPlace
	binary.Write(hasher, binary.LittleEndian, int64(gs.TroopsToPlace))

	// Hash PlayerTroops
	for playerID, troops := range gs.PlayerTroops {
		binary.Write(hasher, binary.LittleEndian, int64(playerID))
		binary.Write(hasher, binary.LittleEndian, int64(troops))
	}

	return StateHash(hasher.Sum64())
}

// Just BFS
func (gs GameState) AreConnected(fromID, toID, playerID int) bool {
	if fromID == toID {
		return true
	}
	visited := make(map[int]bool)
	queue := []int{fromID}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		for _, adjID := range gs.Map.Cantons[current].AdjacentIDs {
			if gs.Ownership[adjID] != playerID {
				continue
			}
			if adjID == toID {
				return true
			}
			if !visited[adjID] {
				queue = append(queue, adjID)
			}
		}
	}
	return false
}

func (gs GameState) Play(move Move) State {
	newGs := gs.Copy()
	gameMove := move.(*GameMove)

	// fmt.Printf("[Play] Called with Phase=%d, ActionType=%d, TroopsToPlace=%d\n",
	// gs.Phase, gameMove.ActionType, gs.TroopsToPlace)
	switch gs.Phase {
	case ReinforcementPhase:
		if gameMove.ActionType == ReinforceAction {
			// Apply reinforcement move
			newGs.TroopCounts[gameMove.ToCantonID] += gameMove.NumTroops
			// Subtract placed troops from troops to place
			newGs.TroopsToPlace -= gameMove.NumTroops

			// Panic if TroopsToPlace becomes negative
			if newGs.TroopsToPlace < 0 {
				panic("TroopsToPlace cannot be negative")
			}

			if newGs.TroopsToPlace == 0 {
				newGs = newGs.AdvancePhase()
			}
		} else {
			// Invalid action for this phase
			panic("Invalid action for ReinforcementPhase")
		}
	case AttackPhase:
		if gameMove.ActionType == AttackAction {
			// Apply attack move
			newState, err := newGs.Attack(gameMove.FromCantonID, gameMove.ToCantonID)
			newGs = newState
			if err != nil {
				// Panic on error
				panic(err)
			}
		} else if gameMove.ActionType == PassAction {
			// End attack phase
			newGs = newGs.AdvancePhase()
		} else {
			// Invalid action for this phase
			panic("Invalid action for AttackPhase")
		}
	case ManeuverPhase:
		if gameMove.ActionType == ManeuverAction {
			// Apply maneuver move
			err := newGs.MoveTroops(gameMove.FromCantonID, gameMove.ToCantonID, gameMove.NumTroops)
			if err != nil {
				// Panic on error
				panic(err)
			}
			// After one maneuver, end the phase
			newGs = newGs.AdvancePhase()
		} else if gameMove.ActionType == PassAction {
			// End maneuver phase
			newGs = newGs.AdvancePhase()
		} else {
			// Invalid action for this phase
			panic("Invalid action for ManeuverPhase")
		}
	default:
		panic("Unknown game phase")
	}

	// Update the last move
	newGs.LastMove = move

	// Check for winner
	newGs.Won = newGs.CheckWinner()

	return &newGs
}

// advancePhase moves the game to the next phase or next player's turn.
func (gs GameState) AdvancePhase() GameState {
	newGs := gs.Copy()
	switch newGs.Phase {
	case ReinforcementPhase:
		newGs.Phase = AttackPhase
	case AttackPhase:
		newGs.Phase = ManeuverPhase
	case ManeuverPhase:
		newGs.AwardCardIfEligible()
		newGs.Phase = ReinforcementPhase
		newGs.CurrentPlayer = gs.NextPlayer()
		newGs.HandleCardTrading()
		newGs = *newGs.calculateTroopsToPlace()
	}
	// fmt.Printf("[AdvancePhase] old=%v => new=%v (Player=%d)\n",
	// 	gs.Phase, newGs.Phase, newGs.CurrentPlayer)
	return newGs
}

func (gs GameState) getEnemyAdjacentTerritories() []int {
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
func (gs GameState) calculateTroopsToPlace() *GameState {
	newGs := gs.Copy()
	numTerritories := 0

	for _, owner := range gs.Ownership {
		if owner == gs.CurrentPlayer {
			numTerritories++
		}
	}

	troops := max(3, numTerritories/3)

	// Debug:
	// fmt.Printf("[calculateTroopsToPlace] CurrentPlayer=%d, numTerritories=%d, baseTroops=%d\n",gs.CurrentPlayer, numTerritories, troops)

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
			troops += region.Bonus
			// fmt.Printf("[calculateTroopsToPlace] Player %d fully controls region %s => bonus %d\n", gs.CurrentPlayer, region.Name, region.Bonus)
		}
	}
	// debug
	// fmt.Printf("[calculateTroopsToPlace] Final troopsToPlace for Player %d = %d\n\n", gs.CurrentPlayer, troops)

	newGs.TroopsToPlace = troops
	return &newGs
}

func (gs GameState) NextPlayer() int {
	if gs.CurrentPlayer == 1 {
		return 2
	}
	return 1
}

// gets the winner of the game
func (gs GameState) Winner() string {
	return gs.Won
}

func (gs GameState) CheckWinner() string {
	// Count how many territories each player owns
	playerTerritories := make(map[int]int)
	for _, owner := range gs.Ownership {
		if owner > 0 {
			playerTerritories[owner]++
		}
	}

	// If only one player has territories, that player is the winner
	if len(playerTerritories) == 1 {
		for playerID := range playerTerritories {
			return fmt.Sprintf("Player%d", playerID)
		}
	}

	// otherwise return empty string
	return ""
}

// assigning `troopsPerTerritory` troops to each territory.
func (gs GameState) AssignTerritoriesEqually(numPlayers, troopsPerTerritory int) {
	totalTerritories := len(gs.Map.Cantons)
	for id := 0; id < totalTerritories; id++ {
		playerID := (id % numPlayers) + 1 // 1,2,1,2,...
		gs.Ownership[id] = playerID
		gs.TroopCounts[id] = troopsPerTerritory
	}
}

func IsMoveValidForPhase(phase Phase, move Move) bool {
	gm, ok := move.(*GameMove)
	if !ok {
		return false
	}

	switch phase {
	case ReinforcementPhase:
		return gm.ActionType == ReinforceAction || gm.ActionType == PassAction

	case AttackPhase:
		return gm.ActionType == AttackAction || gm.ActionType == PassAction

	case ManeuverPhase:
		return gm.ActionType == ManeuverAction || gm.ActionType == PassAction

	default:
		return false
	}
}

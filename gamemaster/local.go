package gamemaster // TODO: rename to engine

import (
	"fmt"
	"risk/game"
	"sync"
)

type UpdateGetter func() (game.Move, game.State)

type Engine interface {
	Init() (game.State, UpdateGetter)
	Play(game.Move) error
}

type update struct {
	move  game.Move
	state game.State
}

type localEngine struct {
	state    game.State
	updateCh chan update
	gameOver bool
}

var (
	singleLocalEngine *localEngine
	once              sync.Once
)

func NewLocalEngine() *localEngine {
	return &localEngine{}
}

// @Zhou to use this, replace any calls like: engine := NewLocalEngine() with engine := gamemaster.GetLocalEngine()
func GetLocalEngine() *localEngine {
	once.Do(func() {
		singleLocalEngine = &localEngine{}
	})
	return singleLocalEngine
}

func (e *localEngine) Init() (game.State, UpdateGetter) {
	// Create the initial game state
	gameMap := game.CreateMap()
	rules := game.NewStandardRules()
	gs := game.NewGameState(gameMap, rules)

	// Assign territories to players TODO IO (input from players)
	totalTerritories := len(gs.Map.Cantons)
	for id := 0; id < totalTerritories; id++ {
		playerID := (id % 2) + 1 //TODO hardcoded values (number of players)
		gs.Ownership[id] = playerID
		gs.TroopCounts[id] = 1 //TODO hardcoded values (initial troops per territory)
	}

	// Players have remaining troops to place (like original logic) TODO check this
	gs.PlayerTroops = map[int]int{
		1: 27, // 40 - 13
		2: 27, //TODO hardcoded values (initial troops per player)
	}

	gs.Phase = game.InitialPlacementPhase
	gs.CurrentPlayer = 1

	e.state = gs
	e.updateCh = make(chan update, 1) // TODO: buffer size? play around
	//return a copy of the state
	return e.state, func() (game.Move, game.State) {
		select {
		case u, ok := <-e.updateCh:
			if !ok { // Game over
				return nil, nil
			}
			// return copies
			stateCopy := *(u.state.(*game.GameState))
			return u.move, &stateCopy
		default:
			// No updates yet, return nil immediately
			return nil, nil
		}
	}
}

func (e *localEngine) Play(move game.Move) error {
	if e.gameOver {
		return fmt.Errorf("game is over - no moves allowed")
	}

	legalMoves := e.state.LegalMoves()
	if len(legalMoves) == 0 {
		// No legal moves available at this time, so the attempted move is illegal
		return fmt.Errorf("illegal move: no legal moves available")
	}

	// Otherwise, we have some legal moves. Check if the move is among them.
	isLegal := false
	for _, lm := range legalMoves {
		if movesEqual(lm, move) {
			isLegal = true
			break
		}
	}

	if !isLegal {
		return fmt.Errorf("illegal move")
	}

	newState := e.state.Play(move).(*game.GameState)
	e.state = newState

	// Check if game is over after this move //TODO DONT CONSUME! Change this
	if isGameOver(e.state.(*game.GameState)) {
		e.gameOver = true
		// Send final update then close
		e.updateCh <- update{move: move, state: e.state}
		close(e.updateCh)
	} else {
		e.updateCh <- update{move: move, state: e.state}
	}

	return nil
}

func movesEqual(m1, m2 game.Move) bool {
	gm1 := m1.(*game.GameMove)
	gm2 := m2.(*game.GameMove)
	return gm1.ActionType == gm2.ActionType &&
		gm1.FromCantonID == gm2.FromCantonID &&
		gm1.ToCantonID == gm2.ToCantonID &&
		gm1.NumTroops == gm2.NumTroops
}

// isGameOver checks if the game is over.
func isGameOver(gs *game.GameState) bool {
	playerTerritories := make(map[int]int)
	for _, owner := range gs.Ownership {
		if owner != -1 {
			playerTerritories[owner]++
		}
	}
	// If any player has no territories or only one player remains
	return len(playerTerritories) == 1
}

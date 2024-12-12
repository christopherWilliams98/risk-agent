package gamemaster

import (
	"reflect"
	"risk/game"
	"testing"
)

func TestLocalEngineInit(t *testing.T) {
	engine := NewLocalEngine()
	state, getUpdate := engine.Init()

	// Validate initial state
	gs := state.(*game.GameState)

	if gs == nil {
		t.Fatal("expected a GameState, got nil")
	}

	if gs.Phase != game.InitialPlacementPhase {
		t.Errorf("expected Phase = InitialPlacementPhase, got %v", gs.Phase)
	}

	// Check that players have initial troops
	if gs.PlayerTroops[1] != 27 || gs.PlayerTroops[2] != 27 {
		t.Errorf("initial PlayerTroops counts are not as expected: got %+v", gs.PlayerTroops)
	}

	// Check that getUpdate returns nil if no moves have been played
	move, newState := getUpdate()
	if move != nil || newState != nil {
		t.Errorf("expected no update yet, got move=%v state=%v", move, newState)
	}
}

func TestLocalEnginePlay_ValidMove(t *testing.T) {
	engine := NewLocalEngine()
	state, getUpdate := engine.Init()

	gs := state.(*game.GameState)
	// The game starts in the initial placement phase.
	// Let's try placing a troop on a territory owned by player 1.
	// We need to pick a territory owned by player 1 (e.g., territory 0 if it belongs to player 1)
	var ownedTerritory int
	for id, owner := range gs.Ownership {
		if owner == 1 {
			ownedTerritory = id
			break
		}
	}

	// Create a reinforce move (initial placement)
	move := &game.GameMove{
		ActionType:   game.ReinforceAction,
		FromCantonID: 0, // not used in reinforcement
		ToCantonID:   ownedTerritory,
		NumTroops:    1, // place one troop
	}

	err := engine.Play(move)
	if err != nil {
		t.Errorf("expected no error for a valid move, got %v", err)
	}

	// Check update returned
	playedMove, updatedState := getUpdate()
	if playedMove == nil || updatedState == nil {
		t.Fatal("expected an update after playing a move, got none")
	}

	updatedGs := updatedState.(*game.GameState)
	if updatedGs == nil {
		t.Fatal("expected updated state to be a *GameState")
	}

	// Check that troops were placed
	if updatedGs.TroopCounts[ownedTerritory] != gs.TroopCounts[ownedTerritory]+1 {
		t.Errorf("expected troop count to increase by 1 in territory %d", ownedTerritory)
	}

	// Check that PlayerTroops count decreased
	if updatedGs.PlayerTroops[1] != 26 { // was 27, placed 1
		t.Errorf("expected PlayerTroops[1] to be 26 after placing one troop, got %d", updatedGs.PlayerTroops[1])
	}
}

func TestLocalEnginePlay_IllegalMove(t *testing.T) {
	engine := NewLocalEngine()

	// Construct an illegal move: e.g., AttackAction during the InitialPlacementPhase.
	// During InitialPlacementPhase, only ReinforceAction is valid.
	illegalMove := &game.GameMove{
		ActionType:   game.AttackAction,
		FromCantonID: 0,
		ToCantonID:   1,
		NumTroops:    5,
	}

	err := engine.Play(illegalMove)
	if err == nil {
		t.Error("expected error for illegal move, got none")
	}
}

func TestLocalEnginePlay_GameOver(t *testing.T) {

	engine := NewLocalEngine()
	state, getUpdate := engine.Init()
	gs := state.(*game.GameState)

	for i := range gs.Ownership {
		gs.Ownership[i] = 1
	}
	engine.state = gs // force internal state

	// Now, if we try to play any move, the game should be over immediately.

	harmlessMove := &game.GameMove{
		ActionType:   game.PassAction,
		FromCantonID: 0,
		ToCantonID:   0,
		NumTroops:    0,
	}

	// Let's call Play() and see what happens.
	err := engine.Play(harmlessMove)
	if err != nil {
		t.Errorf("did not expect error on pass move, got %v", err)
	}

	// Try getting the update, if the game ended, we should get a final state then nil.
	playedMove, updatedState := getUpdate()
	if playedMove == nil || updatedState == nil {
		t.Errorf("expected a final update before game ends")
	}

	// After one update, the channel should be closed
	playedMove2, updatedState2 := getUpdate()
	if playedMove2 != nil || updatedState2 != nil {
		t.Errorf("expected no updates after game over, got move=%v state=%v", playedMove2, updatedState2)
	}

	// Now if we try to play another move, it should return "game is over"
	err = engine.Play(harmlessMove)
	if err == nil || err.Error() != "game is over - no moves allowed" {
		t.Errorf("expected 'game is over - no moves allowed' error, got %v", err)
	}
}

func TestLocalEngine_IdenticalInitStates(t *testing.T) {
	// Test that Init returns a copy of the state each time.
	engine := NewLocalEngine()
	state1, _ := engine.Init()

	engine2 := NewLocalEngine()
	state2, _ := engine2.Init()

	// They should not be the same pointer, but have the same initial data.
	if reflect.DeepEqual(state1, state2) == false {
		t.Error("expected the same initial state configuration, got differences")
	}
}

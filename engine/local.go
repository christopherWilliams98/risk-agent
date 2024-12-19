package gamemaster

import (
	"risk/game"
)

type Engine struct {
	State  *game.GameState
	Agents []Agent
}

type Update struct {
	Move game.Move
	Hash game.StateHash
}

func LocalEngine(players []string, agents []Agent, m *game.Map, r game.Rules) *Engine {
	if len(players) != len(agents) {
		panic("number of players does not match number of agents")
	}
	if len(players) < 2 {
		panic("need at least two players")
	}

	state := game.NewGameState(m, r)

	state.CurrentPlayer = 1

	eng := &Engine{
		State:  state,
		Agents: agents,
	}
	return eng
}

// Run executes the entire game loop until a winner is found.
func (e *Engine) Run() {

	updates := make([][]Update, len(e.Agents))
	for i := range updates {
		updates[i] = []Update{}
	}

	// while Loop until there's a winner
	for e.State.Winner() == "" {
		currentPlayerID := e.State.CurrentPlayer
		agentIndex := currentPlayerID - 1

		move := e.Agents[agentIndex].FindMove(e.State, updates[agentIndex])

		newState := e.State.Play(move).(*game.GameState)

		u := Update{
			Move: move,
			Hash: newState.Hash(),
		}
		updates[agentIndex] = append(updates[agentIndex], u)

		e.State = newState
	}
}

type Agent interface {
	FindMove(state *game.GameState, recentUpdates []Update) game.Move
}

type MCTSAgent struct {
	PlayerID int
}

func NewMCTSAgent(playerID int) *MCTSAgent {
	return &MCTSAgent{PlayerID: playerID}
}

// simple implementation - always returns first legal move
func (mcts *MCTSAgent) FindMove(state *game.GameState, recentUpdates []Update) game.Move {
	legalMoves := state.LegalMoves()
	if len(legalMoves) == 0 {
		panic("No legal moves available - agent stuck")
	}
	return legalMoves[0]
}

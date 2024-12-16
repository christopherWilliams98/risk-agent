package player // TODO: rename to agent, name conflicts with player string

import (
	"math"
	"math/rand/v2"
	"risk/game"
	"risk/gamemaster"
	"risk/searcher"
)

type Controller interface {
	Run()
}

type trainingController struct {
	player string
	mcts   *searcher.MCTS
	engine gamemaster.Engine
}

func NewTrainingController(player string, mcts *searcher.MCTS, engine gamemaster.Engine) *trainingController {
	return &trainingController{
		player: player,
		mcts:   mcts,
		engine: engine,
	}
}

func (l trainingController) Run() {
	state, getUpdate := l.engine.Init()
	updates := []searcher.Segment{}
	for {
		if state.Player() == l.player {
			// TODO: returns a policy by visit counts, rename this variable
			visits := l.mcts.Simulate(state, updates)
			// TODO: apply a temperature schedule as training progresses
			policy := adjustTemperature(visits, 1.0)
			move := sample(policy)
			// TODO: propagate error
			err := l.engine.Play(move)
			if err != nil {
				panic(err)
			}
			updates = []searcher.Segment{}
		}
		move, state := getUpdate()
		// TODO: handle game over
		updates = append(updates, searcher.Segment{Move: move, State: state})
	}
}

func adjustTemperature(visits map[game.Move]int, temperature float64) map[game.Move]float64 {
	// Compute temperature-adjusted move probabilities
	exponent := 1.0 / temperature
	sum := 0.0
	policy := make(map[game.Move]float64, len(visits))
	for move, visit := range visits {
		prob := math.Pow(float64(visit), exponent)
		sum += prob
		policy[move] = prob
	}
	// Normalize
	for move := range policy {
		policy[move] /= sum
	}
	return policy
}

func sample(policy map[game.Move]float64) game.Move {
	// TODO: seed randomizer globally for testing
	sampled := rand.Float64()
	cumulative := 0.0
	var lastMove game.Move
	for move, prob := range policy {
		lastMove = move
		cumulative += prob
		if sampled < cumulative {
			return move
		}
	}
	return lastMove // Fallback in case of rounding errors
}

// TODO: move to EvaluationController
func findMax(visits map[game.Move]int) game.Move {
	var maxMove game.Move
	maxVisits := -1
	for move, visits := range visits {
		if visits > maxVisits {
			maxVisits = visits
			maxMove = move
		}
	}
	return maxMove
}

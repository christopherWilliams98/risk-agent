package searcher

import "risk/game"

type mockMove struct {
	id         int
	stochastic bool
}

func (m mockMove) IsStochastic() bool {
	return m.stochastic
}

type mockState struct {
	player string
	moves  []game.Move
	played []game.Move
	hash   game.StateHash
}

func (m mockState) Player() string {
	return m.player
}

func (m mockState) LegalMoves() []game.Move {
	return m.moves
}

func (m mockState) Play(move game.Move) game.State {
	return mockState{played: append(m.played, move)}
}

func (m mockState) Hash() game.StateHash {
	return m.hash
}

func (m mockState) Winner() string {
	return ""
}

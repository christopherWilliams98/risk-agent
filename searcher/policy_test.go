package searcher

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewUCT(t *testing.T) {
	t.Run("panics with zero parent visits", func(t *testing.T) {
		require.Panics(t, func() {
			newUCT(2.0, 0)
		}, "Should panic when N is 0")
	})
}

func TestUCTEvaluate(t *testing.T) {
	t.Run("computing UCT value", func(t *testing.T) {
		policy := newUCT(2.0, 100)
		got := policy.evaluate(5.0, 10)

		expected := 5.0/10 + math.Sqrt(2.0*math.Log(100)/10.0)
		require.InDelta(t, expected, got, 0.0001,
			"Should compute q/n + sqrt(c^2*ln(N)/n)")
	})

	t.Run("panics with zero child visits", func(t *testing.T) {
		policy := newUCT(2.0, 100)

		require.Panics(t, func() {
			policy.evaluate(5.0, 0)
		}, "Should panic when n is 0")
	})

	t.Run("exploration term increases with parent visits", func(t *testing.T) {
		// More parent visits -> higher exploration
		policy1 := newUCT(2.0, 100)
		policy2 := newUCT(2.0, 1000)
		rewards := 5.0
		visits := 10.0

		score1 := policy1.evaluate(rewards, visits)
		score2 := policy2.evaluate(rewards, visits)

		require.Greater(t, score2, score1,
			"More parent visits should increase exploration term")
	})

	t.Run("exploration term decreases with child visits", func(t *testing.T) {
		// More child visits -> lower exploration
		policy := newUCT(2.0, 100)
		rewards := 5.0

		score1 := policy.evaluate(rewards, 10)
		score2 := policy.evaluate(rewards, 20)

		require.Greater(t, score1, score2,
			"More child visits should decrease exploration term")
	})

	t.Run("exploitation term increases with rewards", func(t *testing.T) {
		policy := newUCT(2.0, 100)
		visits := 10.0

		score1 := policy.evaluate(5.0, visits)
		score2 := policy.evaluate(10.0, visits)

		require.Greater(t, score2, score1,
			"More rewards should increase exploitation term")
	})
}

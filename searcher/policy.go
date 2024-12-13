package searcher

import "math"

type uct struct {
	numerator float64
}

func newUCT(cSquared float64, N int) *uct {
	if N == 0 {
		panic("N cannot be 0")
	}
	return &uct{numerator: cSquared * math.Log(float64(N))}
}

func (u uct) evaluate(q float64, n int) float64 {
	if n == 0 {
		panic("n cannot be 0")
	}
	// UCT = q/n + sqrt(c^2*ln(N)/n)
	return q/float64(n) + math.Sqrt(u.numerator/float64(n))
}

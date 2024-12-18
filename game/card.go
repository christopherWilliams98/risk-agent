package game

type CardType int

const (
	Infantry  CardType = iota // 0
	Cavalry                   // 1
	Artillery                 // 2
	Wild                      // 3
)

type RiskCard struct {
	Type        CardType
	TerritoryID int
}

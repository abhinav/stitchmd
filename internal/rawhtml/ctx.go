package rawhtml

import "github.com/yuin/goldmark/parser"

// _dataKey is the ContextKey under which the [Pairs] are stored
// in the [parser.Context].
var _dataKey = parser.NewContextKey()

// Pairs is a collection of HTML [Pair]s.
type Pairs []Pair

// GetPairs returns the [Pairs] stored in the given [parser.Context].
// If no [Pairs] are stored, it returns an empty [Pairs].
func GetPairs(ctx parser.Context) Pairs {
	p, _ := ctx.Get(_dataKey).(Pairs)
	return p
}

func (p Pairs) set(ctx parser.Context) {
	ctx.Set(_dataKey, p)
}

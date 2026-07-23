// Package signal recognises trading signals on a chart: a body of consecutive
// candles and a boolean rule over their open, close, high and low prices,
// referenced by name (O1, C1, H1, L1, O2 …). Rules can be chained, each on its
// own interval, so a match on one timeframe can hand off to the next.
package signal

import "github.com/dmitrykvasnikov/trademan/internal/binance"

// price reads one of a candle's four prices by the letter that names it in a
// rule: O open, C close, H high, L low. The parser only ever stores these four,
// so an unknown letter cannot reach here.
func price(field byte, c binance.Candle) float64 {
	switch field {
	case 'O':
		return c.Open
	case 'C':
		return c.Close
	case 'H':
		return c.High
	case 'L':
		return c.Low
	}
	return 0
}

// operand is one side of a comparison: a candle reference like H2 or a literal
// number. Its value is read against the body window the rule is evaluated over.
type operand interface {
	value(window []binance.Candle) float64
}

// ref is a body value named by a field letter and a 1-based candle position, so
// H2 is ref{'H', 2} and reads the second candle's high.
type ref struct {
	field byte
	index int
	text  string // as written, for error messages
}

func (r ref) value(window []binance.Candle) float64 { return price(r.field, window[r.index-1]) }

// number is a literal operand, so a rule can compare a price against a constant
// as well as against another price.
type number float64

func (n number) value([]binance.Candle) float64 { return float64(n) }

// expr is a parsed rule: a boolean tree of comparisons combined with NOT, AND
// and OR. It reports whether it holds for one body window of candles.
type expr interface {
	eval(window []binance.Candle) bool
}

// comparison is the leaf of a rule: two operands and the operator between them.
type comparison struct {
	left  operand
	op    string
	right operand
}

func (c comparison) eval(window []binance.Candle) bool {
	l, r := c.left.value(window), c.right.value(window)
	switch c.op {
	case "<":
		return l < r
	case ">":
		return l > r
	case "<=":
		return l <= r
	case ">=":
		return l >= r
	case "==":
		return l == r
	case "!=":
		return l != r
	}
	return false
}

type notExpr struct{ child expr }

func (n notExpr) eval(window []binance.Candle) bool { return !n.child.eval(window) }

type andExpr struct{ left, right expr }

func (a andExpr) eval(window []binance.Candle) bool {
	return a.left.eval(window) && a.right.eval(window)
}

type orExpr struct{ left, right expr }

func (o orExpr) eval(window []binance.Candle) bool {
	return o.left.eval(window) || o.right.eval(window)
}

package signal

import (
	"strings"
	"testing"

	"github.com/dmitrykvasnikov/trademan/internal/binance"
)

// candle is a shorthand for a bar in the parse and evaluation tests, where only
// the four prices matter.
func candle(open, high, low, close float64) binance.Candle {
	return binance.Candle{Open: open, High: high, Low: low, Close: close}
}

// The rules the task spells out have to parse and evaluate, so they are the
// first thing checked.
func TestParsesTheRulesFromTheTask(t *testing.T) {
	for _, src := range []string{
		"NOT (H1 < H2)",
		"((H1 < H2) OR ((O1 < O2) AND (C1 > L2)))",
		"((L3 > H1) OR (L1 > H3))",
	} {
		if _, err := parse(src, 3); err != nil {
			t.Errorf("%q did not parse: %v", src, err)
		}
	}
}

// Every field letter has to read the price it names, on the candle its index
// points at.
func TestReferencesReadTheRightPrice(t *testing.T) {
	// One candle with four distinct prices, so a wrong field shows up.
	window := []binance.Candle{candle(1, 2, 3, 4)}

	cases := map[string]bool{
		"O1 == 1": true,
		"H1 == 2": true,
		"L1 == 3": true,
		"C1 == 4": true,
		"O1 == 2": false,
	}
	for src, want := range cases {
		e, err := parse(src, 1)
		if err != nil {
			t.Fatalf("%q did not parse: %v", src, err)
		}
		if got := e.eval(window); got != want {
			t.Errorf("%q evaluated to %v, want %v", src, got, want)
		}
	}
}

// The operators, the connectives and their precedence all have to land the way
// they are written.
func TestEvaluatesOperatorsAndPrecedence(t *testing.T) {
	// Two candles: the first rising 10→20, the second rising 30→40.
	window := []binance.Candle{candle(10, 25, 5, 20), candle(30, 45, 25, 40)}

	cases := map[string]bool{
		"H1 < H2":                 true,
		"H1 > H2":                 false,
		"H1 <= H1":                true,
		"H1 >= H2":                false,
		"O1 != C1":                true,
		"C1 == 20":                true,
		"C1 > 100":                false,
		"NOT (H1 < H2)":           false,
		"NOT NOT (H1 < H2)":       true,
		"(H1 < H2) AND (L1 < L2)": true,
		"(H1 > H2) OR (L1 < L2)":  true,
		// AND binds tighter than OR: true OR (false AND false) is true, not
		// (true OR false) AND false.
		"H1 < H2 OR H1 > H2 AND L1 > L2": true,
	}
	for src, want := range cases {
		e, err := parse(src, 2)
		if err != nil {
			t.Fatalf("%q did not parse: %v", src, err)
		}
		if got := e.eval(window); got != want {
			t.Errorf("%q evaluated to %v, want %v", src, got, want)
		}
	}
}

// A mistyped rule has to say what is wrong and where, not just fail.
func TestRejectsMalformedRules(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"reference past the body", "H1 < H4", "body is only 3"},
		{"unknown field letter", "X1 < H1", "not a value"},
		{"zero index", "H0 < H1", "not a value"},
		{"missing operator", "H1 H2", "comparison operator"},
		{"lone equals", "H1 = H2", "comparison operator"},
		{"unclosed bracket", "(H1 < H2", "missing ')'"},
		{"dangling AND", "H1 < H2 AND", "expected a value"},
		{"stray character", "H1 < H2 &", "unexpected character"},
		{"trailing tokens", "H1 < H2 H3", "unexpected"},
		{"empty rule", "", "expected a value"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := parse(c.src, 3)
			if err == nil {
				t.Fatalf("%q was accepted, want a parse error", c.src)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("%q failed with %q, want it to mention %q", c.src, err, c.want)
			}
		})
	}
}

// NewRule guards the body length as well as the text.
func TestNewRuleRejectsAnEmptyBody(t *testing.T) {
	if _, err := NewRule(0, "", "H1 < H1"); err == nil {
		t.Error("a rule with a zero-length body was accepted")
	}
}

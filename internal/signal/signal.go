package signal

import (
	"fmt"
	"time"

	"github.com/dmitrykvasnikov/trademan/internal/binance"
)

// Rule is one link of a signal: a boolean expression checked over a body of
// consecutive candles read on one interval. An empty Interval means the rule is
// read on whatever interval the chart is currently showing.
type Rule struct {
	Body     int
	Interval string
	Src      string // the text the expression was parsed from
	expr     expr
}

// NewRule parses src into a rule whose body spans body candles on interval. It
// fails if the body is empty, the text does not parse, or a reference names a
// candle beyond the body.
func NewRule(body int, interval, src string) (Rule, error) {
	if body < 1 {
		return Rule{}, fmt.Errorf("signal: a rule body must span at least one candle, got %d", body)
	}
	e, err := parse(src, body)
	if err != nil {
		return Rule{}, err
	}
	return Rule{Body: body, Interval: interval, Src: src, expr: e}, nil
}

// Matches slides the rule's body across candles and returns the windows where it
// holds, oldest first and non-overlapping: once the body matches, the search
// resumes at the candle just past it rather than one candle on, so a single run
// of candles is reported as one signal instead of several overlapping ones.
func (r Rule) Matches(candles []binance.Candle) []Match {
	var out []Match
	for start := 0; start+r.Body <= len(candles); {
		if r.expr.eval(candles[start : start+r.Body]) {
			out = append(out, Match{Start: start, End: start + r.Body})
			start += r.Body // resume at the candle next to the matched body
		} else {
			start++
		}
	}
	return out
}

// Match is a window a rule held on: the half-open range [Start, End) of candle
// indices its body spans, in the series it was checked against. The completing
// candle — the one a chart circles — is End-1.
type Match struct {
	Start int
	End   int
}

// Signal is a body-and-rules pattern applied to a coin: one or more rules
// checked in order. It fires where the whole chain occurs in sequence.
type Signal struct {
	Name  string
	Rules []Rule
}

// FVG is the built-in fair-value-gap signal: a three-candle body on the chart's
// own interval whose outer candles leave a gap the middle candle does not close.
func FVG() Signal {
	rule, err := NewRule(3, "", "((L3 > H1) OR (L1 > H3))")
	if err != nil {
		// The rule is a constant, so this can only fail if the grammar changed
		// out from under it — a programming error the tests are there to catch.
		panic("signal: built-in FVG rule does not parse: " + err.Error())
	}
	return Signal{Name: "FVG", Rules: []Rule{rule}}
}

// Marks scans one candle series and returns the completing candle index of every
// match — the candle a chart puts its circle on. It is the single-interval path:
// each rule is read against this one series, which is all the FVG shortcut needs.
// Chaining rules across different timeframes goes through Scan instead.
func (s Signal) Marks(candles []binance.Candle) []int {
	source := func(string) ([]binance.Candle, error) { return candles, nil }
	matches, _ := s.Scan("", source)

	out := make([]int, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.End-1)
	}
	return out
}

// Source supplies the candles for an interval, oldest first. A signal whose
// rules span several timeframes reads each rule's interval through it.
type Source func(interval string) ([]binance.Candle, error)

// Scan finds where the signal fires. Its first rule is checked at every window
// across its own series; each match seeds the rest of the chain, which is taken
// up from the first candle after the previous rule's body — on that rule's own
// interval, so a match on one timeframe can hand off to the next. current is the
// interval substituted wherever a rule leaves its own blank.
//
// Each returned Match is a window of the last rule's series: where the whole
// signal completes. For a single-rule signal that is simply every window the
// rule holds on.
func (s Signal) Scan(current string, source Source) ([]Match, error) {
	if len(s.Rules) == 0 {
		return nil, nil
	}

	// Candles are fetched once per interval and reused, so a chain that reads
	// the same timeframe twice does not ask the source for it twice.
	cache := map[string][]binance.Candle{}
	get := func(r Rule) ([]binance.Candle, error) {
		interval := r.Interval
		if interval == "" {
			interval = current
		}
		if candles, ok := cache[interval]; ok {
			return candles, nil
		}
		candles, err := source(interval)
		if err != nil {
			return nil, fmt.Errorf("signal %q: %s candles: %w", s.Name, interval, err)
		}
		cache[interval] = candles
		return candles, nil
	}

	candles, err := get(s.Rules[0])
	if err != nil {
		return nil, err
	}

	var matches []Match
	seen := map[Match]bool{}
	for _, seed := range s.Rules[0].Matches(candles) {
		boundary := candles[seed.End-1].CloseTime
		final, ok, err := s.complete(1, boundary, seed, get)
		if err != nil {
			return nil, err
		}
		// Different seeds can converge on the same completion; each fires once.
		if ok && !seen[final] {
			seen[final] = true
			matches = append(matches, final)
		}
	}
	return matches, nil
}

// complete carries the chain on from rule k, whose body must start no earlier
// than after — the moment the previous rule's body closed. It takes the earliest
// window of each remaining rule that lets the whole chain finish, and returns the
// last rule's window. Once k is past the final rule, prev is that window.
func (s Signal) complete(k int, after time.Time, prev Match, get func(Rule) ([]binance.Candle, error)) (Match, bool, error) {
	if k >= len(s.Rules) {
		return prev, true, nil
	}

	rule := s.Rules[k]
	candles, err := get(rule)
	if err != nil {
		return Match{}, false, err
	}

	for start := firstAfter(candles, after); start+rule.Body <= len(candles); start++ {
		if !rule.expr.eval(candles[start : start+rule.Body]) {
			continue
		}
		window := Match{Start: start, End: start + rule.Body}
		final, ok, err := s.complete(k+1, candles[window.End-1].CloseTime, window, get)
		if err != nil {
			return Match{}, false, err
		}
		if ok {
			return final, true, nil
		}
	}
	return Match{}, false, nil
}

// firstAfter is the index of the first candle that opens at or after t, or the
// length of the series when none does. It is how the next rule skips past the
// candles the previous rule's body already covered.
func firstAfter(candles []binance.Candle, t time.Time) int {
	for i, c := range candles {
		if !c.OpenTime.Before(t) {
			return i
		}
	}
	return len(candles)
}

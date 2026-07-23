package signal

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/dmitrykvasnikov/trademan/internal/binance"
)

var epoch = time.Unix(0, 0).UTC()

// gapSeries builds candles at one per step from a list of [high, low] pairs,
// with open and close at the midpoint — enough for the price-range rules like
// FVG, which only read highs and lows.
func gapSeries(step time.Duration, hl ...[2]float64) []binance.Candle {
	out := make([]binance.Candle, len(hl))
	for i, p := range hl {
		open := epoch.Add(step * time.Duration(i))
		mid := (p[0] + p[1]) / 2
		out[i] = binance.Candle{
			OpenTime:  open,
			CloseTime: open.Add(step - time.Millisecond),
			High:      p[0],
			Low:       p[1],
			Open:      mid,
			Close:     mid,
		}
	}
	return out
}

// closeSeries builds candles whose closes are the given values, one per step. It
// is for the chaining tests, whose rules compare closes.
func closeSeries(step time.Duration, closes ...float64) []binance.Candle {
	out := make([]binance.Candle, len(closes))
	for i, c := range closes {
		open := epoch.Add(step * time.Duration(i))
		out[i] = binance.Candle{
			OpenTime:  open,
			CloseTime: open.Add(step - time.Millisecond),
			Open:      c,
			High:      c,
			Low:       c,
			Close:     c,
		}
	}
	return out
}

func TestFVGIsAThreeCandleGapOnTheCurrentInterval(t *testing.T) {
	fvg := FVG()

	if len(fvg.Rules) != 1 {
		t.Fatalf("FVG has %d rules, want a single one", len(fvg.Rules))
	}
	if r := fvg.Rules[0]; r.Body != 3 || r.Interval != "" {
		t.Errorf("FVG rule is body %d on interval %q, want body 3 on the current interval", r.Body, r.Interval)
	}
}

// FVG marks the candle that completes a gap, once per gap: after a match it
// skips the body it just used, so a single jump is reported as one signal rather
// than one for every overlapping window that spans it. Two separated jumps give
// two marks.
func TestFVGMarksEachGapOnce(t *testing.T) {
	candles := gapSeries(time.Hour,
		[2]float64{20, 10},
		[2]float64{21, 11},
		[2]float64{22, 12},
		[2]float64{40, 30}, // first jump: the gap completes at candle 3
		[2]float64{41, 31},
		[2]float64{42, 32},
		[2]float64{43, 33},
		[2]float64{70, 60}, // second jump: the gap completes at candle 7
		[2]float64{71, 61},
	)

	if got, want := FVG().Marks(candles), []int{3, 7}; !reflect.DeepEqual(got, want) {
		t.Errorf("FVG marked %v, want %v — one mark per gap, its body skipped", got, want)
	}
}

func TestFVGFindsNothingWithoutAGap(t *testing.T) {
	candles := gapSeries(time.Hour,
		[2]float64{20, 10},
		[2]float64{21, 11},
		[2]float64{22, 12},
		[2]float64{23, 13},
		[2]float64{24, 14},
	)

	if got := FVG().Marks(candles); len(got) != 0 {
		t.Errorf("FVG marked %v on a run with no gap in it, want nothing", got)
	}
}

// Matches resumes past a body it has already used, so overlapping windows of the
// same run are not reported as separate signals.
func TestRuleMatchesAdvancesPastEachMatch(t *testing.T) {
	rule, err := NewRule(2, "", "C2 > C1")
	if err != nil {
		t.Fatal(err)
	}

	// Four rising closes. Bodies [0,1] and [2,3] match; the overlapping [1,2]
	// between them is never checked, because [0,1] consumed candle 1.
	candles := closeSeries(time.Hour, 1, 2, 3, 4)

	got := rule.Matches(candles)
	want := []Match{{Start: 0, End: 2}, {Start: 2, End: 4}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("matched %v, want %v — non-overlapping windows", got, want)
	}
}

// A chained signal reads each rule on its own timeframe and only takes up the
// next rule from the first candle after the previous rule's body.
func TestChainSwitchesTimeframeAndStartsAfterThePreviousBody(t *testing.T) {
	// Rule 1 (1h): a rising pair, which appears once at the very start and closes
	// two hours in.
	hourly := closeSeries(time.Hour, 10, 20, 15)

	// Rule 2 (5m): a rising pair. One sits before the two-hour boundary and must
	// be ignored; the one that counts is after it, completing at candle 26.
	closes := make([]float64, 28)
	for i := range closes {
		closes[i] = 10
	}
	closes[10] = 30 // a rising pair at 9→10, before the boundary
	closes[26] = 30 // and the one at 25→26, after it
	fiveMin := closeSeries(5*time.Minute, closes...)

	source := func(interval string) ([]binance.Candle, error) {
		switch interval {
		case "1h":
			return hourly, nil
		case "5m":
			return fiveMin, nil
		}
		return nil, errors.New("no candles for " + interval)
	}

	r1, err := NewRule(2, "1h", "C2 > C1")
	if err != nil {
		t.Fatal(err)
	}
	r2, err := NewRule(2, "5m", "C2 > C1")
	if err != nil {
		t.Fatal(err)
	}
	sig := Signal{Name: "rise-then-rise", Rules: []Rule{r1, r2}}

	matches, err := sig.Scan("", source)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	want := []Match{{Start: 25, End: 27}}
	if !reflect.DeepEqual(matches, want) {
		t.Errorf("chain fired at %v (5m candles), want %v — the rising pair after the 2h boundary", matches, want)
	}
}

// A chain that cannot finish its later rules fires nowhere.
func TestChainThatCannotCompleteFiresNothing(t *testing.T) {
	hourly := closeSeries(time.Hour, 10, 20, 15)
	fiveMin := closeSeries(5*time.Minute, 10, 10, 10, 10, 10, 10) // never rises

	source := func(interval string) ([]binance.Candle, error) {
		if interval == "1h" {
			return hourly, nil
		}
		return fiveMin, nil
	}

	r1, _ := NewRule(2, "1h", "C2 > C1")
	r2, _ := NewRule(2, "5m", "C2 > C1")
	sig := Signal{Rules: []Rule{r1, r2}}

	matches, err := sig.Scan("", source)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("chain fired at %v, want nowhere when the second rule never holds", matches)
	}
}

// A source that cannot supply a rule's candles fails the scan rather than
// silently finding nothing.
func TestScanReportsASourceFailure(t *testing.T) {
	sig := FVG()
	wantErr := errors.New("exchange unreachable")

	_, err := sig.Scan("1h", func(string) ([]binance.Candle, error) { return nil, wantErr })
	if !errors.Is(err, wantErr) {
		t.Errorf("scan reported %v, want it to wrap %v", err, wantErr)
	}
}

// The current interval fills in wherever a rule leaves its own blank.
func TestScanResolvesTheBlankIntervalToCurrent(t *testing.T) {
	candles := closeSeries(time.Hour, 1, 2)
	rule, _ := NewRule(2, "", "C2 > C1")
	sig := Signal{Rules: []Rule{rule}}

	asked := ""
	_, err := sig.Scan("15m", func(interval string) ([]binance.Candle, error) {
		asked = interval
		return candles, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if asked != "15m" {
		t.Errorf("a blank-interval rule asked the source for %q, want the current interval %q", asked, "15m")
	}
}

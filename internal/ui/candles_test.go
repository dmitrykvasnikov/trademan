package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"

	"github.com/dmitrykvasnikov/trademan/internal/binance"
)

// newTestChart returns a chart holding candles, laid out at a workable size, and
// its renderer.
func newTestChart(t *testing.T, drawn []binance.Candle) (*candleChart, *candleChartRenderer) {
	t.Helper()
	test.NewTempApp(t)

	chart := newCandleChart()
	chart.setCandles(drawn)
	chart.Resize(fyne.NewSize(800, 400))

	renderer, ok := test.WidgetRenderer(chart).(*candleChartRenderer)
	if !ok {
		t.Fatalf("the chart is drawn by %T, not its own renderer", test.WidgetRenderer(chart))
	}
	renderer.Layout(fyne.NewSize(800, 400))

	return chart, renderer
}

// The plot has to hold every wick with room to spare, or the extremes are drawn
// on the frame.
func TestPriceRangeCoversEveryCandleWithRoomToSpare(t *testing.T) {
	drawn := []binance.Candle{
		{Open: 100, High: 120, Low: 95, Close: 110},
		{Open: 110, High: 130, Low: 90, Close: 100},
	}

	low, high := priceRange(drawn)

	if low >= 90 {
		t.Errorf("the plot starts at %v, want it below the lowest wick of 90", low)
	}
	if high <= 130 {
		t.Errorf("the plot ends at %v, want it above the highest wick of 130", high)
	}
}

// A market that has not moved at all would otherwise collapse the plot to a
// line and divide by zero on the way there.
func TestPriceRangeSurvivesAFlatMarket(t *testing.T) {
	low, high := priceRange([]binance.Candle{{Open: 100, High: 100, Low: 100, Close: 100}})

	if !(low < 100 && high > 100) {
		t.Errorf("a flat market gave the plot the range %v–%v, want a band around 100", low, high)
	}
}

func TestScaleMapsPricesOntoThePlot(t *testing.T) {
	s := scale{
		origin: fyne.NewPos(10, 20),
		width:  400,
		height: 200,
		low:    100,
		high:   200,
		count:  4,
	}

	cases := map[float64]float32{
		100: 220, // the low sits on the bottom edge
		200: 20,  // the high sits on the top edge
		150: 120, // and the midpoint halfway between
	}
	for price, want := range cases {
		if got := s.y(price); got != want {
			t.Errorf("%v is drawn at y=%v, want %v", price, got, want)
		}
	}

	if got, want := s.x(0), float32(10+50); got != want {
		t.Errorf("the first candle is centred at x=%v, want %v", got, want)
	}
	if got, want := s.x(3), float32(10+350); got != want {
		t.Errorf("the last candle is centred at x=%v, want %v", got, want)
	}
}

// A flat range would divide by zero, so it draws down the middle instead.
func TestScaleHandlesAFlatRange(t *testing.T) {
	s := scale{origin: fyne.NewPos(0, 0), width: 100, height: 200, low: 50, high: 50, count: 1}

	if got := s.y(50); got != 100 {
		t.Errorf("a flat range draws its price at y=%v, want the middle at 100", got)
	}
}

// Candles have to stay apart at 100 and still be visible at 500.
func TestBodyWidthLeavesAGapAndNeverVanishes(t *testing.T) {
	for _, count := range binance.CandleCounts {
		s := scale{width: 800, count: count}

		width := s.bodyWidth()
		if width < 1 {
			t.Errorf("%d candles are %v wide, want at least a full column", count, width)
		}
		if width >= s.slot() {
			t.Errorf("%d candles are %v wide in a %v slot, want a gap between them", count, width, s.slot())
		}
	}
}

func TestFormatPriceKeepsPricesMeaningfulAtAnySize(t *testing.T) {
	cases := map[float64]string{
		66175.99:  "66175.99",
		1234.5:    "1234.50",
		12.3456:   "12.3456",
		1:         "1.0000",
		0.5:       "0.5000",
		0.0000123: "0.00001230",
		0:         "0.00",
	}

	for price, want := range cases {
		if got := formatPrice(price); got != want {
			t.Errorf("%v is printed as %q, want %q", price, got, want)
		}
	}
}

// The axis is read off the intervals the dropdowns actually offer, so those are
// the spans it is checked against.
func TestTimeLayoutSpellsOutJustEnoughOfTheTimestamp(t *testing.T) {
	cases := []struct {
		name     string
		interval string
		candles  int
		want     string
	}{
		{"seconds within a minute or two", "1s", 100, "15:04:05"},
		{"an afternoon", "5m", 100, "15:04"},
		{"across midnight", "5m", 500, "02 Jan 15:04"},
		{"a few days", "1h", 100, "02 Jan 15:04"},
		{"a few months", "1d", 100, "02 Jan"},
		{"more than a year", "1d", 500, "Jan 2006"},
		{"years", "1M", 100, "Jan 2006"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			step, ok := binance.IntervalDuration(c.interval)
			if !ok {
				t.Fatalf("%q is not an offered interval", c.interval)
			}

			span := step * time.Duration(c.candles)
			if got := timeLayout(span, span/(timeTicks-1)); got != c.want {
				t.Errorf("%d × %s spans %v and is timed as %q, want %q", c.candles, c.interval, span, got, c.want)
			}
		})
	}
}

// The drawing objects are reused between redraws, so growing and shrinking the
// set has to keep what is already there.
func TestFitReusesTheObjectsItAlreadyHas(t *testing.T) {
	built := 0
	build := func() *canvas.Line {
		built++
		return canvas.NewLine(nil)
	}

	lines := fit(nil, 3, build)
	first := lines[0]

	lines = fit(lines, 5, build)
	if built != 5 {
		t.Errorf("growing from 3 to 5 built %d objects, want 5 in total", built)
	}
	if lines[0] != first {
		t.Error("growing the set replaced an object that was already there")
	}

	lines = fit(lines, 2, build)
	if len(lines) != 2 {
		t.Errorf("shrinking left %d objects, want 2", len(lines))
	}
	if built != 5 {
		t.Errorf("shrinking built %d more objects, want none", built-5)
	}
}

func TestChartDrawsAWickAndABodyPerCandle(t *testing.T) {
	drawn := candles(30)
	_, renderer := newTestChart(t, drawn)

	if got := len(renderer.wicks); got != len(drawn) {
		t.Errorf("the chart drew %d wicks, want one per candle (%d)", got, len(drawn))
	}
	if got := len(renderer.bodies); got != len(drawn) {
		t.Errorf("the chart drew %d bodies, want one per candle (%d)", got, len(drawn))
	}
	if got := len(renderer.rows); got != priceTicks {
		t.Errorf("the price scale has %d rows, want %d", got, priceTicks)
	}
	if got := len(renderer.times); got != timeTicks {
		t.Errorf("the time scale has %d labels, want %d", got, timeTicks)
	}
}

// A redraw with fewer candles has to leave the extra ones behind, not draw them
// stacked on the left of the new chart.
func TestChartForgetsTheCandlesItNoLongerHas(t *testing.T) {
	chart, renderer := newTestChart(t, candles(50))
	crowded := len(renderer.Objects())

	chart.setCandles(candles(10))

	if got := len(renderer.bodies); got != 10 {
		t.Errorf("after redrawing 10 candles the chart holds %d bodies, want 10", got)
	}
	if got := len(renderer.Objects()); got >= crowded {
		t.Errorf("redrawing 50 candles as 10 left %d objects on screen, down from %d", got, crowded)
	}
}

// Rises and falls have to be told apart at a glance, which is the whole point
// of a candlestick.
func TestChartColoursRisesAndFallsDifferently(t *testing.T) {
	_, renderer := newTestChart(t, []binance.Candle{
		{Open: 100, High: 110, Low: 95, Close: 108},
		{Open: 108, High: 109, Low: 90, Close: 92},
	})

	ink := currentInk()
	if got := renderer.bodies[0].FillColor; got != ink.up {
		t.Errorf("a rising candle is drawn in %v, want %v", got, ink.up)
	}
	if got := renderer.bodies[1].FillColor; got != ink.down {
		t.Errorf("a falling candle is drawn in %v, want %v", got, ink.down)
	}
}

// A candle that opened and closed at the same price has no body to speak of and
// would otherwise disappear.
func TestChartDrawsAnUnchangedCandleAsALine(t *testing.T) {
	_, renderer := newTestChart(t, []binance.Candle{
		{Open: 100, High: 110, Low: 90, Close: 100},
	})

	if got := renderer.bodies[0].Size().Height; got < 1 {
		t.Errorf("an unchanged candle is %v tall, want at least a full row", got)
	}
}

func TestChartMarksTheLatestClose(t *testing.T) {
	drawn := candles(10)
	_, renderer := newTestChart(t, drawn)

	last := drawn[len(drawn)-1].Close
	if got := renderer.markTag.Text; got != formatPrice(last) {
		t.Errorf("the last close is labelled %q, want %q", got, formatPrice(last))
	}
	if renderer.mark.Position1.Y != renderer.mark.Position2.Y {
		t.Error("the last-close marker is not a horizontal line")
	}
}

// An empty chart draws nothing at all; the area around it shows a message.
func TestChartWithoutCandlesDrawsNothing(t *testing.T) {
	_, renderer := newTestChart(t, nil)

	if got := len(renderer.Objects()); got != 0 {
		t.Errorf("an empty chart drew %d objects, want none", got)
	}
}

// A window can be laid out before it has a size; the chart must not divide by
// it.
func TestChartSurvivesAZeroSizedLayout(t *testing.T) {
	chart, renderer := newTestChart(t, candles(10))

	renderer.Layout(fyne.NewSize(0, 0))
	chart.Refresh()
}

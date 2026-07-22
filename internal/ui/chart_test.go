package ui

import (
	"errors"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
)

func newTestChartArea(t *testing.T) *chartArea {
	t.Helper()
	test.NewTempApp(t)

	return newChartArea()
}

func TestChartAreaStartsByExplainingItself(t *testing.T) {
	area := newTestChartArea(t)

	if area.chart.Visible() {
		t.Error("the chart area starts with a chart on it")
	}
	if got := area.message.Text; got != chartPlaceholder {
		t.Errorf("the chart area says %q, want %q", got, chartPlaceholder)
	}
	if area.card.Subtitle != "" {
		t.Errorf("the heading also says %q, printing the same thing twice", area.card.Subtitle)
	}
}

func TestChartAreaNamesWhatItIsWaitingFor(t *testing.T) {
	area := newTestChartArea(t)

	area.await("BTC/USDT · 1h")

	if !strings.Contains(area.message.Text, "BTC/USDT · 1h") {
		t.Errorf("while loading the area says %q, want it to name the chart", area.message.Text)
	}
	if area.chart.Visible() {
		t.Error("the chart is on screen before its candles arrived")
	}
}

func TestChartAreaShowsCandlesAndDatesThem(t *testing.T) {
	area := newTestChartArea(t)

	area.show("BTC/USDT · 1h", candles(20))

	if !area.chart.Visible() {
		t.Fatal("the chart is hidden even though candles arrived")
	}
	if area.message.Visible() {
		t.Error("the placeholder is still on screen behind the chart")
	}
	if got := area.card.Title; got != "BTC/USDT · 1h" {
		t.Errorf("the chart is headed %q, want %q", got, "BTC/USDT · 1h")
	}
	if got := area.card.Subtitle; !strings.Contains(got, "updated") || !strings.Contains(got, "20 candles") {
		t.Errorf("the chart is subtitled %q, want the count and the time it was updated", got)
	}
}

// Candles cannot be plotted without candles; an empty set must leave whatever
// is on screen alone rather than blanking the chart.
func TestChartAreaIgnoresAnEmptySet(t *testing.T) {
	area := newTestChartArea(t)

	area.show("BTC/USDT · 1h", nil)

	if area.chart.Visible() {
		t.Error("an empty set of candles was drawn as a chart")
	}
}

// A refresh can fail on a blip. Stale candles with a warning on them beat an
// empty panel, so the chart has to survive the failure.
func TestChartAreaKeepsALiveChartThroughAFailure(t *testing.T) {
	area := newTestChartArea(t)
	area.show("BTC/USDT · 1h", candles(20))

	area.fail("BTC/USDT · 1h", errors.New("connection reset"))

	if !area.chart.Visible() {
		t.Error("a failed refresh threw away the chart that was already drawn")
	}
	if got := area.card.Subtitle; !strings.Contains(got, "connection reset") {
		t.Errorf("the chart is subtitled %q, want the failure named", got)
	}
}

// With nothing to keep, the failure is all there is to show.
func TestChartAreaReportsAFailureWithNothingDrawn(t *testing.T) {
	area := newTestChartArea(t)
	area.await("BTC/USDT · 1h")

	area.fail("BTC/USDT · 1h", errors.New("no route to host"))

	if area.chart.Visible() {
		t.Error("a chart appeared after a request that never returned candles")
	}
	if got := area.message.Text; !strings.Contains(got, "no route to host") {
		t.Errorf("the chart area says %q, want the failure named", got)
	}
}

func TestChartAreaClearsBackToThePlaceholder(t *testing.T) {
	area := newTestChartArea(t)
	area.show("BTC/USDT · 1h", candles(20))

	area.clear()

	if area.chart.Visible() {
		t.Error("the chart stayed on screen after the area was cleared")
	}
	if got := area.message.Text; got != chartPlaceholder {
		t.Errorf("the cleared area says %q, want %q", got, chartPlaceholder)
	}
	if area.card.Title != "" {
		t.Errorf("the cleared area is still headed %q", area.card.Title)
	}
}

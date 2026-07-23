package ui

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/dmitrykvasnikov/trademan/internal/binance"
)

// tabLog records what a tab asked its surroundings for: the names it gave
// itself, and the feeds it started.
type tabLog struct {
	titles []string
	feeds  []feed
}

func (l *tabLog) title() string {
	if len(l.titles) == 0 {
		return ""
	}
	return l.titles[len(l.titles)-1]
}

// newTestTab builds a tab against client. The feed is stubbed out: it would
// otherwise run on its own goroutine and race whatever the test is checking,
// and the tests that want candles drawn call draw directly instead.
func newTestTab(t *testing.T, client *binance.Client) (*tabView, *tabLog) {
	t.Helper()
	test.NewTempApp(t)

	log := &tabLog{}
	view := newTabView(client, testCatalog(testSymbols), func(title string) {
		log.titles = append(log.titles, title)
	})
	view.launch = func(_ context.Context, f feed) { log.feeds = append(log.feeds, f) }

	return view, log
}

// selectAll fills in every dropdown, which is what puts a chart on screen.
func selectAll(view *tabView, coin, interval, candles string) {
	view.coin.SetSelected(coin)
	view.interval.SetSelected(interval)
	view.candles.SetSelected(candles)
}

func TestTabStartsWithEveryDropdownEmpty(t *testing.T) {
	view, _ := newTestTab(t, offlineClient())

	for name, selector := range map[string]*widget.Select{
		"Coin":          view.coin,
		"Interval":      view.interval,
		"No of candles": view.candles,
	} {
		if selector.Selected != "" {
			t.Errorf("%s starts on %q, want it empty", name, selector.Selected)
		}
		if selector.PlaceHolder != emptySelection {
			t.Errorf("%s reads %q while empty, want %q", name, selector.PlaceHolder, emptySelection)
		}
	}
	if view.chart.chart.Visible() {
		t.Error("a fresh tab shows a chart before anything is selected")
	}
}

func TestTabOffersTheListedIntervalsAndCounts(t *testing.T) {
	view, _ := newTestTab(t, offlineClient())

	if got := strings.Join(view.interval.Options, " "); got != strings.Join(binance.Intervals, " ") {
		t.Errorf("Interval offers %q, want %q", got, strings.Join(binance.Intervals, " "))
	}
	if got := strings.Join(view.candles.Options, " "); got != "100 200 300 500" {
		t.Errorf("No of candles offers %q, want %q", got, "100 200 300 500")
	}
}

func TestTabOffersTheCoinsFromTheCatalog(t *testing.T) {
	view, _ := newTestTab(t, offlineClient())

	if got := strings.Join(view.coin.Options, " "); got != "BTC/USDT ETH/USDT SOL/USDT" {
		t.Errorf("Coin offers %q, want the catalog's coins", got)
	}
}

// Nothing is charted until all three dropdowns are filled in.
func TestSelectionNeedsAllThreeDropdowns(t *testing.T) {
	cases := []struct {
		name                    string
		coin, interval, candles string
	}{
		{"nothing chosen", "", "", ""},
		{"no coin", "", "1h", "100"},
		{"no interval", "BTC/USDT", "", "100"},
		{"no candle count", "BTC/USDT", "1h", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			view, _ := newTestTab(t, offlineClient())
			selectAll(view, c.coin, c.interval, c.candles)

			if _, ok := view.selection(); ok {
				t.Error("an incomplete selection was accepted as a chart")
			}
		})
	}
}

func TestSelectionResolvesToTheChosenChart(t *testing.T) {
	view, _ := newTestTab(t, offlineClient())

	selectAll(view, "ETH/USDT", "15m", "300")

	f, ok := view.selection()
	if !ok {
		t.Fatal("a complete selection was not accepted as a chart")
	}
	if got := f.symbol.Pair(); got != "ETHUSDT" {
		t.Errorf("selection charts %q, want %q", got, "ETHUSDT")
	}
	if f.interval != "15m" || f.candles != 300 {
		t.Errorf("selection is %s × %d candles, want 15m × 300", f.interval, f.candles)
	}
	if got := f.title(); got != "ETH/USDT · 15m" {
		t.Errorf("chart is titled %q, want %q", got, "ETH/USDT · 15m")
	}
}

// A tab keeps its name until it has something to show, and takes the chart's
// name once it does.
func TestTabIsNamedAfterItsChart(t *testing.T) {
	view, log := newTestTab(t, offlineClient())

	selectAll(view, "BTC/USDT", "1h", "100")

	if len(log.titles) == 0 {
		t.Fatal("the tab was never renamed")
	}
	if got := log.title(); got != "BTC/USDT · 1h" {
		t.Errorf("the tab ended up named %q, want %q", got, "BTC/USDT · 1h")
	}
	for _, title := range log.titles[:len(log.titles)-1] {
		if title != newTabTitle {
			t.Errorf("the tab was named %q before its selection was complete, want %q", title, newTabTitle)
		}
	}
}

// A complete selection starts exactly one feed, and changing a dropdown
// replaces it rather than adding a second one alongside.
func TestChangingADropdownReplacesTheFeed(t *testing.T) {
	view, log := newTestTab(t, offlineClient())
	selectAll(view, "BTC/USDT", "1h", "100")

	view.coin.SetSelected("SOL/USDT")

	if len(log.feeds) != 2 {
		t.Fatalf("%d feeds were started, want one per complete selection", len(log.feeds))
	}
	if got := log.feeds[1].symbol.Pair(); got != "SOLUSDT" {
		t.Errorf("the replacement feed charts %q, want %q", got, "SOLUSDT")
	}
	if view.stop == nil {
		t.Error("the replacement feed cannot be stopped")
	}
}

// Emptying a dropdown again retires the chart along with the name it gave the
// tab.
func TestClearingASelectionRetiresTheChart(t *testing.T) {
	view, log := newTestTab(t, klineServer(t, 10))
	selectAll(view, "BTC/USDT", "1h", "100")

	f, _ := view.selection()
	view.draw(context.Background(), f)
	view.interval.ClearSelected()

	if view.chart.chart.Visible() {
		t.Error("the chart stayed on screen after the interval was cleared")
	}
	if got := log.title(); got != newTabTitle {
		t.Errorf("the tab is named %q, want %q once nothing is charted", got, newTabTitle)
	}
	if view.stop != nil {
		t.Error("the feed is still running with nothing selected to chart")
	}
}

func TestDrawPutsCandlesOnScreen(t *testing.T) {
	view, _ := newTestTab(t, klineServer(t, 12))
	selectAll(view, "BTC/USDT", "1h", "100")
	defer view.close()

	f, ok := view.selection()
	if !ok {
		t.Fatal("a complete selection was not accepted as a chart")
	}
	view.draw(context.Background(), f)

	if !view.chart.chart.Visible() {
		t.Fatal("the chart is still hidden after candles arrived")
	}
	if got := len(view.chart.chart.candles); got != 12 {
		t.Errorf("the chart holds %d candles, want the 12 that arrived", got)
	}
	if got := view.chart.card.Title; got != f.title() {
		t.Errorf("the chart is headed %q, want %q", got, f.title())
	}
}

func TestDrawReportsAFailedRequest(t *testing.T) {
	view, _ := newTestTab(t, failingServer(t, "Invalid symbol."))
	selectAll(view, "BTC/USDT", "1h", "100")
	defer view.close()

	f, _ := view.selection()
	view.draw(context.Background(), f)

	if view.chart.chart.Visible() {
		t.Error("a failed request left a chart on screen")
	}
	if got := view.chart.message.Text; !strings.Contains(got, "Invalid symbol.") {
		t.Errorf("the chart area says %q, want it to carry the exchange's explanation", got)
	}
}

// Candles asked for before the user changed a dropdown must not land on the
// chart that replaced them.
func TestDrawDiscardsCandlesFromARetiredFeed(t *testing.T) {
	view, _ := newTestTab(t, klineServer(t, 12))
	selectAll(view, "BTC/USDT", "1h", "100")
	defer view.close()

	f, _ := view.selection()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	view.draw(ctx, f)

	if view.chart.chart.Visible() {
		t.Error("a retired feed drew its candles anyway")
	}
}

// A live chart re-reads its candles at the pace of its own interval, but never
// so fast that it floods the exchange nor so slowly that it looks frozen.
func TestRefreshPeriodFollowsTheInterval(t *testing.T) {
	cases := map[string]time.Duration{
		"1s":       time.Second,
		"1m":       time.Minute,
		"3m":       time.Minute,
		"1d":       time.Minute,
		"1M":       time.Minute,
		"nonsense": time.Minute,
	}

	for interval, want := range cases {
		if got := refreshPeriod(interval); got != want {
			t.Errorf("a %s chart refreshes every %v, want %v", interval, got, want)
		}
	}
}

// 'r' runs the FVG signal over the candles on screen and circles the ones that
// complete a gap.
func TestRunSignalMarksTheGaps(t *testing.T) {
	view, _ := newTestTab(t, offlineClient())
	view.drawn = gappyCandles()

	view.runSignal()

	if !view.signalOn {
		t.Error("running the signal did not turn it on")
	}
	if got := view.chart.chart.marks; !reflect.DeepEqual(got, []int{3, 7}) {
		t.Errorf("the signal marked candles %v, want the gap-completing 3 and 7", got)
	}
}

// 'u' turns the signal off and takes its marks with it.
func TestClearSignalRemovesTheMarks(t *testing.T) {
	view, _ := newTestTab(t, offlineClient())
	view.drawn = gappyCandles()
	view.runSignal()

	view.clearSignal()

	if view.signalOn {
		t.Error("clearing the signal left it on")
	}
	if got := view.chart.chart.marks; got != nil {
		t.Errorf("clearing the signal left marks %v, want none", got)
	}
}

// A signal left on keeps marking as fresh candles arrive, so a live chart does
// not go blank after the first draw.
func TestSignalReMarksThePicksUpNewCandles(t *testing.T) {
	view, _ := newTestTab(t, offlineClient())

	view.runSignal() // on, but nothing is on screen yet
	if got := len(view.chart.chart.marks); got != 0 {
		t.Errorf("the signal marked %d candles with none on screen, want 0", got)
	}

	view.drawn = gappyCandles()
	view.applyMarks() // stands in for the next refresh drawing new candles

	if got := view.chart.chart.marks; !reflect.DeepEqual(got, []int{3, 7}) {
		t.Errorf("after new candles the signal marks %v, want 3 and 7", got)
	}
}

// A running signal follows a change of timeframe: the new candles are scanned,
// not left with the marks from the old interval.
func TestSignalUpdatesWhenTheTimeframeChanges(t *testing.T) {
	view, _ := newTestTab(t, candleServer(t, gappyCandles()))
	selectAll(view, "BTC/USDT", "1h", "100")
	view.runSignal()

	view.interval.SetSelected("5m") // still a complete selection: the feed relaunches
	f, ok := view.selection()
	if !ok {
		t.Fatal("switching the timeframe left an incomplete selection")
	}
	view.draw(context.Background(), f) // the relaunched feed's first draw

	if !view.signalOn {
		t.Error("switching the timeframe turned the signal off")
	}
	if got := view.chart.chart.marks; !reflect.DeepEqual(got, []int{3, 7}) {
		t.Errorf("after switching timeframe the signal marked %v, want 3 and 7", got)
	}
}

// It follows a change of candle count the same way.
func TestSignalUpdatesWhenTheCandleCountChanges(t *testing.T) {
	view, _ := newTestTab(t, candleServer(t, gappyCandles()))
	selectAll(view, "BTC/USDT", "1h", "100")
	view.runSignal()

	view.candles.SetSelected("300")
	f, _ := view.selection()
	view.draw(context.Background(), f)

	if got := view.chart.chart.marks; !reflect.DeepEqual(got, []int{3, 7}) {
		t.Errorf("after changing the candle count the signal marked %v, want 3 and 7", got)
	}
}

// A live chart re-reads its candles each tick; when a new one arrives the set
// shifts, and the signal re-marks the shifted set instead of pointing at where
// the gaps used to be.
func TestSignalFollowsTheLiveCandles(t *testing.T) {
	view, _ := newTestTab(t, offlineClient())
	view.signalOn = true

	view.drawn = gappyCandles()
	view.applyMarks()
	if got := view.chart.chart.marks; !reflect.DeepEqual(got, []int{3, 7}) {
		t.Fatalf("the first frame marked %v, want 3 and 7", got)
	}

	// The oldest candle drops off the front, so every gap shifts one to the left.
	view.drawn = gappyCandles()[1:]
	view.applyMarks()
	if got := view.chart.chart.marks; !reflect.DeepEqual(got, []int{2, 6}) {
		t.Errorf("after the set shifted the marks are %v, want them at 2 and 6", got)
	}
}

// Changing a dropdown drops the marks along with the candles they sat on, so a
// stale mark cannot survive onto a different chart.
func TestReloadClearsTheMarks(t *testing.T) {
	view, _ := newTestTab(t, offlineClient())
	view.drawn = gappyCandles()
	view.runSignal()

	view.interval.SetSelected("1h") // an incomplete selection retires the chart

	if got := view.chart.chart.marks; got != nil {
		t.Errorf("changing a dropdown left marks %v on screen, want none", got)
	}
}

// Closing a tab has to stop its feed, or a closed tab keeps polling forever.
func TestCloseStopsTheFeed(t *testing.T) {
	view, _ := newTestTab(t, klineServer(t, 5))
	selectAll(view, "BTC/USDT", "1h", "100")

	if view.stop == nil {
		t.Fatal("a complete selection did not start a feed")
	}

	view.close()

	if view.stop != nil {
		t.Error("the feed is still running after the tab was closed")
	}
}

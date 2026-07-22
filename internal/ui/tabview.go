package ui

import (
	"context"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/dmitrykvasnikov/trademan/internal/binance"
)

// emptySelection is what a dropdown shows before anything is picked in it.
const emptySelection = "-"

// tabView is the content of a single tab: the three chart selectors in a row on
// top, and the chart area filling everything below them. It also owns the feed
// keeping that chart live, starting one as soon as all three selectors are
// filled in and replacing it whenever one of them changes.
type tabView struct {
	client  *binance.Client
	coins   *coinCatalog
	onTitle func(string)

	coin     *widget.Select
	interval *widget.Select
	candles  *widget.Select
	chart    *chartArea

	// symbols backs the Coin dropdown; the selected index points into it.
	symbols []binance.Symbol

	// stop ends the feed currently drawing into the chart, if there is one.
	stop context.CancelFunc

	// launch starts the feed behind a selection. It is a field because the feed
	// runs on its own goroutine: keeping the hand-off in one replaceable place
	// lets the tests step a feed by hand instead of racing a live one.
	launch func(context.Context, feed)
}

// newTabView builds a tab with all three dropdowns empty, which is the state the
// feature calls for: nothing is charted until the user has picked all three.
func newTabView(client *binance.Client, coins *coinCatalog, onTitle func(string)) *tabView {
	v := &tabView{
		client:   client,
		coins:    coins,
		onTitle:  onTitle,
		coin:     newSelector(nil),
		interval: newSelector(binance.Intervals),
		candles:  newSelector(candleCounts()),
		chart:    newChartArea(),
	}

	v.launch = func(ctx context.Context, f feed) { go v.stream(ctx, f) }

	reload := func(string) { v.reload() }
	v.coin.OnChanged, v.interval.OnChanged, v.candles.OnChanged = reload, reload, reload

	// The coin list is ranked by turnover, so it has to be fetched. Until it
	// lands the dropdown stays empty rather than offering a guess.
	v.coins.load(v.setSymbols)

	return v
}

func (v *tabView) view() fyne.CanvasObject {
	// Equal columns keep the three selectors evenly spread across the tab.
	selectors := container.NewGridWithColumns(3,
		labelled("Coin", v.coin),
		labelled("Interval", v.interval),
		labelled("No of candles", v.candles),
	)
	return container.NewBorder(selectors, nil, nil, nil, v.chart.view())
}

// close stops this tab's feed, so a closed tab stops polling the exchange.
func (v *tabView) close() {
	if v.stop != nil {
		v.stop()
		v.stop = nil
	}
}

// setSymbols fills the Coin dropdown once the ranking arrives.
func (v *tabView) setSymbols(symbols []binance.Symbol) {
	v.symbols = symbols

	labels := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		labels = append(labels, symbol.Label())
	}
	v.coin.SetOptions(labels)
}

// reload restarts the chart after a dropdown changes: the running feed is
// stopped, and a new one starts only once all three selections are filled in.
func (v *tabView) reload() {
	v.close()

	f, ok := v.selection()
	if !ok {
		v.chart.clear()
		v.onTitle(newTabTitle)
		return
	}

	v.onTitle(f.title())
	v.chart.await(f.title())

	ctx, cancel := context.WithCancel(context.Background())
	v.stop = cancel
	v.launch(ctx, f)
}

// selection reports the chart the three dropdowns describe. The second result
// is false while any of them is still empty, which is where a tab starts.
func (v *tabView) selection() (feed, bool) {
	coin := v.coin.SelectedIndex()
	if coin < 0 || coin >= len(v.symbols) || v.interval.Selected == "" {
		return feed{}, false
	}

	count, err := strconv.Atoi(v.candles.Selected)
	if err != nil {
		return feed{}, false
	}

	return feed{symbol: v.symbols[coin], interval: v.interval.Selected, candles: count}, true
}

// stream keeps the chart current: it draws once straight away, then again every
// refresh period until the feed is stopped by another dropdown change or by the
// tab closing.
func (v *tabView) stream(ctx context.Context, f feed) {
	tick := time.NewTicker(refreshPeriod(f.interval))
	defer tick.Stop()

	for {
		v.draw(ctx, f)

		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// draw fetches one set of candles and puts them on screen. The feed is checked
// again after the request, because a dropdown may have changed while it was in
// flight and the answer to the old question must not land on the new chart.
func (v *tabView) draw(ctx context.Context, f feed) {
	candles, err := v.client.Klines(ctx, f.symbol.Pair(), f.interval, f.candles)
	if ctx.Err() != nil {
		return
	}

	fyne.Do(func() {
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			v.chart.fail(f.title(), err)
			return
		}
		v.chart.show(f.title(), candles)
	})
}

// feed is everything needed to draw one chart.
type feed struct {
	symbol   binance.Symbol
	interval string
	candles  int
}

// title names the chart, and with it the tab showing it.
func (f feed) title() string { return f.symbol.Label() + " · " + f.interval }

// refreshPeriod is how often a live chart re-reads its candles. It follows the
// selected interval — a five-minute chart redraws every five minutes — but is
// clamped at both ends: never faster than once a second, so a 1s chart cannot
// hammer the API, and never slower than a minute, so the candle still forming
// on a daily chart keeps moving while it is watched.
func refreshPeriod(interval string) time.Duration {
	const (
		fastest = time.Second
		slowest = time.Minute
	)

	span, ok := binance.IntervalDuration(interval)
	if !ok {
		return slowest
	}
	return min(max(span, fastest), slowest)
}

// newSelector creates a dropdown holding options and picking none of them. It
// shows a dash until something is picked: the caption above already names the
// dropdown, so repeating the name inside it would only make an empty selector
// look like a filled one.
func newSelector(options []string) *widget.Select {
	selector := widget.NewSelect(options, nil)
	selector.PlaceHolder = emptySelection
	return selector
}

// candleCounts renders the offered chart depths as dropdown entries.
func candleCounts() []string {
	labels := make([]string, 0, len(binance.CandleCounts))
	for _, count := range binance.CandleCounts {
		labels = append(labels, strconv.Itoa(count))
	}
	return labels
}

// labelled stacks a caption above a field.
func labelled(name string, field fyne.CanvasObject) fyne.CanvasObject {
	caption := widget.NewLabelWithStyle(name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	return container.NewVBox(caption, field)
}

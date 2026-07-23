package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/dmitrykvasnikov/trademan/internal/binance"
)

// chartPlaceholder tells the user what the empty chart area is waiting for.
const chartPlaceholder = "Pick a coin, an interval and a number of candles to draw a chart"

// chartArea fills the space below a tab's selectors. It shows a message while
// there is nothing to plot — before the selection is complete, and while the
// first candles are on their way — and the chart itself once they arrive. The
// card around it carries the heading: what is being charted, and how fresh it
// is.
type chartArea struct {
	card    *widget.Card
	message *widget.Label
	chart   *candleChart

	// subtitle is the data line under the title — last price, candle count,
	// update time; note is the signal status shown beside it. They are kept
	// apart so a signal can update its half without redrawing the other.
	subtitle string
	note     string
}

func newChartArea() *chartArea {
	c := &chartArea{
		message: widget.NewLabelWithStyle(chartPlaceholder, fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
		chart:   newCandleChart(),
	}
	c.message.Wrapping = fyne.TextWrapWord
	c.chart.Hide()

	// Both live in the same space and take turns being visible, so the area
	// keeps its size whichever of them is on screen.
	c.card = widget.NewCard("", "", container.NewStack(
		container.NewCenter(c.message),
		c.chart,
	))
	return c
}

func (c *chartArea) view() fyne.CanvasObject { return c.card }

// clear returns the area to the state a freshly opened tab is in.
func (c *chartArea) clear() {
	c.say("", chartPlaceholder)
}

// await reports which chart is being fetched, so the wait for the first candles
// is not a blank panel.
func (c *chartArea) await(title string) {
	c.say(title, "Loading "+title+" …")
}

// show puts candles on screen and dates them, since a live chart that quietly
// stopped updating looks exactly like one that is up to date.
func (c *chartArea) show(title string, candles []binance.Candle) {
	if len(candles) == 0 {
		return
	}

	last := candles[len(candles)-1]
	c.card.SetTitle(title)
	c.subtitle = fmt.Sprintf("last %s · %d candles · updated %s",
		formatPrice(last.Close), len(candles), time.Now().Format("15:04:05"))

	c.chart.setCandles(candles)
	c.message.Hide()
	c.chart.Show()
	c.render()
}

// setNote records the signal status shown beside the data line and repaints the
// subtitle. It tells a chart with no matches apart from a keystroke that never
// landed. An empty note removes it.
func (c *chartArea) setNote(note string) {
	c.note = note
	c.render()
}

// render composes the card subtitle from the data line and, when a chart is on
// screen, the active signal's note beside it.
func (c *chartArea) render() {
	subtitle := c.subtitle
	if c.note != "" && c.chart.Visible() {
		subtitle += " · " + c.note
	}
	c.card.SetSubTitle(subtitle)
}

// fail reports a request that did not come back. A chart already on screen is
// kept and marked stale rather than thrown away: a refresh can fail on a blip,
// and old candles with a warning on them beat an empty panel.
func (c *chartArea) fail(title string, err error) {
	if c.chart.Visible() {
		c.subtitle = "not updating — " + err.Error()
		c.render()
		return
	}
	c.say(title, "Could not load "+title+" — "+err.Error())
}

// say hides the chart and puts a message in its place. The subtitle goes with
// it: the message is the whole of what the area has to report, and a subtitle
// alongside would only say the same thing twice.
func (c *chartArea) say(title, message string) {
	c.card.SetTitle(title)
	c.subtitle, c.note = "", ""

	c.message.SetText(message)
	c.chart.Hide()
	c.message.Show()
	c.render()
}

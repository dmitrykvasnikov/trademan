package ui

import (
	"image/color"
	"math"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/dmitrykvasnikov/trademan/internal/binance"
)

// Chart geometry, in the unscaled pixels Fyne lays widgets out in.
const (
	priceAxisWidth = 76 // room to the right of the plot for a price
	timeAxisHeight = 20 // room below the plot for a timestamp
	plotPadding    = 6
	axisTextSize   = 11
	priceTicks     = 5 // horizontal grid lines, counting both edges
	timeTicks      = 6 // vertical grid lines, counting both edges
	bodyFill       = 0.7
)

// candleChart draws a candlestick chart: one candle per interval, a price scale
// down the right-hand side and a time scale along the bottom.
//
// It is a plain widget with no state beyond the candles, so putting a new set
// of candles in and refreshing is the whole update path — which is what the
// live feed does once per tick.
type candleChart struct {
	widget.BaseWidget

	candles []binance.Candle
	// marks are the indices of candles a signal fired on; each is circled.
	marks []int
}

func newCandleChart() *candleChart {
	c := &candleChart{}
	c.ExtendBaseWidget(c)
	return c
}

// setCandles replaces what the chart shows and redraws it.
func (c *candleChart) setCandles(candles []binance.Candle) {
	c.candles = candles
	c.Refresh()
}

// setMarks circles the given candles — the ones a signal fired on — and redraws.
// Passing nil clears the marks.
func (c *candleChart) setMarks(marks []int) {
	c.marks = marks
	c.Refresh()
}

func (c *candleChart) CreateRenderer() fyne.WidgetRenderer {
	r := &candleChartRenderer{chart: c}
	r.build()
	return r
}

// candleChartRenderer holds the drawing objects. They are kept between redraws
// and reshaped in place: a 500-candle chart on a one-second interval would
// otherwise allocate a thousand shapes every second.
type candleChartRenderer struct {
	chart *candleChart

	rows    []*canvas.Line      // horizontal grid lines, one per price tick
	prices  []*canvas.Text      // the price at each row, on the right axis
	columns []*canvas.Line      // vertical grid lines, one per time tick
	times   []*canvas.Text      // the time at each column, on the bottom axis
	wicks   []*canvas.Line      // the high-to-low range of each candle
	bodies  []*canvas.Rectangle // the open-to-close range of each candle
	marks   []*canvas.Circle    // a ring around each candle a signal fired on
	mark    *canvas.Line        // the latest close, drawn across the plot
	markTag *canvas.Text        // and its price, on the right axis

	objects []fyne.CanvasObject
	size    fyne.Size
}

func (r *candleChartRenderer) MinSize() fyne.Size { return fyne.NewSize(280, 180) }

func (r *candleChartRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *candleChartRenderer) Destroy() {}

func (r *candleChartRenderer) Refresh() {
	r.build()
	r.Layout(r.size)
	canvas.Refresh(r.chart)
}

// build brings the set of drawing objects in line with the current candles and
// the current theme, then rebuilds the draw order: grid first, then candles on
// top of it, then the last-price marker, and the axis text above everything.
func (r *candleChartRenderer) build() {
	ink := currentInk()
	candles := r.chart.candles

	r.rows = fit(r.rows, priceTicks, func() *canvas.Line { return canvas.NewLine(ink.grid) })
	r.columns = fit(r.columns, timeTicks, func() *canvas.Line { return canvas.NewLine(ink.grid) })
	r.prices = fit(r.prices, priceTicks, func() *canvas.Text { return axisLabel(fyne.TextAlignLeading) })
	r.times = fit(r.times, timeTicks, func() *canvas.Text { return axisLabel(fyne.TextAlignCenter) })
	r.wicks = fit(r.wicks, len(candles), func() *canvas.Line { return canvas.NewLine(ink.up) })
	r.bodies = fit(r.bodies, len(candles), func() *canvas.Rectangle { return canvas.NewRectangle(ink.up) })
	r.marks = fit(r.marks, len(r.chart.marks), func() *canvas.Circle {
		ring := canvas.NewCircle(color.Transparent)
		ring.StrokeWidth = 2
		return ring
	})

	if r.mark == nil {
		r.mark = canvas.NewLine(ink.mark)
		r.markTag = axisLabel(fyne.TextAlignLeading)
	}
	r.mark.StrokeColor, r.markTag.Color = ink.mark, ink.mark

	for _, line := range r.rows {
		line.StrokeColor = ink.grid
	}
	for _, line := range r.columns {
		line.StrokeColor = ink.grid
	}
	for _, text := range r.prices {
		text.Color = ink.text
	}
	for _, text := range r.times {
		text.Color = ink.text
	}
	for i, candle := range candles {
		colour := ink.down
		if candle.Rising() {
			colour = ink.up
		}
		r.wicks[i].StrokeColor = colour
		r.bodies[i].FillColor = colour
	}
	for _, ring := range r.marks {
		ring.StrokeColor = ink.mark
	}

	// An empty chart draws nothing at all: the area shows a message instead.
	if len(candles) == 0 {
		r.objects = nil
		return
	}

	r.objects = r.objects[:0]
	for _, line := range r.rows {
		r.objects = append(r.objects, line)
	}
	for _, line := range r.columns {
		r.objects = append(r.objects, line)
	}
	for i := range candles {
		r.objects = append(r.objects, r.wicks[i], r.bodies[i])
	}
	// Rings sit above the candles they enclose so a signal reads clearly.
	for _, ring := range r.marks {
		r.objects = append(r.objects, ring)
	}
	r.objects = append(r.objects, r.mark, r.markTag)
	for _, text := range r.prices {
		r.objects = append(r.objects, text)
	}
	for _, text := range r.times {
		r.objects = append(r.objects, text)
	}
}

func (r *candleChartRenderer) Layout(size fyne.Size) {
	r.size = size

	candles := r.chart.candles
	if len(candles) == 0 || size.IsZero() {
		return
	}

	low, high := priceRange(candles)
	s := scale{
		origin: fyne.NewPos(plotPadding, plotPadding),
		width:  max(size.Width-priceAxisWidth-plotPadding, 1),
		height: max(size.Height-timeAxisHeight-2*plotPadding, 1),
		low:    low,
		high:   high,
		count:  len(candles),
	}

	r.layoutPriceAxis(s)
	r.layoutTimeAxis(s, candles)
	r.layoutCandles(s, candles)
	r.layoutMarks(s, candles)
	r.layoutMark(s, candles[len(candles)-1].Close)
}

// layoutMarks rings each candle a signal fired on. The ring is centred on the
// candle's midpoint and widens with the space each candle has, down to a floor
// so it stays visible when 500 candles are packed in, and up to a ceiling so it
// does not swallow the chart when only a few are.
func (r *candleChartRenderer) layoutMarks(s scale, candles []binance.Candle) {
	radius := clamp(s.slot()*0.9, 8, 22)

	for i, index := range r.chart.marks {
		ring := r.marks[i]
		// A mark left over from a longer chart must not be drawn onto a candle
		// that no longer exists; park it off any candle instead.
		if index < 0 || index >= len(candles) {
			ring.Position1, ring.Position2 = fyne.NewPos(0, 0), fyne.NewPos(0, 0)
			continue
		}

		candle := candles[index]
		x := s.x(index)
		y := s.y((candle.High + candle.Low) / 2)
		ring.Position1 = fyne.NewPos(x-radius, y-radius)
		ring.Position2 = fyne.NewPos(x+radius, y+radius)
	}
}

// layoutPriceAxis spreads the grid rows evenly over the price range and prints
// each one's price just past the right edge of the plot.
func (r *candleChartRenderer) layoutPriceAxis(s scale) {
	right := s.origin.X + s.width

	for i, row := range r.rows {
		price := s.low + (s.high-s.low)*float64(i)/float64(priceTicks-1)
		y := s.y(price)

		row.Position1 = fyne.NewPos(s.origin.X, y)
		row.Position2 = fyne.NewPos(right, y)

		label := r.prices[i]
		label.Text = formatPrice(price)
		label.Resize(label.MinSize())
		label.Move(fyne.NewPos(right+plotPadding, y-label.MinSize().Height/2))
	}
}

// layoutTimeAxis puts a column at evenly spaced candles and prints each one's
// open time below the plot.
func (r *candleChartRenderer) layoutTimeAxis(s scale, candles []binance.Candle) {
	bottom := s.origin.Y + s.height
	span := candles[len(candles)-1].CloseTime.Sub(candles[0].OpenTime)
	layout := timeLayout(span, span/(timeTicks-1))

	for i, column := range r.columns {
		index := int(math.Round(float64(i) * float64(len(candles)-1) / float64(timeTicks-1)))
		x := s.x(index)

		column.Position1 = fyne.NewPos(x, s.origin.Y)
		column.Position2 = fyne.NewPos(x, bottom)

		label := r.times[i]
		label.Text = candles[index].OpenTime.Local().Format(layout)
		width := label.MinSize().Width
		label.Resize(label.MinSize())
		// Clamped so the first and last labels stay inside the widget rather
		// than hanging off its edges.
		label.Move(fyne.NewPos(clamp(x-width/2, 0, s.origin.X+s.width), bottom+plotPadding/2))
	}
}

// layoutCandles draws each candle as a wick spanning high to low with the body
// spanning open to close on top of it.
func (r *candleChartRenderer) layoutCandles(s scale, candles []binance.Candle) {
	width := s.bodyWidth()

	for i, candle := range candles {
		x := s.x(i)

		wick := r.wicks[i]
		wick.Position1 = fyne.NewPos(x, s.y(candle.High))
		wick.Position2 = fyne.NewPos(x, s.y(candle.Low))

		top := s.y(math.Max(candle.Open, candle.Close))
		bottom := s.y(math.Min(candle.Open, candle.Close))

		body := r.bodies[i]
		body.Move(fyne.NewPos(x-width/2, top))
		// A candle that opened and closed at the same price has no body at all,
		// so it is drawn as the thin line traders read it as.
		body.Resize(fyne.NewSize(width, max(bottom-top, 1)))
	}
}

// layoutMark rules a line across the plot at the most recent close, which is
// the one price on a live chart that keeps moving.
func (r *candleChartRenderer) layoutMark(s scale, price float64) {
	y := s.y(price)
	right := s.origin.X + s.width

	r.mark.Position1 = fyne.NewPos(s.origin.X, y)
	r.mark.Position2 = fyne.NewPos(right, y)

	r.markTag.Text = formatPrice(price)
	r.markTag.TextStyle = fyne.TextStyle{Bold: true}
	r.markTag.Resize(r.markTag.MinSize())
	r.markTag.Move(fyne.NewPos(right+plotPadding, y-r.markTag.MinSize().Height/2))
}

// scale maps prices and candle positions onto the plot rectangle.
type scale struct {
	origin fyne.Position // top-left corner of the plot
	width  float32
	height float32
	low    float64
	high   float64
	count  int
}

// y is the vertical position of a price: low sits on the bottom edge, high on
// the top one.
func (s scale) y(price float64) float32 {
	if s.high <= s.low {
		return s.origin.Y + s.height/2
	}
	above := (price - s.low) / (s.high - s.low)
	return s.origin.Y + s.height - float32(above)*s.height
}

// x is the horizontal centre of the candle at index i.
func (s scale) x(i int) float32 { return s.origin.X + s.slot()*(float32(i)+0.5) }

// slot is the width one candle has to itself.
func (s scale) slot() float32 {
	if s.count == 0 {
		return s.width
	}
	return s.width / float32(s.count)
}

// bodyWidth leaves a gap between neighbouring candles, narrowing to a single
// column once 500 of them are packed into the plot.
func (s scale) bodyWidth() float32 { return max(s.slot()*bodyFill, 1) }

// priceRange is the span the plot covers: the extremes of the candles plus a
// margin, so the highest wick does not touch the top edge.
func priceRange(candles []binance.Candle) (low, high float64) {
	low, high = candles[0].Low, candles[0].High
	for _, candle := range candles[1:] {
		low = math.Min(low, candle.Low)
		high = math.Max(high, candle.High)
	}

	span := high - low
	if span <= 0 {
		// A flat range would collapse the plot to a line and divide by zero on
		// the way; give it a band around the price to sit in instead.
		span = math.Max(math.Abs(high)*0.01, 1)
	}

	margin := span * 0.05
	return low - margin, high + margin
}

// chartInk is the palette the chart draws with, taken from the active theme so
// the plot follows the light/dark switch along with the rest of the window.
type chartInk struct {
	up, down, grid, text, mark color.Color
}

func currentInk() chartInk {
	return chartInk{
		up:   theme.Color(theme.ColorNameSuccess),
		down: theme.Color(theme.ColorNameError),
		grid: theme.Color(theme.ColorNameSeparator),
		text: theme.Color(theme.ColorNamePlaceHolder),
		mark: theme.Color(theme.ColorNamePrimary),
	}
}

func axisLabel(align fyne.TextAlign) *canvas.Text {
	text := canvas.NewText("", theme.Color(theme.ColorNamePlaceHolder))
	text.TextSize = axisTextSize
	text.Alignment = align
	return text
}

// timeLayout picks how much of a timestamp each tick has to spell out. It needs
// enough resolution to tell neighbouring ticks apart, which is what the spacing
// between them decides; and enough context to place them, which is what the
// span decides — a chart running over midnight has to name the day, or half its
// ticks read as the same hour.
func timeLayout(span, spacing time.Duration) string {
	const day = 24 * time.Hour

	switch {
	case spacing >= day && span >= 365*day:
		return "Jan 2006"
	case spacing >= day:
		return "02 Jan"
	case span >= day:
		return "02 Jan 15:04"
	case spacing >= time.Minute:
		return "15:04"
	default:
		return "15:04:05"
	}
}

// formatPrice prints a price with as many decimals as it needs to stay
// meaningful at its own size: cents for a coin quoted in thousands, and more of
// them as the price shrinks towards the fractions the smaller tokens trade at.
func formatPrice(price float64) string {
	return strconv.FormatFloat(price, 'f', priceDecimals(price), 64)
}

func priceDecimals(price float64) int {
	price = math.Abs(price)
	switch {
	case price >= 1000:
		return 2
	case price >= 1:
		return 4
	case price == 0:
		return 2
	default:
		// Below one, keep four significant digits: 0.00001234 rather than 0.00.
		return min(int(math.Ceil(-math.Log10(price)))+3, 10)
	}
}

// fit grows or shrinks a slice of drawing objects to n entries, keeping the
// ones already in it and calling build for any that are missing.
func fit[T any](items []T, n int, build func() T) []T {
	for len(items) < n {
		items = append(items, build())
	}
	return items[:n]
}

func clamp(v, low, high float32) float32 { return max(low, min(v, high)) }

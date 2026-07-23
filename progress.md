# TradeMan — Progress

Project version: **0.0.1**
Language: Go · Target platform: Linux (Arch, distro-agnostic tooling)
Data source: Binance API (candlesticks)
GUI toolkit: Fyne v2.8

Legend: `[ ]` not started · `[~]` in progress · `[x]` done

---

## 01 — Base UI implementation

| Status | Feature | Description |
|---|---|---|
| [x] | main screen | Header with app name `TradeMan` + version on the left, light/dark theme switcher on the right; tab section fills the rest of the window and starts empty. Keymap: `q` quit (with confirmation), `Ctrl-t` new tab named `No chart` and focus it, `Ctrl-w` close tab (with confirmation) then focus next tab, or previous if none. |
| [x] | tab interface | Each tab has a header row of three dropdowns — `Coin`, `Interval`, `No of candles` — with the chart area filling the remaining tab space below. |

## 02 — Themes implementation

| Status | Feature | Description |
|---|---|---|
| [x] | light / dark themes | Define color sets for light and dark themes, preferring system colors with sensible fallbacks; theme switcher on the main screen applies them, and its initial state follows the system theme (light if the system value is unavailable). |

Both color sets live in `internal/theming/palette.go` and cover every color Fyne
asks for, so nothing falls back to the default palette. The desktop's accent
color is read from the XDG portal, with the GNOME named accent as a fallback,
and becomes the primary color; it is nudged towards the background until it
clears 3:1 contrast, so an accent chosen for a light desktop still reads in dark
mode. The switcher pins the mode against the variant Fyne reports, so the
desktop changing its own scheme mid-session cannot override the user's choice.

## 03 — Tab functionality

| Status | Feature | Description |
|---|---|---|
| [x] | Dropdown lists | `Coin` holds the 20 most popular Binance coins; `Interval` holds 1s, 1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 6h, 8h, 12h, 1d, 3d, 1w, 1M; `No of candles` holds 100, 200, 300, 500. All three default to empty. |
| [x] | Live chart | Once all three dropdowns are non-empty, draw a live chart for the selected coin / interval / candle count; redraw on any dropdown change and refresh on the selected interval. |

The coin list is the 20 busiest USDT pairs by 24-hour turnover, read from the
exchange once and shared by every tab. Stablecoins, tokenised fiat and metals
and leveraged tokens are filtered out — they out-trade most real coins but hold
a flat line — and a fixed list of majors stands in when the exchange cannot be
reached. Market data comes from `data-api.binance.vision`, Binance's public
read-only host, which needs no API key and is not regionally blocked.

A complete selection starts a feed that redraws at the pace of its own interval,
clamped to between a second and a minute so a 1s chart cannot flood the API and
a daily one still shows its forming candle moving. Changing a dropdown retires
the running feed, and a reply that arrives after its feed was retired is
discarded rather than drawn over the new chart. Closing a tab stops its feed. A
failed refresh keeps the candles already on screen and marks them stale instead
of blanking the panel. Each tab renames itself after what it is charting.

The chart is a custom Fyne widget: a candle body and wick per interval, a price
scale, a time scale whose format follows how much time is on screen, and a
marker on the latest close. It draws in theme colours, so it follows the
light/dark switch, and reuses its shapes between redraws rather than
reallocating them every tick.

## 04 — Signal recognition

| Status | Feature | Description |
|---|---|---|
| [x] | Signal model | A signal is a body (a run of consecutive candles on one timeframe) plus one or more rules; each candle exposes open/close/high/low by a 1-based name — O1, C1, H1, L1, O2 … |
| [x] | Rule parser | Parse a boolean rule over body values: comparisons (`< > <= >= == !=`) between a reference or a number, combined with NOT/AND/OR and parentheses, rejecting out-of-range references and bad syntax by name and position. |
| [x] | Rule evaluation | Evaluate a rule against a body window, then slide the body across a candle series to collect matching windows — non-overlapping: after a match the search resumes at the candle next to the matched body. |
| [x] | Rule chaining | Check several rules in order, each on its own timeframe, starting each rule at the first candle after the previous match. |
| [x] | FVG signal (keys r / u) | Built-in FVG signal (body 3, rule `((L3 > H1) OR (L1 > H3))`); `r` starts a live signal on the focused chart at its current timeframe, re-running FVG on every change — new candle, timeframe, candle count or coin — so the circles always match what is drawn; `u` stops it and clears them. |

The engine lives in a new pure `internal/signal` package — a tokeniser, a
recursive-descent parser (OR → AND → NOT → comparison, with brackets), an
expression AST and an evaluator — with no Fyne dependency, so rules are
unit-tested on their own. `NewRule` parses a rule against a body length and
rejects a reference past the body; `Signal.Scan` reads each rule through a
`Source` keyed by interval, so a match on one timeframe hands off to the next
rule from the first candle after the matched body — the chaining the task asks
for. `FVG()` is the built-in single-rule signal, and `Signal.Marks` is the
single-interval shortcut the chart uses. In `internal/ui`, the candlestick
widget gained circle marks it draws in the accent colour (so they follow the
light/dark switch and are parked off-screen if a shorter chart no longer has the
candle), and the tab wires the `r` / `u` runes — canvas-level like `q`, so a
focused dropdown can't trigger them — to the focused chart, re-marking on every
refresh while the signal is active.

---

## Layout

| Path | Holds |
|---|---|
| `main.go` | Entry point and Fyne app metadata |
| `internal/version` | App name and version, kept in sync with `context.md` |
| `internal/theming` | Light/dark color sets, system scheme and accent detection, theme application |
| `internal/binance` | Market-data client: candlesticks, the coin ranking, interval spans |
| `internal/signal` | Signal engine: rule tokeniser and parser, expression AST and evaluator, body matching and cross-timeframe rule chaining, the built-in FVG signal |
| `internal/ui` | Main window, header, tab manager, tab contents, coin catalog, chart area, candlestick widget with signal marks, and the `r` / `u` signal keys |

Build and run: `go build ./...` then `go run .` · Tests: `go test ./...`

---

## Change log

| Date | Change |
|---|---|
| 2026-07-22 | Created `progress.md` from the `features/` directory. No code written yet. |
| 2026-07-22 | Section 01 implemented on Fyne v2.8: main screen (header, empty tab section, `q`/`Ctrl-t`/`Ctrl-w` keymap with confirmations) and the tab interface (three dropdowns above a chart area). System theme detection added via XDG portal, gsettings and GTK theme name, falling back to light. 21 tests added. |
| 2026-07-22 | Section 02 implemented: the project's own light and dark color sets covering every Fyne color name, desktop accent detection (XDG portal, GNOME named accents) feeding the primary color, and contrast helpers that keep text readable on any accent. `Apply` now installs this theme instead of wrapping Fyne's default. 13 tests added (34 in total), including WCAG contrast checks on both sets. |
| 2026-07-22 | Section 03 implemented: new `internal/binance` market-data client (candlesticks, turnover-ranked coin list, interval spans) and, in `internal/ui`, the shared coin catalog, the per-tab live feed and a custom candlestick chart widget. Dropdowns are populated and still start empty; a complete selection draws a live chart that redraws on the selected interval and on any dropdown change. 62 tests added (96 in total), all packages passing under `-race`. |
| 2026-07-23 | Section 04 (signal recognition) specified from `task.md`: added `features/04-signals.md` and the section above. Design planned — a pure `internal/signal` package (tokeniser, recursive-descent parser, AST, evaluator, rule chaining across per-rule timeframes), chart circle marks, and `r` / `u` keys running the built-in FVG signal on the focused chart. |
| 2026-07-23 | Section 04 implemented: new `internal/signal` package (rule tokeniser and recursive-descent parser, expression AST and evaluator, body matching, and cross-timeframe rule chaining through a `Source`, plus the built-in FVG signal) and, in `internal/ui`, circle marks on the candlestick widget and the `r` / `u` keys that run and clear FVG on the focused chart, re-marking on each refresh. 22 tests added (119 in total), all packages passing under `-race`. |
| 2026-07-23 | Fixed the `r` / `u` keys doing nothing on a live chart: a `widget.Select` grabs focus when used and then swallows plain runes, so the canvas never saw the keys. A dropdown change now hands focus back to the window (`returnFocus`), which also restores `q`. Added a signal note beside the chart's data line (`FVG signal · N marks`) so a genuine no-match is told apart from a dead key. 3 tests added (122 in total). |
| 2026-07-23 | Made signal matching non-overlapping: `Rule.Matches` now resumes at the candle next to a matched body instead of one candle on, so a single gap is one signal rather than one per overlapping window (the chained rules already began each next rule after the previous body). Tests updated. |
| 2026-07-23 | Made the FVG signal live: `r` starts it, and it re-runs over the candles on screen on every redraw, so the marks follow a new candle, a timeframe change, a candle-count change or a coin change; `signalNote` now names the timeframe beside the mark count (`FVG · 1h · N marks`). 3 end-to-end tests added (125 in total) driving those changes through a candle server. |

---

## Token & model usage

Cumulative across the project. Figures are approximate session estimates and
count the full context of every request, so they grow with conversation length
rather than with the amount of code produced.

| Date | Model | Input tokens | Output tokens | Total |
|---|---|---|---|---|
| 2026-07-22 | claude-opus-4-8 | ~14,000 | ~1,200 | ~15,200 |
| 2026-07-22 | claude-opus-4-8 | ~1,100,000 | ~19,000 | ~1,119,000 |
| 2026-07-22 | claude-opus-4-8 | ~700,000 | ~15,000 | ~715,000 |
| 2026-07-22 | claude-opus-4-8 | ~900,000 | ~34,000 | ~934,000 |
| 2026-07-23 | claude-opus-4-8 | ~60,000 | ~5,000 | ~65,000 |
| 2026-07-23 | claude-opus-4-8 | ~320,000 | ~18,000 | ~338,000 |
| 2026-07-23 | claude-opus-4-8 | ~200,000 | ~11,000 | ~211,000 |
| 2026-07-23 | claude-opus-4-8 | ~140,000 | ~8,000 | ~148,000 |
| 2026-07-23 | claude-opus-4-8 | ~170,000 | ~9,000 | ~179,000 |
| **Cumulative** | | **~3,604,000** | **~120,200** | **~3,724,200** |

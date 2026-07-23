# Signal recognition

[ ] Signal model
	* a signal marks a moment worth a trading decision on a chosen coin; it is a body plus one or more rules
	* body is a run of consecutive candles on one timeframe; each candle exposes four values named by a letter and a 1-based position: open Ox, close Cx, high Hx, low Lx (O1, C1, H1, L1, O2, C2 …)
[ ] Rule parser
	* a rule is a boolean expression over body values, combined with NOT, AND, OR and parentheses; its result is true or false
	* comparisons use `< > <= >= == !=` between two operands; an operand is a body reference (O/C/H/L + index) or a number
	* must parse and evaluate: `NOT (H1 < H2)` and `((H1 < H2) OR ((O1 < O2) AND (C1 > L2)))`
	* reject a rule that references a candle past the body length, an unknown field letter, or malformed syntax, naming the problem
[ ] Rule evaluation
	* evaluate a parsed rule against one body window, returning true or false
	* slide the body across a candle series and collect the windows where the rule holds
	* matches do not overlap: once a signal is found, resume the search at the candle next to its body, so one run of candles is one signal rather than several overlapping ones
[ ] Rule chaining
	* a signal may hold several rules checked in order; when a rule matches, the next rule starts from the first candle after the matched body
	* each rule carries its own timeframe (e.g. check rule 1 on 1h candles, then rule 2 on 5m); an empty timeframe means the current chart interval
[ ] FVG signal (keys r / u)
	* FVG is a built-in signal: body length 3, timeframe of the current chart, single rule `((L3 > H1) OR (L1 > H3))`
	* key `r`: start FVG on the focused chart at its current timeframe, circling each matching body
	* the signal is live — once started it re-runs over the candles on screen on every change: a new candle forming or arriving, a different timeframe, a different candle count, a different coin — so the marks always match what is drawn
	* key `u`: stop the signal and clear its marks
	* marks are drawn by the chart widget in the theme's accent colour and follow the light/dark switch like the rest of the chart

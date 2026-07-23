let's start to implement trading system.
first of all we need to implement system which can recognize a sygnal on a chart.  
signal is an indicator when user can make a trading decision. signal is applied to chosen coin.  
signal consists of:
	* body - apecific amount of consequetive candles (see about candles below)
	* set of rules, combined with parenthesis and logic operators (AND, OR, NOT). (see about rules below)
	
lets describe signal body. signal body is array of candles with specific timeframe. each candle is set of 4 numbers:
	* open price - Ox
	* close price - Cx
	* maximum price - Hx
	* minimum price - Lx
	where x is number of candle in row of candles from signal body.  
	
so in case when signal body length is 2, we have array like which [O1: val, C1: val, H1: val, L1 : val, O2: val, C2: val, H2: val, L2: val]. you can choose reperesantaion - but is think map or collection should be fine. take which is most effective for our task

lets descrive rule. rule if set of comparison of array body values, referenced by names (like O1, H2 etc), which are combined with logical operations like NOT, AND and OR and paranthesis. for example:
	* NOT (H1 < H2)
	* ((H1 < H2) OR ((O1 < O2) AND (C1 > L2)))

result of rule of boolean value true of false  

what i would like you to implement if feature when i can chain rules. if first rule is valid, we start to check next rule from the first candle which came after first rule body. it's important to notice that different rules could have differnet timeframse, for example for first rule i look to candles on 1 hour timeframe, but when rule is true i want to switch to 5 min timeframe for next rule

for example lets us FVG sygnal:
	* body has lenght of 3 and timeframe depends from current chart 
	* rule is simple: ((L3 > H1) OR (L1 > H3))

let's do it like this - if i press 'r' use FVG signal with current timeframe. check all current candles and mark when FVG rule is trule (just circle is enough for now), and then check when chart is updated for last candles set. if i press 'u' - clear signal marks.
     

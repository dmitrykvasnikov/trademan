# Base UI implementation
[ ] main screen
	* main screen contain header and tab section (the rest of application area)
	* header has on the left application name 'TradeMan' and version (get verstion from context.md) on the left side and light / dark theme switcher on the right side
	* state of theme switcher should be the same as system one. if you can not get info from system use light as default position
	* tab section should be list of tabs. when application starts there is no tabs at all.
	* keymap for application:
		** 'q' - quit the application, ask used for confirmation first.
		** 'Ctrl-t' - create new tab with name 'No chart' and focus on this tab
		** 'Ctrl-w' - close tab, ask user for confirmation first. focus on the next tab if there is any, or on prevoius tab.
[ ] tab interface
	* every tab has header part with 3 dropdown lists, placed in a row. names of those lists: 'Coin', 'Interval', 'No of candles'
	* chart area beneath it, which takes rest of tab area
	

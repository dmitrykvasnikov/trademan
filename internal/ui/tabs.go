package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"github.com/dmitrykvasnikov/trademan/internal/binance"
)

// newTabTitle is the name every tab gets until it shows a chart.
const newTabTitle = "No chart"

// tabManager owns the tab section of the main screen. It starts with no tabs at
// all; they are opened with Ctrl-T and closed with Ctrl-W.
type tabManager struct {
	docs   *container.DocTabs
	client *binance.Client
	coins  *coinCatalog

	// views keeps each tab's contents reachable, so its live feed can be shut
	// down when the tab closes.
	views map[*container.TabItem]*tabView
}

// newTabManager wires the tab bar's own close buttons to requestClose, so they
// ask for the same confirmation the keyboard shortcut does. Every tab it opens
// shares one exchange client and one coin catalog.
func newTabManager(client *binance.Client, coins *coinCatalog, requestClose func(*container.TabItem)) *tabManager {
	t := &tabManager{
		docs:   container.NewDocTabs(),
		client: client,
		coins:  coins,
		views:  map[*container.TabItem]*tabView{},
	}
	t.docs.CloseIntercept = requestClose
	return t
}

func (t *tabManager) view() fyne.CanvasObject {
	return t.docs
}

// open adds a fresh, chart-less tab and focuses it. The tab renames itself as
// its selection changes, so the bar says what each tab is watching rather than
// showing a row of identical "No chart" labels.
func (t *tabManager) open() {
	var tab *container.TabItem

	view := newTabView(t.client, t.coins, func(title string) {
		tab.Text = title
		t.docs.Refresh()
	})

	tab = container.NewTabItem(newTabTitle, view.view())
	t.views[tab] = view
	t.docs.Append(tab)
	t.docs.Select(tab)
}

// selected returns the focused tab, or nil while no tab is open.
func (t *tabManager) selected() *container.TabItem {
	if len(t.docs.Items) == 0 {
		return nil
	}
	return t.docs.Selected()
}

// close removes tab and focuses the one that takes its place: the next tab, or
// the previous one when the closed tab was the last.
func (t *tabManager) close(tab *container.TabItem) {
	index := t.indexOf(tab)
	if index < 0 {
		return
	}

	if view, ok := t.views[tab]; ok {
		view.close()
		delete(t.views, tab)
	}
	t.docs.Remove(tab)

	remaining := len(t.docs.Items)
	switch {
	case remaining == 0:
		// Nothing left to focus.
	case index < remaining:
		t.docs.SelectIndex(index)
	default:
		t.docs.SelectIndex(remaining - 1)
	}
}

func (t *tabManager) indexOf(tab *container.TabItem) int {
	for i, item := range t.docs.Items {
		if item == tab {
			return i
		}
	}
	return -1
}

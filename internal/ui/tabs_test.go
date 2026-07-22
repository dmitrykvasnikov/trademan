package ui

import (
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

// newTestTabs builds a tab manager with n open tabs. The close requests are
// recorded instead of confirmed, mirroring what the tab bar's own close button
// does before the user answers the dialog.
func newTestTabs(t *testing.T, n int) (*tabManager, *[]*container.TabItem) {
	t.Helper()
	test.NewTempApp(t)

	var requested []*container.TabItem
	tabs := newTabManager(offlineClient(), testCatalog(testSymbols), func(tab *container.TabItem) {
		requested = append(requested, tab)
	})
	for range n {
		tabs.open()
	}
	return tabs, &requested
}

func TestTabsStartEmpty(t *testing.T) {
	tabs, _ := newTestTabs(t, 0)

	if got := len(tabs.docs.Items); got != 0 {
		t.Errorf("new tab section has %d tabs, want 0", got)
	}
	if tab := tabs.selected(); tab != nil {
		t.Errorf("new tab section has %v selected, want nil", tab)
	}
}

func TestOpenFocusesNewTab(t *testing.T) {
	tabs, _ := newTestTabs(t, 2)

	if got := len(tabs.docs.Items); got != 2 {
		t.Fatalf("opened 2 tabs, tab section holds %d", got)
	}
	if got, want := tabs.selected(), tabs.docs.Items[1]; got != want {
		t.Errorf("selected tab is %v, want the one just opened %v", got, want)
	}
	if got := tabs.docs.Items[0].Text; got != newTabTitle {
		t.Errorf("new tab is named %q, want %q", got, newTabTitle)
	}
}

func TestCloseFocusesNextTab(t *testing.T) {
	tabs, _ := newTestTabs(t, 3)
	first, second, third := tabs.docs.Items[0], tabs.docs.Items[1], tabs.docs.Items[2]

	tabs.close(first)

	if got := tabs.selected(); got != second {
		t.Errorf("after closing the first tab, %v is selected, want the next tab %v", got, second)
	}
	if got, want := len(tabs.docs.Items), 2; got != want {
		t.Errorf("tab section holds %d tabs, want %d", got, want)
	}
	if tabs.indexOf(third) != 1 {
		t.Errorf("remaining tabs are out of order: %v", tabs.docs.Items)
	}
}

func TestCloseLastTabFocusesPreviousTab(t *testing.T) {
	tabs, _ := newTestTabs(t, 3)
	second, third := tabs.docs.Items[1], tabs.docs.Items[2]

	tabs.close(third)

	if got := tabs.selected(); got != second {
		t.Errorf("after closing the last tab, %v is selected, want the previous tab %v", got, second)
	}
}

func TestCloseOnlyTabLeavesNothingSelected(t *testing.T) {
	tabs, _ := newTestTabs(t, 1)

	tabs.close(tabs.docs.Items[0])

	if got := len(tabs.docs.Items); got != 0 {
		t.Fatalf("tab section holds %d tabs, want 0", got)
	}
	if tab := tabs.selected(); tab != nil {
		t.Errorf("empty tab section has %v selected, want nil", tab)
	}
}

func TestCloseIgnoresUnknownTab(t *testing.T) {
	tabs, _ := newTestTabs(t, 2)
	selected := tabs.selected()

	tabs.close(container.NewTabItem("stray", nil))

	if got := len(tabs.docs.Items); got != 2 {
		t.Errorf("closing an unknown tab changed the tab count to %d, want 2", got)
	}
	if got := tabs.selected(); got != selected {
		t.Errorf("closing an unknown tab moved focus to %v, want %v", got, selected)
	}
}

// Closing a tab has to shut down its feed as well; a chart nobody can see must
// not go on polling the exchange.
func TestCloseStopsTheTabsFeed(t *testing.T) {
	tabs, _ := newTestTabs(t, 1)
	tab := tabs.docs.Items[0]

	view, ok := tabs.views[tab]
	if !ok {
		t.Fatal("the tab manager lost track of the tab's contents")
	}
	stopped := false
	view.stop = func() { stopped = true }

	tabs.close(tab)

	if !stopped {
		t.Error("closing the tab left its feed running")
	}
	if _, held := tabs.views[tab]; held {
		t.Error("the closed tab's contents are still being held on to")
	}
}

// The tab bar's close buttons must ask the same question Ctrl-W asks rather
// than dropping the tab straight away.
func TestTabBarCloseButtonAsksFirst(t *testing.T) {
	tabs, requested := newTestTabs(t, 1)
	tab := tabs.docs.Items[0]

	tabs.docs.CloseIntercept(tab)

	if got := len(tabs.docs.Items); got != 1 {
		t.Errorf("tab was closed without confirmation, %d tabs left", got)
	}
	if len(*requested) != 1 || (*requested)[0] != tab {
		t.Errorf("close requests are %v, want exactly [%v]", *requested, tab)
	}
}

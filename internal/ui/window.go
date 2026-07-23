// Package ui builds the TradeMan main screen.
package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/dmitrykvasnikov/trademan/internal/binance"
	"github.com/dmitrykvasnikov/trademan/internal/theming"
	"github.com/dmitrykvasnikov/trademan/internal/version"
)

// Default window size, large enough for a readable chart on first launch.
const (
	windowWidth  = 1280
	windowHeight = 800
)

// MainWindow is the application's main screen: a header on top and the tab
// section filling the rest of the area.
type MainWindow struct {
	app    fyne.App
	win    fyne.Window
	header *header
	tabs   *tabManager

	// question is the confirmation currently on screen, or nil. Holding it
	// keeps repeated shortcut presses from stacking dialogs.
	question *dialog.ConfirmDialog
}

// New assembles the main window against the live exchange. One client and one
// coin catalog serve every tab, so opening a second tab costs nothing beyond
// its own candles.
func New(app fyne.App) *MainWindow {
	client := binance.New()
	return newWindow(app, client, newCoinCatalog(client))
}

// newWindow builds the window around a given exchange client and coin catalog,
// which is what lets the tests drive it without reaching the network. The theme
// switcher starts in the mode the desktop session reports, falling back to light
// when it reports nothing.
func newWindow(app fyne.App, client *binance.Client, coins *coinCatalog) *MainWindow {
	mode, _ := theming.DetectSystemMode()
	theming.Apply(app, mode)

	m := &MainWindow{app: app, win: app.NewWindow(version.Name)}
	m.header = newHeader(mode, m.applyMode)
	m.tabs = newTabManager(client, coins, m.confirmCloseTab)

	m.win.SetContent(container.NewBorder(m.header.view(), nil, nil, nil, m.tabs.view()))
	m.win.Resize(fyne.NewSize(windowWidth, windowHeight))
	m.win.CenterOnScreen()

	m.bindKeys()
	// Closing the window from the title bar asks the same question 'q' does.
	m.win.SetCloseIntercept(m.confirmQuit)

	return m
}

// ShowAndRun displays the main window and runs the event loop until quit.
func (m *MainWindow) ShowAndRun() {
	m.win.ShowAndRun()
}

func (m *MainWindow) applyMode(mode theming.Mode) {
	theming.Apply(m.app, mode)
}

// bindKeys installs the application keymap: 'q' quits, Ctrl-T opens a tab and
// Ctrl-W closes the focused one.
func (m *MainWindow) bindKeys() {
	canvas := m.win.Canvas()

	// Runes only reach the canvas when no widget holds focus, so typing into a
	// dropdown can never quit the application or fire a signal by accident.
	canvas.SetOnTypedRune(func(r rune) {
		switch r {
		case 'q', 'Q':
			m.confirmQuit()
		case 'r', 'R':
			if v := m.tabs.active(); v != nil {
				v.runSignal()
			}
		case 'u', 'U':
			if v := m.tabs.active(); v != nil {
				v.clearSignal()
			}
		}
	})

	canvas.AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyT, Modifier: fyne.KeyModifierControl},
		func(fyne.Shortcut) { m.tabs.open() },
	)
	canvas.AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: fyne.KeyModifierControl},
		func(fyne.Shortcut) { m.confirmCloseTab(m.tabs.selected()) },
	)
}

func (m *MainWindow) confirmQuit() {
	m.confirm("Quit "+version.Name, "Quit the application?", m.app.Quit)
}

// confirmCloseTab does nothing when no tab is open, so Ctrl-W on an empty tab
// section is a no-op rather than a pointless question.
func (m *MainWindow) confirmCloseTab(tab *container.TabItem) {
	if tab == nil {
		return
	}
	m.confirm("Close tab", fmt.Sprintf("Close tab %q?", tab.Text), func() { m.tabs.close(tab) })
}

// confirm asks one question at a time; presses arriving while a question is
// already on screen are ignored.
func (m *MainWindow) confirm(title, message string, onConfirm func()) {
	if m.question != nil {
		return
	}

	m.question = dialog.NewConfirm(title, message, func(confirmed bool) {
		m.question = nil
		if confirmed {
			onConfirm()
		}
	}, m.win)
	m.question.SetConfirmText("Yes")
	m.question.SetDismissText("No")
	m.question.Show()
}

package ui

import (
	"image/color"
	"reflect"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"github.com/dmitrykvasnikov/trademan/internal/theming"
)

func newTestWindow(t *testing.T) *MainWindow {
	t.Helper()
	return newWindow(test.NewTempApp(t), offlineClient(), testCatalog(testSymbols))
}

// press dispatches a Ctrl-modified shortcut the way the desktop driver does, so
// the tests exercise the real binding rather than the handler alone.
//
// Fyne's test canvas wraps the canvas that actually dispatches shortcuts in an
// interface that hides the entry point, so the wrapped value is unwrapped here.
// A failure means that wrapping changed, not that the keymap is broken — fix
// this helper before reading anything into the surrounding tests.
func press(t *testing.T, m *MainWindow, key fyne.KeyName) {
	t.Helper()

	dispatcher, ok := unwrapCanvas(m.win.Canvas()).(interface{ TypedShortcut(fyne.Shortcut) })
	if !ok {
		t.Fatalf("cannot dispatch shortcuts to Fyne's test canvas (%T)", m.win.Canvas())
	}
	dispatcher.TypedShortcut(&desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierControl})
}

// unwrapCanvas returns the canvas embedded in Fyne's test canvas, or c itself
// when there is nothing to unwrap.
func unwrapCanvas(c fyne.Canvas) any {
	value := reflect.ValueOf(c)
	if value.Kind() != reflect.Pointer || value.Elem().Kind() != reflect.Struct {
		return c
	}
	embedded := value.Elem().FieldByName("WindowlessCanvas")
	if !embedded.IsValid() || !embedded.CanInterface() {
		return c
	}
	return embedded.Interface()
}

// openDialogs counts the confirmation dialogs currently on screen.
func openDialogs(m *MainWindow) int {
	return len(m.win.Canvas().Overlays().List())
}

func TestMainWindowStartsWithNoTabs(t *testing.T) {
	m := newTestWindow(t)

	if got := len(m.tabs.docs.Items); got != 0 {
		t.Errorf("application starts with %d tabs, want 0", got)
	}
}

// The theme switcher must start in the mode the desktop reports, and Light when
// nothing reports anything.
func TestSwitcherStartsInSystemMode(t *testing.T) {
	m := newTestWindow(t)

	systemMode, _ := theming.DetectSystemMode()
	if m.header.mode != systemMode {
		t.Errorf("switcher starts in %v, want the system mode %v", m.header.mode, systemMode)
	}
	if got := m.header.switcher.Text; got != systemMode.String() {
		t.Errorf("switcher is labelled %q, want %q", got, systemMode.String())
	}
}

func TestSwitcherTogglesMode(t *testing.T) {
	m := newTestWindow(t)
	start := m.header.mode

	m.header.toggle()

	if m.header.mode != start.Toggle() {
		t.Errorf("switcher moved to %v, want %v", m.header.mode, start.Toggle())
	}
	if got := m.header.switcher.Text; got != start.Toggle().String() {
		t.Errorf("switcher is labelled %q, want %q", got, start.Toggle().String())
	}
}

// Flipping the switcher must repaint the application, not just relabel the
// button: the app theme has to become the palette of the new mode.
func TestSwitcherAppliesTheme(t *testing.T) {
	m := newTestWindow(t)

	background := func() color.Color {
		return m.app.Settings().Theme().Color(theme.ColorNameBackground, theme.VariantLight)
	}

	before := background()
	if want := theming.Theme(m.header.mode).Color(theme.ColorNameBackground, theme.VariantLight); before != want {
		t.Errorf("application starts on background %v, want the %v palette's %v", before, m.header.mode, want)
	}

	m.header.toggle()

	after := background()
	if want := theming.Theme(m.header.mode).Color(theme.ColorNameBackground, theme.VariantLight); after != want {
		t.Errorf("after switching to %v the background is %v, want %v", m.header.mode, after, want)
	}
	if before == after {
		t.Errorf("switching to %v left the background at %v", m.header.mode, after)
	}
}

func TestNewTabShortcutOpensAndFocusesTab(t *testing.T) {
	m := newTestWindow(t)

	press(t, m, fyne.KeyT)

	if got := len(m.tabs.docs.Items); got != 1 {
		t.Fatalf("Ctrl-T left %d tabs open, want 1", got)
	}
	if got := m.tabs.selected(); got != m.tabs.docs.Items[0] {
		t.Errorf("Ctrl-T selected %v, want the tab it opened", got)
	}
	if got := m.tabs.docs.Items[0].Text; got != newTabTitle {
		t.Errorf("Ctrl-T named the tab %q, want %q", got, newTabTitle)
	}
}

func TestCloseTabShortcutAsksFirst(t *testing.T) {
	m := newTestWindow(t)
	press(t, m, fyne.KeyT)

	press(t, m, fyne.KeyW)

	if got := len(m.tabs.docs.Items); got != 1 {
		t.Errorf("Ctrl-T closed the tab without asking, %d tabs left", got)
	}
	if openDialogs(m) != 1 {
		t.Errorf("Ctrl-W raised %d dialogs, want 1 confirmation", openDialogs(m))
	}
}

// Ctrl-W with an empty tab section is a no-op, not a question about nothing.
func TestCloseTabShortcutWithoutTabsAsksNothing(t *testing.T) {
	m := newTestWindow(t)

	press(t, m, fyne.KeyW)

	if m.question != nil {
		t.Error("Ctrl-W asked to close a tab while no tab was open")
	}
	if got := openDialogs(m); got != 0 {
		t.Errorf("Ctrl-W raised %d dialogs with no tabs open, want 0", got)
	}
}

// Holding the shortcut down must not pile dialogs on top of each other.
func TestRepeatedShortcutRaisesOneDialog(t *testing.T) {
	m := newTestWindow(t)
	press(t, m, fyne.KeyT)

	press(t, m, fyne.KeyW)
	press(t, m, fyne.KeyW)
	press(t, m, fyne.KeyW)

	if got := openDialogs(m); got != 1 {
		t.Errorf("three Ctrl-W presses raised %d dialogs, want 1", got)
	}
}

func TestQuitKeyAsksFirst(t *testing.T) {
	m := newTestWindow(t)

	test.TypeOnCanvas(m.win.Canvas(), "q")

	if m.question == nil {
		t.Error("'q' did not ask for confirmation")
	}
	if got := openDialogs(m); got != 1 {
		t.Errorf("'q' raised %d dialogs, want 1 confirmation", got)
	}
}

// There is no test for 'q' being swallowed by a focused widget: the desktop
// driver decides that, and test.TypeOnCanvas calls the canvas handler directly
// instead of routing through focus, so it could not tell the two apart.

// Answering "No" must clear the guard so the next press asks again.
func TestDeclinedDialogAllowsAskingAgain(t *testing.T) {
	m := newTestWindow(t)
	press(t, m, fyne.KeyT)

	press(t, m, fyne.KeyW)
	if m.question == nil {
		t.Fatal("Ctrl-W did not ask for confirmation")
	}
	m.question.Hide() // answer "No"

	press(t, m, fyne.KeyW)
	if m.question == nil {
		t.Error("Ctrl-W did not ask again after the first question was declined")
	}
	if got := len(m.tabs.docs.Items); got != 1 {
		t.Errorf("declining the question still closed the tab, %d tabs left", got)
	}
}

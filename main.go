// Command trademan is a GUI trading assistant: each tab shows a live
// candlestick chart for one coin, marked up with the signals a trader acts on.
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/dmitrykvasnikov/trademan/internal/ui"
	"github.com/dmitrykvasnikov/trademan/internal/version"
)

// appID identifies the application to the desktop session and to Fyne's
// preference storage.
const appID = "com.github.dmitrykvasnikov.trademan"

func main() {
	// Set in code rather than left to FyneApp.toml, which is only read for
	// development builds and only when it sits next to the executable.
	app.SetMetadata(fyne.AppMetadata{
		ID:      appID,
		Name:    version.Name,
		Version: version.Version,
		Build:   1,
		// The UI is driven entirely from the main goroutine, so the current
		// threading model is already satisfied.
		Migrations: map[string]bool{"fyneDo": true},
	})

	ui.New(app.NewWithID(appID)).ShowAndRun()
}

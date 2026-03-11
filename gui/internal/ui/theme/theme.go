// Package theme provides theme management utilities.
package theme

import "github.com/diamondburned/gotk4-adwaita/pkg/adw"

// Apply applies dark or light theme to the application
func Apply(app *adw.Application, darkMode bool) {
	if darkMode {
		app.StyleManager().SetColorScheme(adw.ColorSchemeForceDark)
	} else {
		app.StyleManager().SetColorScheme(adw.ColorSchemeForceLight)
	}
}

// Konrul - Theme support
// Different color schemes for the UI

package main

import (
	ui "github.com/gizak/termui/v3"
)

// Theme represents a color theme for the UI
type Theme struct {
	Name string

	// Gauge colors
	CPUColor    ui.Color
	MemoryColor ui.Color
	SwapColor   ui.Color

	// Border and text colors
	BorderColor    ui.Color
	TitleColor     ui.Color
	TextColor      ui.Color
	LabelColor     ui.Color

	// Selection colors
	SelectionFg ui.Color
	SelectionBg ui.Color

	// Header colors
	HeaderFg ui.Color
	HeaderBg ui.Color

	// Bar chart colors
	BarColors []ui.Color
}

// Available themes
var themes = map[string]Theme{
	"default": {
		Name:           "default",
		CPUColor:       ui.ColorGreen,
		MemoryColor:    ui.ColorYellow,
		SwapColor:      ui.ColorMagenta,
		BorderColor:    ui.ColorCyan,
		TitleColor:     ui.ColorCyan,
		TextColor:      ui.ColorWhite,
		LabelColor:     ui.ColorCyan,
		SelectionFg:    ui.ColorBlack,
		SelectionBg:    ui.ColorCyan,
		HeaderFg:       ui.ColorYellow,
		HeaderBg:       ui.ColorClear,
		BarColors:      []ui.Color{ui.ColorGreen, ui.ColorYellow, ui.ColorRed, ui.ColorCyan, ui.ColorMagenta, ui.ColorBlue, ui.ColorWhite},
	},
	"dark": {
		Name:           "dark",
		CPUColor:       ui.ColorBlue,
		MemoryColor:    ui.ColorMagenta,
		SwapColor:      ui.ColorCyan,
		BorderColor:    ui.ColorWhite,
		TitleColor:     ui.ColorWhite,
		TextColor:      ui.ColorWhite,
		LabelColor:     ui.ColorWhite,
		SelectionFg:    ui.ColorBlack,
		SelectionBg:    ui.ColorWhite,
		HeaderFg:       ui.ColorWhite,
		HeaderBg:       ui.ColorClear,
		BarColors:      []ui.Color{ui.ColorBlue, ui.ColorMagenta, ui.ColorCyan, ui.ColorWhite},
	},
	"light": {
		Name:           "light",
		CPUColor:       ui.ColorGreen,
		MemoryColor:    ui.ColorBlue,
		SwapColor:      ui.ColorRed,
		BorderColor:    ui.ColorBlack,
		TitleColor:     ui.ColorBlack,
		TextColor:      ui.ColorBlack,
		LabelColor:     ui.ColorBlack,
		SelectionFg:    ui.ColorWhite,
		SelectionBg:    ui.ColorBlue,
		HeaderFg:       ui.ColorBlue,
		HeaderBg:       ui.ColorClear,
		BarColors:      []ui.Color{ui.ColorGreen, ui.ColorBlue, ui.ColorRed, ui.ColorYellow},
	},
	"monokai": {
		Name:           "monokai",
		CPUColor:       ui.ColorGreen,
		MemoryColor:    ui.ColorYellow,
		SwapColor:      ui.ColorMagenta,
		BorderColor:    ui.ColorYellow,
		TitleColor:     ui.ColorYellow,
		TextColor:      ui.ColorWhite,
		LabelColor:     ui.ColorYellow,
		SelectionFg:    ui.ColorBlack,
		SelectionBg:    ui.ColorYellow,
		HeaderFg:       ui.ColorMagenta,
		HeaderBg:       ui.ColorClear,
		BarColors:      []ui.Color{ui.ColorGreen, ui.ColorYellow, ui.ColorMagenta, ui.ColorCyan, ui.ColorRed},
	},
}

// currentTheme holds the active theme
var currentTheme = themes["default"]

// GetTheme returns a theme by name, defaults to "default" if not found
func GetTheme(name string) Theme {
	if theme, ok := themes[name]; ok {
		return theme
	}
	return themes["default"]
}

// SetTheme sets the current theme by name
func SetTheme(name string) {
	currentTheme = GetTheme(name)
}

// GetCurrentTheme returns the current active theme
func GetCurrentTheme() Theme {
	return currentTheme
}

// GetThemeNames returns a list of available theme names
func GetThemeNames() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	return names
}

// NextTheme cycles to the next theme and returns its name
func NextTheme() string {
	themeNames := []string{"default", "dark", "light", "monokai"}
	currentName := currentTheme.Name

	for i, name := range themeNames {
		if name == currentName {
			nextIndex := (i + 1) % len(themeNames)
			SetTheme(themeNames[nextIndex])
			return themeNames[nextIndex]
		}
	}

	SetTheme("default")
	return "default"
}

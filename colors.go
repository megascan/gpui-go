package gpui

// Colors is the default set of colors for gpui, used for styling base components, examples and more.
type Colors struct {
	// Text is the text color.
	Text Rgba
	// SelectedText is the selected text color.
	SelectedText Rgba
	// Background is the background color.
	Background Rgba
	// Disabled is the disabled color.
	Disabled Rgba
	// Selected is the selected color.
	Selected Rgba
	// Border is the border color.
	Border Rgba
	// Separator is the separator color.
	Separator Rgba
	// Container is the container color.
	Container Rgba
}

// ColorsForAppearance returns the default colors for the window's appearance.
func ColorsForAppearance(window *Window) Colors {
	switch window.Appearance() {
	case WindowAppearanceDark, WindowAppearanceVibrantDark:
		return ColorsDark()
	default:
		return ColorsLight()
	}
}

// ColorsDark returns the default dark colors.
func ColorsDark() Colors {
	return Colors{
		Text:         Rgb(0xffffff),
		SelectedText: Rgb(0xffffff),
		Disabled:     Rgb(0x565656),
		Selected:     Rgb(0x2457ca),
		Background:   Rgb(0x222222),
		Border:       Rgb(0x000000),
		Separator:    Rgb(0xd9d9d9),
		Container:    Rgb(0x262626),
	}
}

// ColorsLight returns the default light colors; they are also upstream's Default.
func ColorsLight() Colors {
	return Colors{
		Text:         Rgb(0x252525),
		SelectedText: Rgb(0xffffff),
		Background:   Rgb(0xffffff),
		Disabled:     Rgb(0xb0b0b0),
		Selected:     Rgb(0x2a63d9),
		Border:       Rgb(0xd9d9d9),
		Separator:    Rgb(0xe6e6e6),
		Container:    Rgb(0xf4f5f5),
	}
}

// ColorsGetGlobal returns the Colors of the GlobalColors global.
func ColorsGetGlobal(cx *App) *Colors {
	return cx.Global[GlobalColors]().Colors
}

// GlobalColors is the global holding the default Colors; it embeds them (upstream's Deref).
type GlobalColors struct {
	*Colors
}

// DefaultColors gives access to the global default Colors.
type DefaultColors interface {
	// DefaultColors returns the default Colors.
	DefaultColors() *Colors
}

// DefaultColors returns the Colors of the GlobalColors global.
func (cx *App) DefaultColors() *Colors {
	return cx.Global[GlobalColors]().Colors
}

// DefaultAppearance is the appearance of the base GPUI colors, following the system's WindowAppearance.
type DefaultAppearance uint8

const (
	// DefaultAppearanceLight uses the colors for light appearances.
	DefaultAppearanceLight DefaultAppearance = iota
	// DefaultAppearanceDark uses the colors for dark appearances.
	DefaultAppearanceDark
)

// ToDefaultAppearance returns the DefaultAppearance for the window appearance.
func (a WindowAppearance) ToDefaultAppearance() DefaultAppearance {
	switch a {
	case WindowAppearanceDark, WindowAppearanceVibrantDark:
		return DefaultAppearanceDark
	default:
		return DefaultAppearanceLight
	}
}
